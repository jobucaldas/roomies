use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct User {
    pub id: String,
    pub name: String,
    pub email: String,
    pub created_at: String,
}
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct House {
    pub id: String,
    pub name: String,
    pub created_at: String,
}
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct HouseMember {
    pub id: String,
    pub house_id: String,
    pub user_id: String,
    pub role: String,
    pub joined_at: String,
    #[serde(default)]
    pub user_name: String,
    #[serde(default)]
    pub user_email: String,
}
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct Expense {
    pub id: String,
    pub house_id: String,
    pub payer_id: String,
    pub amount: f64,
    pub description: String,
    #[serde(default)]
    pub category: String,
    pub date: String,
    pub visibility: String,
    pub created_at: String,
    #[serde(default)]
    pub payer_name: String,
}
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct ExpenseSplit {
    pub id: String,
    pub expense_id: String,
    pub user_id: String,
    pub share_amount: f64,
    #[serde(default)]
    pub user_name: String,
}
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct Note {
    pub id: String,
    pub house_id: String,
    pub author_id: String,
    pub title: String,
    pub content: String,
    pub created_at: String,
    pub updated_at: String,
    #[serde(default)]
    pub author_name: String,
}
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct BalanceEntry {
    pub user_id: String,
    pub user_name: String,
    pub paid: f64,
    pub owed: f64,
    pub net: f64,
}
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct BalanceSettlement {
    pub from_user_id: String,
    pub from_user_name: String,
    pub to_user_id: String,
    pub to_user_name: String,
    pub amount: f64,
}
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct BalanceResponse {
    pub balances: Vec<BalanceEntry>,
    pub settlements: Vec<BalanceSettlement>,
}
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuthResponse {
    pub token: String,
    pub user: User,
}
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct MessageResponse {
    pub message: String,
}
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct HouseInvitation {
    pub id: String,
    pub house_id: String,
    pub email: String,
    pub role: String,
    pub status: String,
    pub created_by: String,
    pub created_at: DateTime<Utc>,
    pub expires_at: DateTime<Utc>,
    #[serde(default)]
    pub accepted_by: Option<String>,
    #[serde(default)]
    pub accepted_at: Option<DateTime<Utc>>,
    #[serde(default)]
    pub revoked_by: Option<String>,
    #[serde(default)]
    pub revoked_at: Option<DateTime<Utc>>,
    #[serde(default, skip_serializing)]
    pub manual_acceptance_url: Option<String>,
}
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct InvitationAcceptanceResponse {
    pub invitation: HouseInvitation,
}
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct CreateInvitationRequest {
    pub email: String,
    pub role: String,
}
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct AcceptInvitationRequest {
    pub token: String,
}
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ErrorResponse {
    pub error: String,
}
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateExpenseRequest {
    pub amount: f64,
    pub description: String,
    #[serde(default)]
    pub category: String,
    pub date: String,
    #[serde(default = "default_visibility")]
    pub visibility: String,
    #[serde(default)]
    pub visible_to: Vec<String>,
    #[serde(default)]
    pub split: Vec<SplitEntry>,
}
fn default_visibility() -> String {
    "shared".to_string()
}
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SetVisibilityRequest {
    pub visibility: String,
    #[serde(default)]
    pub visible_to: Vec<String>,
}
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SplitEntry {
    pub user_id: String,
    pub amount: f64,
}
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateExpenseRequest {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub amount: Option<f64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub description: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub category: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub date: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub split: Option<Vec<SplitEntry>>,
}
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LoginRequest {
    pub email: String,
    pub password: String,
}
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RegisterRequest {
    pub name: String,
    pub email: String,
    pub password: String,
}
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateNoteRequest {
    pub title: String,
    pub content: String,
}
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateNoteRequest {
    pub title: String,
    pub content: String,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct NotificationPreferences {
    #[serde(default)]
    pub house_id: String,
    #[serde(default)]
    pub user_id: String,
    #[serde(default = "default_true")]
    pub expense_created_enabled: bool,
    #[serde(default = "default_true")]
    pub reminder_enabled: bool,
    #[serde(default = "default_cadence")]
    pub cadence: String,
    #[serde(default = "default_timezone")]
    pub timezone: String,
    #[serde(default)]
    pub quiet_start_minutes: Option<u16>,
    #[serde(default)]
    pub quiet_end_minutes: Option<u16>,
    #[serde(default)]
    pub digest_minutes: u16,
}
fn default_true() -> bool {
    true
}
fn default_cadence() -> String {
    "immediate".into()
}
fn default_timezone() -> String {
    "UTC".into()
}
impl Default for NotificationPreferences {
    fn default() -> Self {
        Self {
            house_id: String::new(),
            user_id: String::new(),
            expense_created_enabled: true,
            reminder_enabled: true,
            cadence: default_cadence(),
            timezone: default_timezone(),
            quiet_start_minutes: None,
            quiet_end_minutes: None,
            digest_minutes: 540,
        }
    }
}
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct NotificationSubscription {
    pub id: String,
    pub house_id: String,
    pub user_id: String,
    pub platform: String,
    pub device_label: String,
    pub created_at: String,
    pub last_seen_at: String,
    #[serde(default)]
    pub revoked_at: Option<String>,
}
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateNotificationSubscriptionRequest {
    pub platform: String,
    pub device_label: String,
    pub endpoint: String,
    pub p256dh: String,
    pub auth: String,
}
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VapidPublicKey {
    #[serde(default)]
    pub public_key: String,
}
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct ScheduledHouseEvent {
    pub id: String,
    pub house_id: String,
    pub creator_id: String,
    pub title: String,
    pub timezone: String,
    pub dtstart_local: String,
    pub rrule: String,
    #[serde(default)]
    pub exdates: Vec<String>,
    #[serde(default)]
    pub next_occurrence_at: Option<String>,
    pub enabled: bool,
    pub created_at: String,
    pub updated_at: String,
}
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ScheduledEventRequest {
    pub title: String,
    pub timezone: String,
    pub dtstart_local: String,
    pub rrule: String,
    pub exdates: Vec<String>,
    pub enabled: Option<bool>,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct GroceryItem {
    pub id: String,
    pub house_id: String,
    pub creator_id: String,
    pub name: String,
    pub quantity: String,
    pub unit: String,
    pub note: String,
    #[serde(default)]
    pub assignee_id: Option<String>,
    pub checked: bool,
    #[serde(default)]
    pub checked_by: Option<String>,
    #[serde(default)]
    pub checked_at: Option<String>,
    pub position: i32,
    pub version: i64,
    pub created_at: String,
    pub updated_at: String,
}
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GroceryRequest {
    pub name: String,
    pub quantity: String,
    pub unit: String,
    pub note: String,
    pub assignee_id: Option<String>,
    pub position: i32,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub version: Option<i64>,
}
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GroceryToggleRequest {
    pub checked: bool,
    pub version: i64,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct Chore {
    pub id: String,
    pub house_id: String,
    pub creator_id: String,
    pub title: String,
    pub description: String,
    #[serde(default)]
    pub assignee_id: Option<String>,
    pub timezone: String,
    pub due_local: String,
    pub rrule: String,
    #[serde(default)]
    pub exdates: Vec<String>,
    pub enabled: bool,
    pub version: i64,
    pub created_at: String,
    pub updated_at: String,
}
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChoreRequest {
    pub title: String,
    pub description: String,
    pub assignee_id: Option<String>,
    pub timezone: String,
    pub due_local: String,
    pub rrule: String,
    pub exdates: Vec<String>,
    pub enabled: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub version: Option<i64>,
}
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct ChoreCompletion {
    pub id: String,
    pub chore_id: String,
    pub house_id: String,
    pub occurrence_at: String,
    pub completed_by: String,
    pub completed_at: String,
}
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CompleteChoreRequest {
    pub occurrence_at: String,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct CalendarEvent {
    pub id: String,
    pub house_id: String,
    pub creator_id: String,
    pub title: String,
    pub description: String,
    pub timezone: String,
    pub start_local: String,
    pub end_local: String,
    pub all_day: bool,
    pub rrule: String,
    #[serde(default)]
    pub exdates: Vec<String>,
    pub version: i64,
    pub created_at: String,
    pub updated_at: String,
}
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CalendarEventRequest {
    pub title: String,
    pub description: String,
    pub timezone: String,
    pub start_local: String,
    pub end_local: String,
    pub all_day: bool,
    pub rrule: String,
    pub exdates: Vec<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub version: Option<i64>,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct ChatMessage {
    pub cursor: i64,
    pub id: String,
    pub house_id: String,
    pub author_id: String,
    #[serde(default)]
    pub body: Option<String>,
    pub created_at: String,
    pub updated_at: String,
    #[serde(default)]
    pub deleted_at: Option<String>,
    #[serde(default)]
    pub redacted_at: Option<String>,
}
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct ChatPage {
    pub messages: Vec<ChatMessage>,
    #[serde(default)]
    pub next_cursor: String,
}
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChatMessageRequest {
    pub body: String,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn household_update_requests_include_versions_and_create_requests_omit_them() {
        let create = GroceryRequest {
            name: "Milk".into(),
            quantity: "1".into(),
            unit: "L".into(),
            note: String::new(),
            assignee_id: None,
            position: 0,
            version: None,
        };
        let update = ChoreRequest {
            title: "Bins".into(),
            description: String::new(),
            assignee_id: None,
            timezone: "UTC".into(),
            due_local: "2025-02-01T09:00:00".into(),
            rrule: "FREQ=WEEKLY;COUNT=1".into(),
            exdates: vec!["2025-02-08T09:00:00".into()],
            enabled: Some(false),
            version: Some(2),
        };
        let create_json = serde_json::to_value(create).unwrap();
        let update_json = serde_json::to_value(update).unwrap();
        assert!(create_json.get("version").is_none());
        assert_eq!(update_json["version"], 2);
        assert_eq!(update_json["exdates"][0], "2025-02-08T09:00:00");
    }
}
