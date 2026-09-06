use crate::api::{ApiClient, ApiError};
use crate::models::User;
use crate::router::Route;
use crate::storage;
use dioxus::prelude::*;

/// Captures the one-time URL token into session storage and removes it from the
/// address bar before any auth redirect. The token is never rendered.
fn capture_invitation_token() {
    #[cfg(target_arch = "wasm32")]
    {
        let Some(window) = web_sys::window() else {
            return;
        };
        let Ok(search) = window.location().search() else {
            return;
        };
        if let Ok(params) = web_sys::UrlSearchParams::new_with_str(&search) {
            if let Some(token) = params.get("token") {
                if !token.is_empty() {
                    storage::save_pending_invitation(&token);
                    if let Ok(history) = window.history() {
                        let _ = history.replace_state_with_url(
                            &wasm_bindgen::JsValue::NULL,
                            "",
                            Some("/accept-invitation"),
                        );
                    }
                }
            }
        }
    }
}

#[component]
pub fn AcceptInvitation() -> Element {
    let navigator = use_navigator();
    let api = use_context::<Signal<ApiClient>>();
    let current_user = use_context::<Signal<Option<User>>>();
    let mut attempted = use_signal(|| false);
    let mut loading = use_signal(|| false);
    let mut status = use_signal(String::new);
    let mut success = use_signal(|| false);
    let mut destination = use_signal(String::new);
    let mut retryable = use_signal(|| false);

    {
        use_effect(move || {
            let house_id = destination.read().clone();
            if *success.read() && !house_id.is_empty() {
                navigator.replace(Route::HouseDetail { id: house_id });
            }
        });
    }

    use_effect(move || {
        capture_invitation_token();
        let authenticated = api.read().is_authenticated();
        if !authenticated {
            navigator.replace(Route::Login {});
            return;
        }
        if *attempted.read() {
            return;
        }
        attempted.set(true);
        let Some(token) = storage::pending_invitation() else {
            status.set("This invitation link is missing or has already been used.".into());
            return;
        };
        loading.set(true);
        let api = api.read().clone();
        let navigator = navigator;
        spawn(async move {
            match api.accept_invitation(&token).await {
                Ok(response) => {
                    storage::clear_pending_invitation();
                    success.set(true);
                    destination.set(response.invitation.house_id);
                    status.set("You joined the house. Refreshing your membership…".into());
                }
                Err(error) => {
                    if invitation_failure_is_retryable(&error) {
                        // Keep a still-valid bearer token across transient failures so
                        // the recipient can retry without reopening the email.
                        retryable.set(true);
                        status.set(format!("Unable to accept this invitation: {error}"));
                        if matches!(error, ApiError::Http { status: 401, .. }) {
                            navigator.replace(Route::Login {});
                        }
                    } else {
                        // A rejected, expired, or revoked bearer token must not remain
                        // available to an unrelated later session in this browser.
                        retryable.set(false);
                        storage::clear_pending_invitation();
                        status.set(format!("Unable to accept this invitation: {error}"));
                    }
                }
            }
            loading.set(false);
        });
    });

    // Keep the user context observed so a completed auth redirect re-renders this
    // route even when the session signal is updated by the auth form.
    let _ = current_user.read();
    rsx! {
        main { class: "container",
            h1 { "Roomies invitation" }
            if *loading.read() {
                p { role: "status", "Joining house…" }
            } else if *success.read() {
                p { role: "status", "{status}" }
            } else if !status.read().is_empty() {
                p { class: "error", role: "alert", "{status}" }
                if *retryable.read() {
                    button {
                        r#type: "button",
                        onclick: move |_| {
                            retryable.set(false);
                            status.set("Retrying invitation acceptance…".into());
                            attempted.set(false);
                        },
                        "Retry acceptance"
                    }
                }
            } else {
                p { "Checking invitation…" }
            }
        }
    }
}

fn invitation_failure_is_retryable(error: &ApiError) -> bool {
    match error {
        ApiError::Transport(_) | ApiError::Decode(_) => true,
        ApiError::Http { status, .. } => matches!(*status, 401 | 408 | 429) || *status >= 500,
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn transient_invitation_failures_are_retryable() {
        assert!(invitation_failure_is_retryable(&ApiError::Transport(
            "offline".into()
        )));
        assert!(invitation_failure_is_retryable(&ApiError::Decode(
            "bad json".into()
        )));
        for status in [401, 408, 429, 500, 503] {
            assert!(invitation_failure_is_retryable(&ApiError::Http {
                status,
                message: "temporary".into(),
            }));
        }
    }

    #[test]
    fn definitive_invitation_failures_are_cleared() {
        for status in [400, 403, 404, 409] {
            assert!(!invitation_failure_is_retryable(&ApiError::Http {
                status,
                message: "definitive".into(),
            }));
        }
    }
}
