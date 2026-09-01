use crate::api::ApiClient;
use crate::models::User;
use crate::router::Route;
use dioxus::prelude::*;
use dioxus::prelude::{use_navigator, Link};

#[component]
pub fn Login() -> Element {
    let router = use_navigator();
    let mut email = use_signal(String::new);
    let mut password = use_signal(String::new);
    let mut error = use_signal(String::new);
    let mut loading = use_signal(|| false);

    let mut api = use_context::<Signal<ApiClient>>();
    let mut current_user = use_context::<Signal<Option<User>>>();

    let on_submit = move |_| {
        loading.set(true);
        error.set(String::new());
        let email = email.read().clone();
        let password = password.read().clone();
        spawn(async move {
            let mut client = api.write().clone();
            match client.login(&email, &password).await {
                Ok(response) => {
                    current_user.set(Some(response.user));
                    api.set(client);
                    router.push("/dashboard");
                }
                Err(e) => {
                    error.set(e.to_string());
                    loading.set(false);
                }
            }
        });
    };

    rsx! {
        div { class: "container",
            h1 { "Roomies" }
            h2 { "Login" }

            if !error.read().is_empty() {
                div { class: "error", "{error}" }
            }

            form { onsubmit: on_submit,
            input {
                r#type: "email",
                placeholder: "Email",
                value: "{email}",
                oninput: move |e| email.set(e.value()),
            }
            br {}
            input {
                r#type: "password",
                placeholder: "Password",
                value: "{password}",
                oninput: move |e| password.set(e.value()),
            }
            br {}
            button {
                r#type: "submit",
                disabled: *loading.read(),
                if *loading.read() { "Logging in..." } else { "Login" }
            }
            }
            br {}
            Link { to: Route::Register {}, "Don't have an account? Register" }
        }
    }
}
