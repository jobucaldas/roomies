use crate::models::*;
use crate::storage;
use reqwest::{Client, Method};
use serde::de::DeserializeOwned;
use serde::Serialize;
use std::fmt;
use std::sync::{
    atomic::{AtomicBool, Ordering},
    Arc,
};

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
}
impl fmt::Display for ApiError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Http { message, .. } | Self::Transport(message) | Self::Decode(message) => {
                f.write_str(message)
            }
        }
    }
}
impl std::error::Error for ApiError {}

#[derive(Clone)]
pub struct ApiClient {
    pub token: Option<String>,
    pub base_url: String,
    client: Client,
    session_valid: Arc<AtomicBool>,
}
impl Default for ApiClient {
    fn default() -> Self {
        Self::new()
    }
}
impl ApiClient {
    pub fn new() -> Self {
        let mut client = Self::with_base_url(api_base_url());
        client.token = storage::load_token();
        client
    }
    pub fn with_base_url(base_url: impl Into<String>) -> Self {
        Self {
            token: None,
            base_url: base_url.into().trim_end_matches('/').to_string(),
            client: Client::new(),
            session_valid: Arc::new(AtomicBool::new(true)),
        }
    }
    pub fn is_authenticated(&self) -> bool {
        self.token.is_some() && self.session_valid.load(Ordering::Relaxed)
    }
    fn set_token(&mut self, token: String) {
        storage::save_token(&token);
        self.token = Some(token);
        self.session_valid.store(true, Ordering::Relaxed);
    }
    pub fn invalidate_session(&self) {
        self.session_valid.store(false, Ordering::Relaxed);
    }
    fn clear_persisted_session(&self) {
        self.invalidate_session();
        storage::clear_token();
    }
    fn request(&self, method: Method, path: &str) -> reqwest::RequestBuilder {
        let builder = self
            .client
            .request(method, format!("{}{}", self.base_url, path))
            .header("Content-Type", "application/json");
        if self.is_authenticated() {
            builder.bearer_auth(self.token.as_ref().expect("authenticated token"))
        } else {
            builder
        }
    }
    async fn send<T: DeserializeOwned>(
        &self,
        request: reqwest::RequestBuilder,
    ) -> Result<T, ApiError> {
        let response = request
            .send()
            .await
            .map_err(|e| ApiError::Transport(e.to_string()))?;
        let status = response.status();
        if !status.is_success() {
            if status.as_u16() == 401 {
                self.clear_persisted_session();
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
    async fn empty(&self, request: reqwest::RequestBuilder) -> Result<(), ApiError> {
        let response = request
            .send()
            .await
            .map_err(|e| ApiError::Transport(e.to_string()))?;
        if response.status().is_success() {
            Ok(())
        } else {
            if response.status().as_u16() == 401 {
                self.clear_persisted_session();
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
        self.set_token(a.token.clone());
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
        self.set_token(a.token.clone());
        Ok(a)
    }
    pub async fn me(&self) -> Result<User, ApiError> {
        self.send(self.request(Method::GET, "/auth/me")).await
    }
    pub fn logout(&mut self) {
        self.token = None;
        self.clear_persisted_session();
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
        let mut client = ApiClient::with_base_url("http://example");
        client.token = Some("token".into());
        assert!(client.is_authenticated());
        client.invalidate_session();
        assert!(!client.is_authenticated());
        assert_eq!(client.token.as_deref(), Some("token"));
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
