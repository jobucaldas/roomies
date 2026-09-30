use super::expenses::ExpensesSection;
use super::household::{CalendarSection, ChatSection, ChoresSection, GroceriesSection};
use super::invitations::InvitationPanel;
use super::notes::NotesSection;
use super::notifications::NotificationsSection;
use crate::api::ApiClient;
use crate::core::{can_manage, Role};
use crate::models::{BalanceResponse, House, HouseMember, User};
use dioxus::prelude::*;

#[component]
pub fn HouseDetail(id: String) -> Element {
    let navigator = use_navigator();
    let api = use_context::<Signal<ApiClient>>();
    let user = use_context::<Signal<Option<User>>>();
    let mut house = use_signal(|| Option::<House>::None);
    let mut members = use_signal(Vec::<HouseMember>::new);
    let mut balances = use_signal(|| Option::<BalanceResponse>::None);
    let mut tab = use_signal(|| "expenses".to_string());
    let mut error = use_signal(String::new);
    let mut loading = use_signal(|| true);
    let load_id = id.clone();
    let authenticated = api.read().is_authenticated();

    use_effect(move || {
        if !authenticated {
            navigator.replace(crate::router::Route::Login {});
        }
    });

    use_effect(move || {
        if !authenticated {
            return;
        }
        let api = api.read().cloned();
        let house_id = load_id.clone();
        spawn(async move {
            loading.set(true);
            match (
                api.get_house(&house_id).await,
                api.get_members(&house_id).await,
            ) {
                (Ok(home), Ok(list)) => {
                    house.set(Some(home));
                    members.set(list);
                }
                (Err(e), _) | (_, Err(e)) => error.set(e.to_string()),
            }
            loading.set(false);
        });
    });

    let role = user.read().as_ref().and_then(|u| {
        members
            .read()
            .iter()
            .find(|m| m.user_id == u.id)
            .and_then(|m| Role::parse(&m.role))
    });
    let admin = role.map(can_manage).unwrap_or(false);
    let role_name = match role {
        Some(Role::Admin) => "admin",
        Some(Role::Member) => "member",
        Some(Role::Monitor) | None => "monitor",
    }
    .to_string();

    if !authenticated {
        return rsx! {
            main { class: "container",
                p { "Redirecting to login…" }
            }
        };
    }

    rsx! {
        main { class: "container",
            if !error.read().is_empty() { p { class: "error", role: "alert", "{error}" } }
            if *loading.read() {
                p { "Loading house…" }
            } else {
                if let Some(home) = house.read().clone() {
                    h1 { "{home.name}" }
                    if admin {
                        HouseEditor {
                            house: home,
                            on_saved: move |updated| house.set(Some(updated))
                        }
                    }
                }
                div { class: "tabs", role: "tablist",
                    button {
                        role: "tab",
                        id: "expenses-tab",
                        aria_selected: *tab.read() == "expenses",
                        aria_controls: "expenses-panel",
                        class: if *tab.read() == "expenses" { "active" } else { "" },
                        onclick: move |_| tab.set("expenses".into()),
                        "Expenses"
                    }
                    button {
                        role: "tab",
                        id: "notes-tab",
                        aria_selected: *tab.read() == "notes",
                        aria_controls: "notes-panel",
                        class: if *tab.read() == "notes" { "active" } else { "" },
                        onclick: move |_| tab.set("notes".into()),
                        "Notes"
                    }
                    button {
                        role: "tab",
                        id: "groceries-tab",
                        aria_selected: *tab.read() == "groceries",
                        aria_controls: "groceries-panel",
                        class: if *tab.read() == "groceries" { "active" } else { "" },
                        onclick: move |_| tab.set("groceries".into()),
                        "Groceries"
                    }
                    button {
                        role: "tab",
                        id: "chores-tab",
                        aria_selected: *tab.read() == "chores",
                        aria_controls: "chores-panel",
                        class: if *tab.read() == "chores" { "active" } else { "" },
                        onclick: move |_| tab.set("chores".into()),
                        "Chores"
                    }
                    button {
                        role: "tab",
                        id: "calendar-tab",
                        aria_selected: *tab.read() == "calendar",
                        aria_controls: "calendar-panel",
                        class: if *tab.read() == "calendar" { "active" } else { "" },
                        onclick: move |_| tab.set("calendar".into()),
                        "Calendar"
                    }
                    button {
                        role: "tab",
                        id: "chat-tab",
                        aria_selected: *tab.read() == "chat",
                        aria_controls: "chat-panel",
                        class: if *tab.read() == "chat" { "active" } else { "" },
                        onclick: move |_| tab.set("chat".into()),
                        "Chat"
                    }
                    button {
                        role: "tab",
                        id: "balances-tab",
                        aria_selected: *tab.read() == "balances",
                        aria_controls: "balances-panel",
                        class: if *tab.read() == "balances" { "active" } else { "" },
                        onclick: {
                            let balances_id = id.clone();
                            move |_| {
                            tab.set("balances".into());
                            let api = api.read().cloned();
                            let house_id = balances_id.clone();
                            spawn(async move {
                                match api.get_balances(&house_id).await {
                                    Ok(value) => balances.set(Some(value)),
                                    Err(e) => error.set(e.to_string()),
                                }
                            });
                            }
                        },
                        "Balances"
                    }
                    button {
                        role: "tab",
                        id: "notifications-tab",
                        aria_selected: *tab.read() == "notifications",
                        aria_controls: "notifications-panel",
                        class: if *tab.read() == "notifications" { "active" } else { "" },
                        onclick: move |_| tab.set("notifications".into()),
                        "Notifications / Schedule"
                    }
                    button {
                        role: "tab",
                        id: "members-tab",
                        aria_selected: *tab.read() == "members",
                        aria_controls: "members-panel",
                        class: if *tab.read() == "members" { "active" } else { "" },
                        onclick: move |_| tab.set("members".into()),
                        "Members"
                    }
                }
                div {
                    role: "tabpanel",
                    id: "{tab.read()}-panel",
                    aria_labelledby: "{tab.read()}-tab",
                    match tab.read().as_str() {
                        "expenses" => rsx! { ExpensesSection { house_id: id.clone() } },
                        "notes" => rsx! {
                            NotesSection { house_id: id.clone(), role: role_name.clone() }
                        },
                        "groceries" => rsx! { GroceriesSection { house_id: id.clone(), role: role_name.clone(), members: members.read().clone() } },
                        "chores" => rsx! { ChoresSection { house_id: id.clone(), role: role_name.clone(), members: members.read().clone(), user_id: user.read().as_ref().map(|u| u.id.clone()).unwrap_or_default() } },
                        "calendar" => rsx! { CalendarSection { house_id: id.clone(), role: role_name.clone(), user_id: user.read().as_ref().map(|u| u.id.clone()).unwrap_or_default() } },
                        "chat" => rsx! { ChatSection { house_id: id.clone(), role: role_name.clone(), user_id: user.read().as_ref().map(|u| u.id.clone()).unwrap_or_default() } },
                        "balances" => rsx! { BalancesTab { balances: balances.read().clone() } },
                        "notifications" => rsx! {
                            NotificationsSection { house_id: id.clone(), role: role_name.clone(), user_id: user.read().as_ref().map(|u| u.id.clone()).unwrap_or_default() }
                        },
                        "members" => rsx! {
                            MembersTab {
                                house_id: id.clone(),
                                members: members.read().clone(),
                                admin,
                                on_refresh: {
                                    let members_id = id.clone();
                                    move |_| {
                                    let api = api.read().cloned();
                                    let house_id = members_id.clone();
                                    spawn(async move {
                                        match api.get_members(&house_id).await {
                                            Ok(value) => members.set(value),
                                            Err(e) => error.set(e.to_string()),
                                        }
                                    });
                                    }
                                }
                            }
                        },
                        _ => rsx! {},
                    }
                }
            }
            button { onclick: move |_| navigator.go_back(), "Back" }
        }
    }
}

#[component]
fn HouseEditor(house: House, on_saved: EventHandler<House>) -> Element {
    let api = use_context::<Signal<ApiClient>>();
    let mut name = use_signal(|| house.name.clone());
    let mut status = use_signal(String::new);
    let save = move |_| {
        if name.read().trim().is_empty() {
            status.set("House name is required".into());
            return;
        }
        let api = api.read().cloned();
        let id = house.id.clone();
        let name = name.read().trim().to_string();
        spawn(async move {
            match api.update_house(&id, &name).await {
                Ok(updated) => {
                    status.set("House updated.".into());
                    on_saved.call(updated);
                }
                Err(e) => status.set(e.to_string()),
            }
        });
    };
    rsx! {
        div { class: "card",
            h2 { "House settings" }
            label { "Name", input { value: "{name}", oninput: move |e| name.set(e.value()) } }
            button { onclick: save, "Save house" }
            p { "{status}" }
        }
    }
}

#[component]
fn BalancesTab(balances: Option<BalanceResponse>) -> Element {
    rsx! {
        section {
            h2 { "Balances" }
            p { "Balances refresh when this tab is selected." }
            if let Some(b) = balances {
                if b.balances.is_empty() {
                    p { "No expenses yet." }
                } else {
                    for entry in b.balances {
                        article { class: "card", h3 { "{entry.user_name}" }, p { "Paid ${entry.paid:.2}; owed ${entry.owed:.2}; net ${entry.net:.2}" } }
                    }
                    for settlement in b.settlements {
                        p { "{settlement.from_user_name} pays {settlement.to_user_name} ${settlement.amount:.2}" }
                    }
                }
            } else {
                p { "Select this tab to load balances. Former users remain identified by name." }
            }
        }
    }
}

#[component]
fn MembersTab(
    house_id: String,
    members: Vec<HouseMember>,
    admin: bool,
    on_refresh: EventHandler<()>,
) -> Element {
    let api = use_context::<Signal<ApiClient>>();
    let mut user_id = use_signal(String::new);
    let mut role = use_signal(|| "member".to_string());
    let mut status = use_signal(String::new);
    let mut invitations = use_signal(Vec::<crate::models::HouseInvitation>::new);
    let mut invitations_loading = use_signal(|| admin);
    let mut invitations_error = use_signal(String::new);
    {
        let api = api.read().cloned();
        let house_id = house_id.clone();
        use_effect(move || {
            if !admin {
                return;
            }
            let api = api.clone();
            let house_id = house_id.clone();
            spawn(async move {
                match api.list_invitations(&house_id).await {
                    Ok(value) => invitations.set(value),
                    Err(error) => invitations_error.set(error.to_string()),
                }
                invitations_loading.set(false);
            });
        });
    }
    let add_house_id = house_id.clone();
    let add = move |_| {
        if user_id.read().trim().is_empty() {
            status.set("User ID is required".into());
            return;
        }
        let api = api.read().cloned();
        let house_id = add_house_id.clone();
        let user_id = user_id.read().trim().to_string();
        let role = role.read().clone();
        spawn(async move {
            match api.add_member(&house_id, &user_id, &role).await {
                Ok(_) => {
                    status.set("Member added.".into());
                    on_refresh.call(());
                }
                Err(e) => status.set(e.to_string()),
            }
        });
    };
    let invitation_refresh_house_id = house_id.clone();
    rsx! {
        section {
            h2 { "Members" }
            if admin {
                InvitationPanel {
                    house_id: house_id.clone(),
                    invitations: invitations.read().clone(),
                    loading: *invitations_loading.read(),
                    error: invitations_error.read().clone(),
                    on_refresh: move |_| {
                        let api = api.read().cloned();
                        let house_id = invitation_refresh_house_id.clone();
                        spawn(async move {
                            match api.list_invitations(&house_id).await {
                                Ok(value) => invitations.set(value),
                                Err(error) => invitations_error.set(error.to_string()),
                            }
                        });
                    },
                }
                div { class: "card",
                    h3 { "Add existing member" }
                    label { "User ID", input { value: "{user_id}", oninput: move |e| user_id.set(e.value()) } }
                    select { value: "{role}", oninput: move |e| role.set(e.value()), option { value: "admin", "Admin" }, option { value: "member", "Member" }, option { value: "monitor", "Monitor" } }
                    button { onclick: add, "Add member" }
                    p { "{status}" }
                }
            }
            for member in members {
                article { class: "card",
                    h3 { "{member.user_name}" }
                    p { "{member.user_email} · {member.role}" }
                    if admin { MemberControls { house_id: house_id.clone(), member, on_refresh } }
                }
            }
        }
    }
}

#[component]
fn MemberControls(house_id: String, member: HouseMember, on_refresh: EventHandler<()>) -> Element {
    let api = use_context::<Signal<ApiClient>>();
    let mut role = use_signal(|| member.role.clone());
    let mut status = use_signal(String::new);
    let update_house_id = house_id.clone();
    let update_user_id = member.user_id.clone();
    let update = move |_| {
        let api = api.read().cloned();
        let house_id = update_house_id.clone();
        let user_id = update_user_id.clone();
        let role = role.read().clone();
        spawn(async move {
            match api.update_member_role(&house_id, &user_id, &role).await {
                Ok(response) => {
                    status.set(response.message);
                    on_refresh.call(());
                }
                Err(e) => status.set(e.to_string()),
            }
        });
    };
    let remove_house_id = house_id.clone();
    let remove_user_id = member.user_id.clone();
    let remove = move |_| {
        let api = api.read().cloned();
        let house_id = remove_house_id.clone();
        let user_id = remove_user_id.clone();
        spawn(async move {
            match api.remove_member(&house_id, &user_id).await {
                Ok(()) => {
                    status.set("Member removed.".into());
                    on_refresh.call(());
                }
                Err(e) => status.set(e.to_string()),
            }
        });
    };
    rsx! {
        div {
            select { value: "{role}", oninput: move |e| role.set(e.value()), option { value: "admin", "Admin" }, option { value: "member", "Member" }, option { value: "monitor", "Monitor" } }
            button { onclick: update, "Change role" }
            button { onclick: remove, "Remove" }
            p { "{status}" }
        }
    }
}
