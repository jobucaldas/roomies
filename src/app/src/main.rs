use dioxus::prelude::*;
use roomies_app::api::{ApiClient, ApiError};
use roomies_app::models::User;
use roomies_app::router::Route;

fn main() {
    dioxus::launch(App);
}

#[component]
fn App() -> Element {
    use_context_provider(|| Signal::new(ApiClient::new()));
    use_context_provider(|| Signal::new(Option::<User>::None));
    let api = use_context::<Signal<ApiClient>>();
    let current_user = use_context::<Signal<Option<User>>>();
    let session_ready = use_signal(|| false);
    let mut restore_started = use_signal(|| false);

    use_effect(move || {
        if *restore_started.read() {
            return;
        }
        restore_started.set(true);
        let client = api.read().clone();
        let mut api = api;
        let mut current_user = current_user;
        let mut session_ready = session_ready;
        spawn(async move {
            if client.has_saved_token() {
                match client.me().await {
                    Ok(user) => current_user.set(Some(user)),
                    Err(ApiError::Http { status: 401, .. }) => {
                        let _ = client.logout();
                        api.set(client);
                        current_user.set(None);
                    }
                    Err(_) => {
                        client.invalidate_session();
                        api.set(client);
                        current_user.set(None);
                    }
                }
            } else {
                current_user.set(None);
            }
            session_ready.set(true);
        });
    });

    rsx! {
        style { { CSS } }
        if *session_ready.read() {
            Router::<Route> {}
        } else {
            main { class: "container", p { "Restoring session…" } }
        }
    }
}

const CSS: &str = r#"
* {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
}
body {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
    background: #f0f2f5;
    color: #333;
    line-height: 1.6;
}
.container {
    max-width: 960px;
    margin: 0 auto;
    padding: 20px;
}
h1 {
    color: #1a73e8;
    margin-bottom: 10px;
}
h2 {
    color: #444;
    margin: 20px 0 10px;
}
h3 {
    color: #555;
    margin-bottom: 8px;
}
.card {
    background: white;
    border-radius: 8px;
    padding: 16px;
    margin: 10px 0;
    box-shadow: 0 1px 3px rgba(0,0,0,0.1);
}
input, textarea, select {
    width: 100%;
    padding: 8px 12px;
    margin: 4px 0;
    border: 1px solid #ddd;
    border-radius: 4px;
    font-size: 14px;
}
button {
    background: #1a73e8;
    color: white;
    border: none;
    padding: 8px 16px;
    border-radius: 4px;
    cursor: pointer;
    font-size: 14px;
    margin: 4px;
}
button:hover {
    background: #1557b0;
}
button:disabled {
    background: #ccc;
    cursor: not-allowed;
}
.error {
    background: #fde8e8;
    color: #c53030;
    padding: 8px 12px;
    border-radius: 4px;
    margin: 8px 0;
}
table {
    width: 100%;
    border-collapse: collapse;
    margin: 10px 0;
    background: white;
    border-radius: 8px;
    overflow: hidden;
    box-shadow: 0 1px 3px rgba(0,0,0,0.1);
}
th, td {
    padding: 10px 12px;
    text-align: left;
    border-bottom: 1px solid #eee;
}
th {
    background: #f8f9fa;
    font-weight: 600;
    color: #555;
}
.positive {
    color: #2e7d32;
}
.negative {
    color: #c62828;
}
.tabs {
    display: flex;
    gap: 4px;
    margin: 16px 0;
}
.tabs button {
    flex: 1;
    background: #e0e0e0;
    color: #555;
    border-radius: 4px 4px 0 0;
}
.tabs button.active {
    background: #1a73e8;
    color: white;
}
@media (max-width: 600px) {
    .tabs {
        display: grid;
        grid-template-columns: repeat(2, minmax(0, 1fr));
    }
    .tabs button {
        min-width: 0;
    }
}
.section {
    margin: 16px 0;
}
.note {
    margin: 12px 0;
}
.note small {
    color: #888;
}
br {
    display: block;
    margin: 4px 0;
}
"#;
