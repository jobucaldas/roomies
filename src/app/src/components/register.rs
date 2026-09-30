use crate::api::ApiClient;
use crate::models::User;
use crate::router::Route;
use dioxus::prelude::*;
use dioxus::prelude::{use_navigator, Link};

#[component]
pub fn Register() -> Element {
    let router = use_navigator();
    let mut name = use_signal(String::new);
    let mut email = use_signal(String::new);
    let mut password = use_signal(String::new);
    let mut error = use_signal(String::new);
    let mut loading = use_signal(|| false);

    let mut api = use_context::<Signal<ApiClient>>();
    let mut current_user = use_context::<Signal<Option<User>>>();
    let saved_storage_error = api
        .read()
        .storage_error()
        .map(|error| error.to_string())
        .unwrap_or_default();

    let on_submit = move |event: Event<FormData>| {
        event.prevent_default();
        loading.set(true);
        error.set(String::new());
        let name = name.read().clone();
        let email = email.read().clone();
        let password = password.read().clone();
        spawn(async move {
            let mut client = api.write().clone();
            match client.register(&name, &email, &password).await {
                Ok(response) => {
                    current_user.set(Some(response.user));
                    api.set(client);
                    if crate::storage::pending_invitation().is_some() {
                        router.replace(Route::AcceptInvitation { token: None });
                    } else {
                        router.push("/dashboard");
                    }
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
            h2 { "Register" }

            if !error.read().is_empty() {
                div { class: "error", "{error}" }
            } else if !saved_storage_error.is_empty() {
                div { class: "error", "{saved_storage_error}" }
            }

            form { onsubmit: on_submit,
            input {
                placeholder: "Name",
                value: "{name}",
                oninput: move |e| name.set(e.value()),
            }
            br {}
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
                if *loading.read() { "Creating account..." } else { "Register" }
            }
            }
            br {}
            Link { to: Route::Login {}, "Already have an account? Login" }
        }
    }
}
