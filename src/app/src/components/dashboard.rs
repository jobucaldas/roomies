use crate::api::ApiClient;
use crate::models::{House, User};
use crate::router::Route;
use dioxus::prelude::*;
use dioxus::prelude::{use_navigator, Link};

#[component]
pub fn Dashboard() -> Element {
    let router = use_navigator();
    let mut api = use_context::<Signal<ApiClient>>();
    let mut current_user = use_context::<Signal<Option<User>>>();
    let mut houses = use_signal(Vec::<House>::new);
    let mut new_house_name = use_signal(String::new);
    let mut error = use_signal(String::new);
    let mut loading = use_signal(|| false);
    let authenticated = api.read().is_authenticated();

    use_effect(move || {
        if !authenticated {
            router.replace(Route::Login {});
        }
    });

    let mut fetch = move || {
        if !authenticated {
            return;
        }
        loading.set(true);
        spawn(async move {
            let api = api.read().cloned();
            match api.get_houses().await {
                Ok(h) => houses.set(h),
                Err(e) => error.set(e.to_string()),
            }
            loading.set(false);
        });
    };

    use_effect(move || {
        fetch();
    });

    let create_house = move |_| {
        let name = new_house_name.read().clone();
        if name.is_empty() {
            return;
        }
        loading.set(true);
        error.set(String::new());
        spawn(async move {
            let api = api.read().cloned();
            match api.create_house(&name).await {
                Ok(_) => {
                    new_house_name.set(String::new());
                    loading.set(false);
                    fetch();
                }
                Err(e) => {
                    error.set(e.to_string());
                    loading.set(false);
                }
            }
        });
    };

    if !authenticated {
        return rsx! {
            div { class: "container",
                p { "Redirecting to login…" }
            }
        };
    }

    rsx! {
        div { class: "container",
            h1 { "Dashboard" }
            p { "Welcome to Roomies!" }

            div { class: "card",
                h3 { "Create New House" }
                input {
                    placeholder: "House name",
                    value: "{new_house_name}",
                    oninput: move |e| new_house_name.set(e.value()),
                }
                button { onclick: create_house, "Create House" }
            }

            if !error.read().is_empty() {
                div { class: "error", "{error}" }
            }

            h2 { "Your Houses" }
            if *loading.read() {
                p { "Loading..." }
            } else if houses.read().is_empty() {
                p { "No houses yet. Create one above!" }
            } else {
                for house in houses.read().iter() {
                    div { class: "card",
                        h3 { "{house.name}" }
                        Link {
                            to: Route::HouseDetail { id: house.id.clone() },
                            "View House"
                        }
                    }
                }
            }

            br {}
            button { onclick: move |_| {
                api.write().logout();
                current_user.set(None);
                router.replace(Route::Login {});
            }, "Logout" }
        }
    }
}
