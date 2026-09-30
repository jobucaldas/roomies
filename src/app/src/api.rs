use crate::models::*;
use crate::storage;
use reqwest::{Client, Method};
use serde::de::DeserializeOwned;
use serde::Serialize;
use std::fmt;
use std::sync::{Arc, Mutex};

pub fn api_base_url() -> String {
    #[cfg(target_arch = "wasm32")]
    let origin = web_sys::window().and_then(|window| window.location().origin().ok());
    #[cfg(not(target_arch = "wasm32"))]
    let origin = None;
    resolve_api_base_url(
        option_env!("ROOMIES_API_URL"),
        cfg!(target_arch = "wasm32"),
        cfg!(feature = "mobile"),
        origin,
    )
}

fn resolve_api_base_url(
    configured: Option<&str>,
    is_web: bool,
    is_mobile: bool,
    origin: Option<String>,
) -> String {
    let fallback = if is_web {
        "/api"
    } else if is_mobile {
        "http://10.0.2.2:8080/api"
    } else {
        "http://localhost:8080/api"
    };
    let configured = configured.unwrap_or(fallback);
    if is_web && configured.starts_with('/') {
        if let Some(origin) = origin {
            return format!("{}{}", origin.trim_end_matches('/'), configured)
                .trim_end_matches('/')
                .to_string();
        }
    }
    configured.trim_end_matches('/').to_string()
}

#[derive(Debug, Clone, PartialEq)]
pub enum ApiError {
    Http { status: u16, message: String },
    Transport(String),
    Decode(String),
    SessionStorage(storage::StorageError),
}
impl fmt::Display for ApiError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Http { message, .. } | Self::Transport(message) | Self::Decode(message) => {
                f.write_str(message)
            }
            Self::SessionStorage(error) => error.fmt(f),
        }
    }
}
impl std::error::Error for ApiError {}

#[derive(Default)]
struct SessionState {
    token: Option<String>,
    generation: u64,
    valid: bool,
}

struct SessionRequest {
    builder: reqwest::RequestBuilder,
    generation: Option<u64>,
}

impl SessionRequest {
    fn json<B: Serialize>(mut self, body: &B) -> Self {
        self.builder = self.builder.json(body);
        self
    }
}

#[derive(Clone)]
pub struct ApiClient {
    session: Arc<Mutex<SessionState>>,
    transition: Arc<Mutex<()>>,
    pub base_url: String,
    client: Client,
    session_store: Arc<dyn storage::SessionTokenStore>,
    storage_error: Arc<Mutex<Option<storage::StorageError>>>,
}
impl Default for ApiClient {
    fn default() -> Self {
        Self::new()
    }
}
impl ApiClient {
    pub fn new() -> Self {
        let store = storage::production_store();
        let client = Self::with_base_url_and_store(api_base_url(), store);
        match client.session_store.load() {
            Ok(token) => {
                let mut session = client.session.lock().expect("session state lock poisoned");
                session.valid = token.is_some();
                session.token = token;
            }
            Err(error) => client.remember_storage_error(error),
        }
        client
    }
    pub fn with_base_url(base_url: impl Into<String>) -> Self {
        Self::with_base_url_and_store(base_url, storage::production_store())
    }
    fn with_base_url_and_store(
        base_url: impl Into<String>,
        session_store: Arc<dyn storage::SessionTokenStore>,
    ) -> Self {
        Self {
            session: Arc::new(Mutex::new(SessionState::default())),
            transition: Arc::new(Mutex::new(())),
            base_url: base_url.into().trim_end_matches('/').to_string(),
            client: Client::new(),
            session_store,
            storage_error: Arc::new(Mutex::new(None)),
        }
    }
    pub fn has_saved_token(&self) -> bool {
        self.session
            .lock()
            .expect("session state lock poisoned")
            .token
            .is_some()
    }
    pub fn storage_error(&self) -> Option<storage::StorageError> {
        *self
            .storage_error
            .lock()
            .expect("storage error lock poisoned")
    }
    pub fn is_authenticated(&self) -> bool {
        let session = self.session.lock().expect("session state lock poisoned");
        session.token.is_some() && session.valid
    }
    fn set_token(&self, token: String) -> Result<(), ApiError> {
        let _transition = self
            .transition
            .lock()
            .expect("session transition lock poisoned");
        self.session_store
            .save(&token)
            .map_err(ApiError::SessionStorage)?;
        let mut session = self.session.lock().expect("session state lock poisoned");
        session.generation = session.generation.wrapping_add(1);
        session.token = Some(token);
        session.valid = true;
        *self
            .storage_error
            .lock()
            .expect("storage error lock poisoned") = None;
        Ok(())
    }
    pub fn invalidate_session(&self) {
        self.session
            .lock()
            .expect("session state lock poisoned")
            .valid = false;
    }
    fn remember_storage_error(&self, error: storage::StorageError) {
        *self
            .storage_error
            .lock()
            .expect("storage error lock poisoned") = Some(error);
    }
    fn clear_persisted_session(
        &self,
        expected_generation: Option<u64>,
    ) -> Result<(), storage::StorageError> {
        let _transition = self
            .transition
            .lock()
            .expect("session transition lock poisoned");
        {
            let mut session = self.session.lock().expect("session state lock poisoned");
            if expected_generation.is_some_and(|generation| generation != session.generation) {
                return Ok(());
            }
            session.generation = session.generation.wrapping_add(1);
            session.token = None;
            session.valid = false;
        }
        let result = self.session_store.clear();
        match result {
            Ok(()) => {
                *self
                    .storage_error
                    .lock()
                    .expect("storage error lock poisoned") = None;
            }
            Err(error) => self.remember_storage_error(error),
        }
        result
    }
    fn handle_unauthorized(&self, generation: Option<u64>) {
        if let Some(generation) = generation {
            let _ = self.clear_persisted_session(Some(generation));
        }
    }
    fn request(&self, method: Method, path: &str) -> SessionRequest {
        let snapshot = {
            let session = self.session.lock().expect("session state lock poisoned");
            session
                .valid
                .then(|| {
                    session
                        .token
                        .clone()
                        .map(|token| (token, session.generation))
                })
                .flatten()
        };
        let builder = self
            .client
            .request(method, format!("{}{}", self.base_url, path))
            .header("Content-Type", "application/json");
        match snapshot {
            Some((token, generation)) => SessionRequest {
                builder: builder.bearer_auth(token),
                generation: Some(generation),
            },
            None => SessionRequest {
                builder,
                generation: None,
            },
        }
    }
    async fn send<T: DeserializeOwned>(&self, request: SessionRequest) -> Result<T, ApiError> {
        let generation = request.generation;
        let response = request
            .builder
            .send()
            .await
            .map_err(|e| ApiError::Transport(e.to_string()))?;
        let status = response.status();
        if !status.is_success() {
            if status.as_u16() == 401 {
                self.handle_unauthorized(generation);
            }
            return Err(ApiError::Http {
                status: status.as_u16(),
                message: response
                    .json::<ErrorResponse>()
                    .await
                    .map(|e| e.error)
                    .unwrap_or_else(|_| status.to_string()),
            });
        }
        response
            .json()
            .await
            .map_err(|e| ApiError::Decode(e.to_string()))
    }
    async fn empty(&self, request: SessionRequest) -> Result<(), ApiError> {
        let generation = request.generation;
        let response = request
            .builder
            .send()
            .await
            .map_err(|e| ApiError::Transport(e.to_string()))?;
        if response.status().is_success() {
            response
                .bytes()
                .await
                .map_err(|e| ApiError::Transport(e.to_string()))?;
            Ok(())
        } else {
            if response.status().as_u16() == 401 {
                self.handle_unauthorized(generation);
            }
            Err(ApiError::Http {
                status: response.status().as_u16(),
                message: response
                    .text()
                    .await
                    .unwrap_or_else(|_| "request failed".into()),
            })
        }
    }
    async fn json<T: DeserializeOwned, B: Serialize>(
        &self,
        method: Method,
        path: &str,
        body: &B,
    ) -> Result<T, ApiError> {
        self.send(self.request(method, path).json(body)).await
    }
    pub async fn register(
        &mut self,
        name: &str,
        email: &str,
        password: &str,
    ) -> Result<AuthResponse, ApiError> {
        let a: AuthResponse = self
            .json(
                Method::POST,
                "/auth/register",
                &serde_json::json!({"name":name,"email":email,"password":password}),
            )
            .await?;
        self.set_token(a.token.clone())?;
        Ok(a)
    }
    pub async fn login(&mut self, email: &str, password: &str) -> Result<AuthResponse, ApiError> {
        let a: AuthResponse = self
            .json(
                Method::POST,
                "/auth/login",
                &serde_json::json!({"email":email,"password":password}),
            )
            .await?;
        self.set_token(a.token.clone())?;
        Ok(a)
    }
    pub async fn me(&self) -> Result<User, ApiError> {
        self.send(self.request(Method::GET, "/auth/me")).await
    }
    pub fn logout(&self) -> Result<(), storage::StorageError> {
        self.clear_persisted_session(None)
    }
    pub async fn get_houses(&self) -> Result<Vec<House>, ApiError> {
        self.send(self.request(Method::GET, "/houses")).await
    }
    pub async fn get_house(&self, id: &str) -> Result<House, ApiError> {
        self.send(self.request(Method::GET, &format!("/houses/{id}")))
            .await
    }
    pub async fn create_house(&self, name: &str) -> Result<House, ApiError> {
        self.json(Method::POST, "/houses", &serde_json::json!({"name":name}))
            .await
    }
    pub async fn update_house(&self, id: &str, name: &str) -> Result<House, ApiError> {
        self.json(
            Method::PUT,
            &format!("/houses/{id}"),
            &serde_json::json!({"name":name}),
        )
        .await
    }
    pub async fn get_members(&self, id: &str) -> Result<Vec<HouseMember>, ApiError> {
        self.send(self.request(Method::GET, &format!("/houses/{id}/members")))
            .await
    }
    pub async fn add_member(
        &self,
        id: &str,
        user_id: &str,
        role: &str,
    ) -> Result<HouseMember, ApiError> {
        self.json(
            Method::POST,
            &format!("/houses/{id}/members"),
            &serde_json::json!({"user_id":user_id,"role":role}),
        )
        .await
    }
    pub async fn update_member_role(
        &self,
        id: &str,
        user_id: &str,
        role: &str,
    ) -> Result<MessageResponse, ApiError> {
        self.json(
            Method::PUT,
            &format!("/houses/{id}/members/{user_id}"),
            &serde_json::json!({"role":role}),
        )
        .await
    }
    pub async fn remove_member(&self, id: &str, user_id: &str) -> Result<(), ApiError> {
        self.empty(self.request(Method::DELETE, &format!("/houses/{id}/members/{user_id}")))
            .await
    }
    pub async fn create_invitation(
        &self,
        house_id: &str,
        email: &str,
        role: &str,
    ) -> Result<HouseInvitation, ApiError> {
        self.json(
            Method::POST,
            &format!("/houses/{house_id}/invites"),
            &CreateInvitationRequest {
                email: email.into(),
                role: role.into(),
            },
        )
        .await
    }
    pub async fn list_invitations(&self, house_id: &str) -> Result<Vec<HouseInvitation>, ApiError> {
        self.send(self.request(Method::GET, &format!("/houses/{house_id}/invites")))
            .await
    }
    pub async fn revoke_invitation(
        &self,
        house_id: &str,
        invitation_id: &str,
    ) -> Result<HouseInvitation, ApiError> {
        self.send(self.request(
            Method::DELETE,
            &format!("/houses/{house_id}/invites/{invitation_id}"),
        ))
        .await
    }
    pub async fn accept_invitation(
        &self,
        token: &str,
    ) -> Result<InvitationAcceptanceResponse, ApiError> {
        self.json(
            Method::POST,
            "/invitations/accept",
            &AcceptInvitationRequest {
                token: token.into(),
            },
        )
        .await
    }
    pub async fn get_expenses(&self, id: &str) -> Result<Vec<Expense>, ApiError> {
        self.send(self.request(Method::GET, &format!("/houses/{id}/expenses")))
            .await
    }
    pub async fn get_expense(&self, h: &str, e: &str) -> Result<ExpenseDetail, ApiError> {
        self.send(self.request(Method::GET, &format!("/houses/{h}/expenses/{e}")))
            .await
    }
    pub async fn create_expense(
        &self,
        id: &str,
        req: &CreateExpenseRequest,
    ) -> Result<Expense, ApiError> {
        self.json(Method::POST, &format!("/houses/{id}/expenses"), req)
            .await
    }
    pub async fn update_expense(
        &self,
        h: &str,
        e: &str,
        req: &UpdateExpenseRequest,
    ) -> Result<Expense, ApiError> {
        self.json(Method::PUT, &format!("/houses/{h}/expenses/{e}"), req)
            .await
    }
    pub async fn delete_expense(&self, h: &str, e: &str) -> Result<(), ApiError> {
        self.empty(self.request(Method::DELETE, &format!("/houses/{h}/expenses/{e}")))
            .await
    }
    pub async fn set_visibility(
        &self,
        h: &str,
        e: &str,
        visibility: &str,
        visible_to: &[String],
    ) -> Result<(), ApiError> {
        self.empty(
            self.request(
                Method::POST,
                &format!("/houses/{h}/expenses/{e}/visibility"),
            )
            .json(&SetVisibilityRequest {
                visibility: visibility.to_owned(),
                visible_to: visible_to.to_owned(),
            }),
        )
        .await
    }
    pub async fn get_notes(&self, id: &str) -> Result<Vec<Note>, ApiError> {
        self.send(self.request(Method::GET, &format!("/houses/{id}/notes")))
            .await
    }
    pub async fn get_note(&self, h: &str, n: &str) -> Result<Note, ApiError> {
        self.send(self.request(Method::GET, &format!("/houses/{h}/notes/{n}")))
            .await
    }
    pub async fn create_note(&self, h: &str, title: &str, content: &str) -> Result<Note, ApiError> {
        self.json(
            Method::POST,
            &format!("/houses/{h}/notes"),
            &CreateNoteRequest {
                title: title.into(),
                content: content.into(),
            },
        )
        .await
    }
    pub async fn update_note(
        &self,
        h: &str,
        n: &str,
        req: &UpdateNoteRequest,
    ) -> Result<Note, ApiError> {
        self.json(Method::PUT, &format!("/houses/{h}/notes/{n}"), req)
            .await
    }
    pub async fn delete_note(&self, h: &str, n: &str) -> Result<(), ApiError> {
        self.empty(self.request(Method::DELETE, &format!("/houses/{h}/notes/{n}")))
            .await
    }
    pub async fn get_balances(&self, id: &str) -> Result<BalanceResponse, ApiError> {
        self.send(self.request(Method::GET, &format!("/houses/{id}/balances")))
            .await
    }
    pub async fn get_notification_preferences(
        &self,
        id: &str,
    ) -> Result<NotificationPreferences, ApiError> {
        self.send(self.request(
            Method::GET,
            &format!("/houses/{id}/notification-preferences"),
        ))
        .await
    }
    pub async fn put_notification_preferences(
        &self,
        id: &str,
        value: &NotificationPreferences,
    ) -> Result<NotificationPreferences, ApiError> {
        self.json(
            Method::PUT,
            &format!("/houses/{id}/notification-preferences"),
            value,
        )
        .await
    }
    pub async fn get_notification_subscriptions(
        &self,
        id: &str,
    ) -> Result<Vec<NotificationSubscription>, ApiError> {
        self.send(self.request(
            Method::GET,
            &format!("/houses/{id}/notification-subscriptions"),
        ))
        .await
    }
    pub async fn create_notification_subscription(
        &self,
        id: &str,
        value: &CreateNotificationSubscriptionRequest,
    ) -> Result<NotificationSubscription, ApiError> {
        self.json(
            Method::POST,
            &format!("/houses/{id}/notification-subscriptions"),
            value,
        )
        .await
    }
    pub async fn delete_notification_subscription(
        &self,
        house_id: &str,
        subscription_id: &str,
    ) -> Result<(), ApiError> {
        self.empty(self.request(
            Method::DELETE,
            &format!("/houses/{house_id}/notification-subscriptions/{subscription_id}"),
        ))
        .await
    }
    pub async fn get_scheduled_events(
        &self,
        id: &str,
    ) -> Result<Vec<ScheduledHouseEvent>, ApiError> {
        self.send(self.request(Method::GET, &format!("/houses/{id}/scheduled-events")))
            .await
    }
    pub async fn create_scheduled_event(
        &self,
        id: &str,
        value: &ScheduledEventRequest,
    ) -> Result<ScheduledHouseEvent, ApiError> {
        self.json(
            Method::POST,
            &format!("/houses/{id}/scheduled-events"),
            value,
        )
        .await
    }
    pub async fn update_scheduled_event(
        &self,
        house_id: &str,
        event_id: &str,
        value: &ScheduledEventRequest,
    ) -> Result<ScheduledHouseEvent, ApiError> {
        self.json(
            Method::PUT,
            &format!("/houses/{house_id}/scheduled-events/{event_id}"),
            value,
        )
        .await
    }
    pub async fn delete_scheduled_event(
        &self,
        house_id: &str,
        event_id: &str,
    ) -> Result<(), ApiError> {
        self.empty(self.request(
            Method::DELETE,
            &format!("/houses/{house_id}/scheduled-events/{event_id}"),
        ))
        .await
    }
    pub async fn get_groceries(&self, house_id: &str) -> Result<Vec<GroceryItem>, ApiError> {
        self.send(self.request(Method::GET, &format!("/houses/{house_id}/groceries")))
            .await
    }
    pub async fn create_grocery(
        &self,
        house_id: &str,
        value: &GroceryRequest,
    ) -> Result<GroceryItem, ApiError> {
        self.json(
            Method::POST,
            &format!("/houses/{house_id}/groceries"),
            value,
        )
        .await
    }
    pub async fn update_grocery(
        &self,
        house_id: &str,
        id: &str,
        value: &GroceryRequest,
    ) -> Result<GroceryItem, ApiError> {
        self.json(
            Method::PUT,
            &format!("/houses/{house_id}/groceries/{id}"),
            value,
        )
        .await
    }
    pub async fn toggle_grocery(
        &self,
        house_id: &str,
        id: &str,
        value: &GroceryToggleRequest,
    ) -> Result<GroceryItem, ApiError> {
        self.json(
            Method::POST,
            &format!("/houses/{house_id}/groceries/{id}/toggle"),
            value,
        )
        .await
    }
    pub async fn delete_grocery(
        &self,
        house_id: &str,
        id: &str,
        version: i64,
    ) -> Result<(), ApiError> {
        self.empty(self.request(
            Method::DELETE,
            &format!("/houses/{house_id}/groceries/{id}?version={version}"),
        ))
        .await
    }
    pub async fn get_chores(&self, house_id: &str) -> Result<Vec<Chore>, ApiError> {
        self.send(self.request(Method::GET, &format!("/houses/{house_id}/chores")))
            .await
    }
    pub async fn create_chore(
        &self,
        house_id: &str,
        value: &ChoreRequest,
    ) -> Result<Chore, ApiError> {
        self.json(Method::POST, &format!("/houses/{house_id}/chores"), value)
            .await
    }
    pub async fn update_chore(
        &self,
        house_id: &str,
        id: &str,
        value: &ChoreRequest,
    ) -> Result<Chore, ApiError> {
        self.json(
            Method::PUT,
            &format!("/houses/{house_id}/chores/{id}"),
            value,
        )
        .await
    }
    pub async fn delete_chore(
        &self,
        house_id: &str,
        id: &str,
        version: i64,
    ) -> Result<(), ApiError> {
        self.empty(self.request(
            Method::DELETE,
            &format!("/houses/{house_id}/chores/{id}?version={version}"),
        ))
        .await
    }
    pub async fn get_chore_completions(
        &self,
        house_id: &str,
        id: &str,
    ) -> Result<Vec<ChoreCompletion>, ApiError> {
        self.send(self.request(
            Method::GET,
            &format!("/houses/{house_id}/chores/{id}/completions"),
        ))
        .await
    }
    pub async fn complete_chore(
        &self,
        house_id: &str,
        id: &str,
        value: &CompleteChoreRequest,
    ) -> Result<ChoreCompletion, ApiError> {
        self.json(
            Method::POST,
            &format!("/houses/{house_id}/chores/{id}/completions"),
            value,
        )
        .await
    }
    pub async fn get_calendar(&self, house_id: &str) -> Result<Vec<CalendarEvent>, ApiError> {
        self.send(self.request(Method::GET, &format!("/houses/{house_id}/calendar")))
            .await
    }
    pub async fn create_calendar(
        &self,
        house_id: &str,
        value: &CalendarEventRequest,
    ) -> Result<CalendarEvent, ApiError> {
        self.json(Method::POST, &format!("/houses/{house_id}/calendar"), value)
            .await
    }
    pub async fn update_calendar(
        &self,
        house_id: &str,
        id: &str,
        value: &CalendarEventRequest,
    ) -> Result<CalendarEvent, ApiError> {
        self.json(
            Method::PUT,
            &format!("/houses/{house_id}/calendar/{id}"),
            value,
        )
        .await
    }
    pub async fn delete_calendar(
        &self,
        house_id: &str,
        id: &str,
        version: i64,
    ) -> Result<(), ApiError> {
        self.empty(self.request(
            Method::DELETE,
            &format!("/houses/{house_id}/calendar/{id}?version={version}"),
        ))
        .await
    }
    pub async fn get_chat(
        &self,
        house_id: &str,
        before: Option<i64>,
    ) -> Result<ChatPage, ApiError> {
        let suffix = before
            .map(|value| format!("?before={value}&limit=50"))
            .unwrap_or_else(|| "?limit=50".into());
        self.send(self.request(Method::GET, &format!("/houses/{house_id}/chat{suffix}")))
            .await
    }
    pub async fn create_chat(
        &self,
        house_id: &str,
        value: &ChatMessageRequest,
    ) -> Result<ChatMessage, ApiError> {
        self.json(Method::POST, &format!("/houses/{house_id}/chat"), value)
            .await
    }
    pub async fn update_chat(
        &self,
        house_id: &str,
        id: &str,
        value: &ChatMessageRequest,
    ) -> Result<ChatMessage, ApiError> {
        self.json(Method::PUT, &format!("/houses/{house_id}/chat/{id}"), value)
            .await
    }
    pub async fn delete_chat(&self, house_id: &str, id: &str) -> Result<(), ApiError> {
        self.empty(self.request(Method::DELETE, &format!("/houses/{house_id}/chat/{id}")))
            .await
    }
    pub async fn get_vapid_public_key(&self) -> Result<VapidPublicKey, ApiError> {
        self.send(self.request(Method::GET, "/notifications/vapid-public-key"))
            .await
    }
}

#[derive(Debug, Clone, Serialize, serde::Deserialize, PartialEq)]
pub struct ExpenseDetail {
    pub expense: Expense,
    #[serde(default)]
    pub splits: Vec<ExpenseSplit>,
}

#[cfg(all(feature = "web", feature = "desktop"))]
compile_error!("select exactly one renderer feature");
#[cfg(all(feature = "web", feature = "mobile"))]
compile_error!("select exactly one renderer feature");
#[cfg(all(feature = "desktop", feature = "mobile"))]
compile_error!("select exactly one renderer feature");

#[cfg(test)]
mod tests {
    use super::*;

    #[derive(Default)]
    struct FakeSessionStore {
        token: Mutex<Option<String>>,
        save_error: Mutex<Option<storage::StorageError>>,
        clear_error: Mutex<Option<storage::StorageError>>,
        save_started: Mutex<Option<std::sync::mpsc::Sender<()>>>,
        save_release: Mutex<Option<std::sync::mpsc::Receiver<()>>>,
        clears: Mutex<usize>,
    }

    impl storage::SessionTokenStore for FakeSessionStore {
        fn load(&self) -> Result<Option<String>, storage::StorageError> {
            Ok(self.token.lock().unwrap().clone())
        }
        fn save(&self, token: &str) -> Result<(), storage::StorageError> {
            if let Some(started) = self.save_started.lock().unwrap().take() {
                let _ = started.send(());
            }
            if let Some(release) = self.save_release.lock().unwrap().take() {
                let _ = release.recv();
            }
            if let Some(error) = *self.save_error.lock().unwrap() {
                return Err(error);
            }
            *self.token.lock().unwrap() = Some(token.to_owned());
            Ok(())
        }
        fn clear(&self) -> Result<(), storage::StorageError> {
            *self.clears.lock().unwrap() += 1;
            *self.token.lock().unwrap() = None;
            if let Some(error) = *self.clear_error.lock().unwrap() {
                Err(error)
            } else {
                Ok(())
            }
        }
    }

    #[test]
    fn detail_decodes_backend_envelope() {
        let detail: ExpenseDetail = serde_json::from_str(r#"{
            "expense": {"id":"e","house_id":"h","payer_id":"u","amount":3.25,"description":"Lunch","category":"food","date":"2025-01-01","visibility":"private","created_at":"now"},
            "splits": []
        }"#).unwrap();
        assert_eq!(detail.expense.description, "Lunch");
    }

    #[test]
    fn member_role_update_decodes_message_response() {
        let response: MessageResponse =
            serde_json::from_str(r#"{"message":"role updated"}"#).unwrap();
        assert_eq!(response.message, "role updated");
    }

    #[test]
    fn invitation_contract_decodes_create_list_and_accept_fields() {
        let invite: HouseInvitation = serde_json::from_str(
            r#"{
            "id":"invite-1","house_id":"house-1","email":"user@example.test",
            "role":"member","status":"pending","created_by":"admin-1",
            "created_at":"2025-01-01T00:00:00Z","expires_at":"2025-01-08T00:00:00Z"
        }"#,
        )
        .unwrap();
        assert_eq!(invite.email, "user@example.test");
        assert_eq!(invite.status, "pending");
        assert!(invite.manual_acceptance_url.is_none());
        let accepted: InvitationAcceptanceResponse = serde_json::from_str(&format!(
            "{{\"invitation\":{}}}",
            serde_json::to_string(&invite).unwrap()
        ))
        .unwrap();
        assert_eq!(accepted.invitation.id, "invite-1");
    }

    #[test]
    fn api_base_url_defaults_are_environment_specific() {
        assert_eq!(
            resolve_api_base_url(None, false, false, None),
            "http://localhost:8080/api"
        );
        assert_eq!(
            resolve_api_base_url(None, false, true, None),
            "http://10.0.2.2:8080/api"
        );
        assert_eq!(resolve_api_base_url(None, true, false, None), "/api");
        assert_eq!(
            resolve_api_base_url(None, true, false, Some("https://app.example".into())),
            "https://app.example/api"
        );
        assert_eq!(
            resolve_api_base_url(Some("http://192.168.1.10:8080/api"), false, true, None),
            "http://192.168.1.10:8080/api"
        );
    }

    #[test]
    fn session_invalidation_disables_stale_tokens() {
        let client = ApiClient::with_base_url("http://example");
        {
            let mut session = client.session.lock().unwrap();
            session.token = Some("token".into());
            session.valid = true;
        }
        assert!(client.is_authenticated());
        client.invalidate_session();
        assert!(!client.is_authenticated());
        assert_eq!(
            client.session.lock().unwrap().token.as_deref(),
            Some("token")
        );
    }

    #[test]
    fn failed_secure_save_does_not_authenticate_the_client() {
        let store = Arc::new(FakeSessionStore::default());
        *store.save_error.lock().unwrap() = Some(storage::StorageError::WriteFailed);
        let client = ApiClient::with_base_url_and_store("http://example", store);

        assert!(matches!(
            client.set_token("token".into()),
            Err(ApiError::SessionStorage(storage::StorageError::WriteFailed))
        ));
        assert!(!client.has_saved_token());
        assert!(!client.is_authenticated());
    }

    #[test]
    fn logout_clears_memory_even_when_secure_deletion_fails() {
        let store = Arc::new(FakeSessionStore::default());
        *store.clear_error.lock().unwrap() = Some(storage::StorageError::DeleteFailed);
        let client = ApiClient::with_base_url_and_store("http://example", store.clone());
        client.set_token("token".into()).unwrap();

        assert_eq!(client.logout(), Err(storage::StorageError::DeleteFailed));
        assert!(!client.has_saved_token());
        assert_eq!(
            client.storage_error(),
            Some(storage::StorageError::DeleteFailed)
        );
        assert_eq!(*store.clears.lock().unwrap(), 1);
    }

    #[test]
    fn unauthorized_cleanup_clears_memory_and_retains_bounded_error() {
        let store = Arc::new(FakeSessionStore::default());
        *store.clear_error.lock().unwrap() = Some(storage::StorageError::DeleteFailed);
        let client = ApiClient::with_base_url_and_store("http://example", store.clone());
        client.set_token("token".into()).unwrap();
        let generation = client.session.lock().unwrap().generation;

        client.handle_unauthorized(Some(generation));

        assert!(!client.has_saved_token());
        assert!(!client.is_authenticated());
        assert_eq!(
            client.storage_error(),
            Some(storage::StorageError::DeleteFailed)
        );
        assert_eq!(*store.clears.lock().unwrap(), 1);
    }

    #[test]
    fn request_snapshot_remains_panic_free_during_logout() {
        let store = Arc::new(FakeSessionStore::default());
        let client = ApiClient::with_base_url_and_store("http://example", store);
        client.set_token("token".into()).unwrap();

        let request = client.request(Method::GET, "/protected");
        client.logout().unwrap();

        assert!(request.generation.is_some());
        assert!(!client.is_authenticated());
    }

    #[test]
    fn stale_unauthorized_response_does_not_clear_replacement_session() {
        let store = Arc::new(FakeSessionStore::default());
        let client = ApiClient::with_base_url_and_store("http://example", store.clone());
        client.set_token("first-token".into()).unwrap();
        let stale_generation = client.request(Method::GET, "/protected").generation;
        client.set_token("replacement-token".into()).unwrap();

        client.handle_unauthorized(stale_generation);

        assert!(client.is_authenticated());
        assert_eq!(
            client.session.lock().unwrap().token.as_deref(),
            Some("replacement-token")
        );
        assert_eq!(
            store.token.lock().unwrap().as_deref(),
            Some("replacement-token")
        );
        assert_eq!(*store.clears.lock().unwrap(), 0);
    }

    #[test]
    fn concurrent_logout_is_ordered_after_in_progress_session_save() {
        let store = Arc::new(FakeSessionStore::default());
        let client = ApiClient::with_base_url_and_store("http://example", store.clone());
        client.set_token("first-token".into()).unwrap();
        let (started_tx, started_rx) = std::sync::mpsc::channel();
        let (release_tx, release_rx) = std::sync::mpsc::channel();
        *store.save_started.lock().unwrap() = Some(started_tx);
        *store.save_release.lock().unwrap() = Some(release_rx);

        let saving = {
            let client = client.clone();
            std::thread::spawn(move || client.set_token("replacement-token".into()))
        };
        started_rx.recv().unwrap();
        let logging_out = {
            let client = client.clone();
            std::thread::spawn(move || client.logout())
        };
        release_tx.send(()).unwrap();

        assert!(saving.join().unwrap().is_ok());
        assert!(logging_out.join().unwrap().is_ok());
        assert!(!client.is_authenticated());
        assert_eq!(*store.token.lock().unwrap(), None);
    }

    #[test]
    fn visibility_request_preserves_recipients() {
        let request = SetVisibilityRequest {
            visibility: "private".into(),
            visible_to: vec!["member".into()],
        };
        assert_eq!(
            serde_json::to_string(&request).unwrap(),
            r#"{"visibility":"private","visible_to":["member"]}"#
        );
    }
}
