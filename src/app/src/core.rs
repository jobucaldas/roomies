//! Renderer-independent domain rules used by UI and tests.

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Role {
    Admin,
    Member,
    Monitor,
}
impl Role {
    pub fn parse(value: &str) -> Option<Self> {
        match value {
            "admin" => Some(Self::Admin),
            "member" => Some(Self::Member),
            "monitor" => Some(Self::Monitor),
            _ => None,
        }
    }
}

pub fn can_create(role: Role, kind: &str) -> bool {
    match kind {
        "expense" | "note" => !matches!(role, Role::Monitor),
        _ => false,
    }
}
pub fn can_manage(role: Role) -> bool {
    matches!(role, Role::Admin)
}
pub fn can_mutate_owned(role: Role, owner: bool) -> bool {
    owner && !matches!(role, Role::Monitor)
}
pub fn can_mutate_note(role: Role, author: bool) -> bool {
    author || matches!(role, Role::Admin)
}
pub fn can_change_expense_visibility(role: Role, payer: bool) -> bool {
    payer && !matches!(role, Role::Monitor)
}

pub fn parse_money_cents(value: &str) -> Result<i64, &'static str> {
    let value = value.trim();
    if value.is_empty() || value.starts_with('-') {
        return Err("amount must be positive");
    }
    let mut parts = value.split('.');
    let whole: i64 = parts
        .next()
        .unwrap()
        .parse()
        .map_err(|_| "invalid amount")?;
    let fraction = parts.next().unwrap_or("");
    if parts.next().is_some() || fraction.len() > 2 || !fraction.chars().all(|c| c.is_ascii_digit())
    {
        return Err("invalid amount");
    }
    let cents: i64 = fraction.parse().unwrap_or(0) * if fraction.len() == 1 { 10 } else { 1 };
    whole
        .checked_mul(100)
        .and_then(|v| v.checked_add(cents))
        .ok_or("amount is too large")
}

/// Converts validated cents only at the HTTP boundary; forms never add floating point values.
pub fn cents_to_api_amount(cents: i64) -> f64 {
    cents as f64 / 100.0
}

pub fn format_cents(cents: i64) -> String {
    format!("{}.{:02}", cents / 100, cents % 100)
}

pub fn equal_split_cents(
    total: i64,
    user_ids: &[String],
) -> Result<Vec<(String, i64)>, &'static str> {
    if total <= 0 || user_ids.is_empty() {
        return Err("an expense needs an amount and at least one participant");
    }
    let base = total / user_ids.len() as i64;
    let remainder = total % user_ids.len() as i64;
    Ok(user_ids
        .iter()
        .enumerate()
        .map(|(index, id)| (id.clone(), base + i64::from((index as i64) < remainder)))
        .collect())
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum Session {
    SignedOut,
    SignedIn { user_id: String },
}
impl Session {
    pub fn login(user_id: impl Into<String>) -> Self {
        Self::SignedIn {
            user_id: user_id.into(),
        }
    }
    pub fn logout(&mut self) {
        *self = Self::SignedOut;
    }
    pub fn is_authenticated(&self) -> bool {
        matches!(self, Self::SignedIn { .. })
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn permissions_are_deterministic() {
        assert!(!can_create(Role::Monitor, "expense"));
        assert!(can_manage(Role::Admin));
        assert!(!can_mutate_owned(Role::Member, false));
        assert!(can_mutate_note(Role::Admin, false));
        assert!(can_mutate_note(Role::Member, true));
        assert!(!can_mutate_note(Role::Member, false));
        assert!(can_change_expense_visibility(Role::Member, true));
        assert!(!can_change_expense_visibility(Role::Monitor, true));
    }
    #[test]
    fn money_is_cent_exact() {
        assert_eq!(parse_money_cents("12.3"), Ok(1230));
        assert_eq!(parse_money_cents("1.234"), Err("invalid amount"));
        assert_eq!(
            equal_split_cents(100, &["a".into(), "b".into(), "c".into()]),
            Ok(vec![("a".into(), 34), ("b".into(), 33), ("c".into(), 33)])
        );
        assert_eq!(format_cents(123), "1.23");
    }
    #[test]
    fn session_transitions() {
        let mut s = Session::login("u");
        assert!(s.is_authenticated());
        s.logout();
        assert!(!s.is_authenticated());
    }
}
