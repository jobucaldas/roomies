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
