use crate::api::ApiClient;
use crate::models::HouseInvitation;
use dioxus::prelude::*;

pub fn invitation_status_label(invitation: &HouseInvitation) -> String {
    match invitation.status.as_str() {
        "pending" => format!("Pending · expires {}", invitation.expires_at.to_rfc3339()),
        "expired" => "Expired".into(),
        "revoked" => "Revoked".into(),
        "accepted" => "Accepted".into(),
        other => other.to_owned(),
    }
}

#[component]
pub fn InvitationPanel(
    house_id: String,
    invitations: Vec<HouseInvitation>,
    loading: bool,
    error: String,
    on_refresh: EventHandler<()>,
) -> Element {
    let api = use_context::<Signal<ApiClient>>();
    let mut email = use_signal(String::new);
    let mut role = use_signal(|| "member".to_string());
    let mut status = use_signal(String::new);
    let mut submitting = use_signal(|| false);
    let create_house_id = house_id.clone();
    let create = move |_| {
        let email_value = email.read().trim().to_owned();
        if email_value.is_empty() {
            status.set("Email is required.".into());
            return;
        }
        submitting.set(true);
        status.set(String::new());
        let api = api.read().clone();
        let house_id = create_house_id.clone();
        let role = role.read().clone();
        spawn(async move {
            match api.create_invitation(&house_id, &email_value, &role).await {
                Ok(_) => {
                    // The one-time URL is intentionally not rendered or copied to
                    // general UI; delivery is handled by the configured mail sink.
                    email.set(String::new());
                    status.set("Invitation sent. It expires according to house policy.".into());
                    on_refresh.call(());
                }
                Err(error) => status.set(format!("Unable to send invitation: {error}")),
            }
            submitting.set(false);
        });
    };

    rsx! {
        section { class: "card", aria_label: "Email invitations",
            h3 { "Invite by email" }
            label { "Email address", input {
                r#type: "email",
                autocomplete: "email",
                value: "{email}",
                oninput: move |event| email.set(event.value()),
            } }
            label { "Role", select {
                value: "{role}",
                oninput: move |event| role.set(event.value()),
                option { value: "member", "Member" }
                option { value: "admin", "Admin" }
                option { value: "monitor", "Monitor" }
            } }
            button { disabled: *submitting.read(), onclick: create,
                if *submitting.read() { "Sending…" } else { "Send invitation" }
            }
            if !status.read().is_empty() { p { role: "status", "{status}" } }
            if !error.is_empty() { p { class: "error", role: "alert", "{error}" } }
            h3 { "Invitation history" }
            if loading {
                p { "Loading invitations…" }
            } else if invitations.is_empty() {
                p { "No invitations yet." }
            } else {
                for invitation in invitations {
                    InvitationRow {
                        house_id: house_id.clone(),
                        invitation,
                        on_refresh,
                    }
                }
            }
        }
    }
}

#[component]
fn InvitationRow(
    house_id: String,
    invitation: HouseInvitation,
    on_refresh: EventHandler<()>,
) -> Element {
    let api = use_context::<Signal<ApiClient>>();
    let mut revoking = use_signal(|| false);
    let mut error = use_signal(String::new);
    let revoke_house_id = house_id.clone();
    let revoke_id = invitation.id.clone();
    let revoke = move |_| {
        revoking.set(true);
        error.set(String::new());
        let api = api.read().clone();
        let house_id = revoke_house_id.clone();
        let invitation_id = revoke_id.clone();
        spawn(async move {
            match api.revoke_invitation(&house_id, &invitation_id).await {
                Ok(_) => on_refresh.call(()),
                Err(value) => error.set(format!("Unable to revoke invitation: {value}")),
            }
            revoking.set(false);
        });
    };
    let can_revoke = invitation.status == "pending";
    rsx! {
        article { class: "invitation-row",
            strong { "{invitation.email}" }
            p { "Role: {invitation.role} · {invitation_status_label(&invitation)}" }
            if can_revoke {
                button { disabled: *revoking.read(), onclick: revoke,
                    if *revoking.read() { "Revoking…" } else { "Revoke" }
                }
            }
            if !error.read().is_empty() { p { class: "error", role: "alert", "{error}" } }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use chrono::{TimeZone, Utc};

    fn invitation(status: &str) -> HouseInvitation {
        HouseInvitation {
            id: "id".into(),
            house_id: "house".into(),
            email: "user@example.test".into(),
            role: "member".into(),
            status: status.into(),
            created_by: "admin".into(),
            created_at: Utc.timestamp_opt(0, 0).unwrap(),
            expires_at: Utc.timestamp_opt(60, 0).unwrap(),
            accepted_by: None,
            accepted_at: None,
            revoked_by: None,
            revoked_at: None,
            manual_acceptance_url: None,
        }
    }

    #[test]
    fn invitation_status_is_user_safe_and_does_not_include_token() {
        let label = invitation_status_label(&invitation("pending"));
        assert!(label.contains("Pending"));
        assert!(!label.contains("token"));
        assert_eq!(invitation_status_label(&invitation("expired")), "Expired");
        assert_eq!(invitation_status_label(&invitation("revoked")), "Revoked");
    }
}
