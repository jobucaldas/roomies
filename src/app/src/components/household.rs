use crate::api::ApiClient;
use crate::models::*;
use dioxus::prelude::*;

#[derive(Clone, Debug, PartialEq)]
enum LoadState {
    Loading,
    Ready,
    Error(String),
}

fn writable(role: &str) -> bool {
    role != "monitor"
}

fn owns_or_admin(role: &str, creator_id: &str, user_id: &str) -> bool {
    role == "admin" || creator_id == user_id
}

fn input_time(value: &str) -> String {
    value.get(..16).unwrap_or(value).to_string()
}

fn local_time(value: &str) -> String {
    format!("{value}:00")
}

fn recurrence_request(
    title: &str,
    start: &str,
    zone: &str,
    frequency: &str,
    interval: &str,
    count: &str,
    exdates: &str,
) -> Result<(String, Vec<String>), String> {
    if title.trim().is_empty() || title.chars().count() > 160 {
        return Err("A title of 1–160 characters is required.".into());
    }
    if start.len() != 16 || zone.trim().is_empty() {
        return Err("Local start and IANA timezone are required.".into());
    }
    let interval = interval
        .parse::<u16>()
        .map_err(|_| "Interval must be 1–366.".to_string())?;
    if !(1..=366).contains(&interval) {
        return Err("Interval must be 1–366.".into());
    }
    let count = count
        .parse::<u16>()
        .map_err(|_| "Count must be 1–366.".to_string())?;
    if !(1..=366).contains(&count) {
        return Err("Count must be 1–366.".into());
    }
    let dates = exdates
        .split(',')
        .map(str::trim)
        .filter(|value| !value.is_empty())
        .map(String::from)
        .collect::<Vec<_>>();
    if dates.len() > 100 {
        return Err("At most 100 exception dates are allowed.".into());
    }
    Ok((
        format!("FREQ={frequency};INTERVAL={interval};COUNT={count}"),
        dates,
    ))
}

fn recurrence_fields(rule: &str) -> (String, String, String) {
    let value = |prefix| {
        rule.split(';')
            .find_map(|part| part.strip_prefix(prefix))
            .unwrap_or("")
            .to_string()
    };
    let interval = value("INTERVAL=");
    (
        value("FREQ="),
        if interval.is_empty() {
            "1".into()
        } else {
            interval
        },
        value("COUNT="),
    )
}

fn assignee_name(id: &Option<String>, members: &[HouseMember]) -> String {
    id.as_ref()
        .and_then(|id| members.iter().find(|member| &member.user_id == id))
        .map(|member| member.user_name.clone())
        .unwrap_or_else(|| "Unassigned".into())
}

#[component]
pub fn GroceriesSection(house_id: String, role: String, members: Vec<HouseMember>) -> Element {
    let api = use_context::<Signal<ApiClient>>();
    let mut items = use_signal(Vec::<GroceryItem>::new);
    let mut state = use_signal(|| LoadState::Loading);
    let mut name = use_signal(String::new);
    let mut quantity = use_signal(String::new);
    let mut unit = use_signal(String::new);
    let mut note = use_signal(String::new);
    let mut assignee = use_signal(String::new);
    let mut editing = use_signal(|| None::<GroceryItem>);
    let mut status = use_signal(String::new);
    let load_house = house_id.clone();
    use_effect(move || {
        if !matches!(*state.read(), LoadState::Loading) {
            return;
        }
        let api = api.read().clone();
        let house = load_house.clone();
        spawn(async move {
            match api.get_groceries(&house).await {
                Ok(values) => {
                    items.set(values);
                    state.set(LoadState::Ready);
                }
                Err(error) => state.set(LoadState::Error(error.to_string())),
            }
        });
    });
    let save_house = house_id.clone();
    let save = move |_| {
        if name.read().trim().is_empty() {
            status.set("A grocery name is required.".into());
            return;
        }
        let current = editing();
        let assignee_id = if assignee().is_empty() {
            None
        } else {
            Some(assignee())
        };
        let request = GroceryRequest {
            name: name(),
            quantity: quantity(),
            unit: unit(),
            note: note(),
            assignee_id,
            position: current.as_ref().map_or(0, |item| item.position),
            version: current.as_ref().map(|item| item.version),
        };
        let api = api.read().clone();
        let house = save_house.clone();
        spawn(async move {
            let result = match current {
                Some(item) => api.update_grocery(&house, &item.id, &request).await,
                None => api.create_grocery(&house, &request).await,
            };
            match result {
                Ok(saved) => {
                    let mut values = items();
                    if let Some(index) = values.iter().position(|item| item.id == saved.id) {
                        values[index] = saved;
                    } else {
                        values.push(saved);
                    }
                    items.set(values);
                    editing.set(None);
                    name.set(String::new());
                    quantity.set(String::new());
                    unit.set(String::new());
                    note.set(String::new());
                    assignee.set(String::new());
                    status.set("Grocery saved.".into());
                }
                Err(error) => status.set(format!("Could not save grocery: {error}")),
            }
        });
    };
    let can_write = writable(&role);
    rsx! { section { h2 { "Groceries" }
        if matches!(*state.read(), LoadState::Loading) { p { role: "status", "Loading groceries…" } }
        if let LoadState::Error(error) = &*state.read() { p { class: "error", role: "alert", "Unable to load groceries: {error}" } button { onclick: move |_| state.set(LoadState::Loading), "Retry groceries" } }
        if matches!(*state.read(), LoadState::Ready) {
            if can_write { div { class: "card", h3 { if editing().is_some() { "Edit grocery" } else { "Add grocery" } }
                label { "Name", input { value: "{name}", oninput: move |event| name.set(event.value()) } }
                label { "Quantity", input { value: "{quantity}", oninput: move |event| quantity.set(event.value()) } }
                label { "Unit", input { value: "{unit}", oninput: move |event| unit.set(event.value()) } }
                label { "Note", input { value: "{note}", oninput: move |event| note.set(event.value()) } }
                label { "Assignee", select { value: "{assignee}", oninput: move |event| assignee.set(event.value()), option { value: "", "Unassigned" } for member in members.iter() { option { value: "{member.user_id}", "{member.user_name}" } } } }
                button { onclick: save, if editing().is_some() { "Save grocery" } else { "Add grocery" } }
                if editing().is_some() { button { onclick: move |_| { editing.set(None); status.set("Grocery editing cancelled.".into()); }, "Cancel edit" } }
                p { role: "status", "{status}" }
            } } else { p { "Your monitor role is view-only." } }
            if items.read().is_empty() { p { "No groceries yet." } }
            for (item, item_toggle, edit, item_delete, assigned, toggle_house, delete_house) in items.read().iter().cloned().map(|item| { let assigned = assignee_name(&item.assignee_id, &members); (item.clone(), item.clone(), item.clone(), item, assigned, house_id.clone(), house_id.clone()) }) { article { class: "card", h3 { "{item.name}" } p { "{item.quantity} {item.unit} · assigned to {assigned}" } if !item.note.is_empty() { p { "{item.note}" } } p { if item.checked { "Checked" } else { "Needed" } }
                if can_write { button { onclick: move |_| { let api = api.read().clone(); let item = item_toggle.clone(); let house = toggle_house.clone(); spawn(async move { match api.toggle_grocery(&house, &item.id, &GroceryToggleRequest { checked: !item.checked, version: item.version }).await { Ok(saved) => { let mut values = items(); if let Some(index) = values.iter().position(|value| value.id == saved.id) { values[index] = saved; items.set(values); } }, Err(error) => status.set(format!("Could not update grocery: {error}")), } }); }, if item.checked { "Uncheck" } else { "Check" } }
                    button { onclick: move |_| { name.set(edit.name.clone()); quantity.set(edit.quantity.clone()); unit.set(edit.unit.clone()); note.set(edit.note.clone()); assignee.set(edit.assignee_id.clone().unwrap_or_default()); editing.set(Some(edit.clone())); status.set("Editing grocery.".into()); }, "Edit" }
                    button { onclick: move |_| { let api = api.read().clone(); let item = item_delete.clone(); let house = delete_house.clone(); spawn(async move { match api.delete_grocery(&house, &item.id, item.version).await { Ok(()) => { items.set(items().into_iter().filter(|value| value.id != item.id).collect()); status.set("Grocery deleted.".into()); }, Err(error) => status.set(format!("Could not delete grocery: {error}")), } }); }, "Delete" }
                }
            } }
        }
    } }
}

#[component]
pub fn ChoresSection(
    house_id: String,
    role: String,
    members: Vec<HouseMember>,
    user_id: String,
) -> Element {
    let api = use_context::<Signal<ApiClient>>();
    let mut chores = use_signal(Vec::<Chore>::new);
    let mut state = use_signal(|| LoadState::Loading);
    let mut title = use_signal(String::new);
    let mut description = use_signal(String::new);
    let mut assignee = use_signal(String::new);
    let mut start = use_signal(String::new);
    let mut zone = use_signal(|| "UTC".to_string());
    let mut frequency = use_signal(|| "WEEKLY".to_string());
    let mut interval = use_signal(|| "1".to_string());
    let mut count = use_signal(|| "1".to_string());
    let mut exdates = use_signal(String::new);
    let mut enabled = use_signal(|| true);
    let mut occurrence = use_signal(String::new);
    let mut editing = use_signal(|| None::<Chore>);
    let mut status = use_signal(String::new);
    let load_house = house_id.clone();
    use_effect(move || {
        if !matches!(*state.read(), LoadState::Loading) {
            return;
        }
        let api = api.read().clone();
        let house = load_house.clone();
        spawn(async move {
            match api.get_chores(&house).await {
                Ok(values) => {
                    chores.set(values);
                    state.set(LoadState::Ready);
                }
                Err(error) => state.set(LoadState::Error(error.to_string())),
            }
        });
    });
    let save_house = house_id.clone();
    let save = move |_| {
        let (rrule, dates) = match recurrence_request(
            &title(),
            &start(),
            &zone(),
            &frequency(),
            &interval(),
            &count(),
            &exdates(),
        ) {
            Ok(value) => value,
            Err(error) => {
                status.set(error);
                return;
            }
        };
        let current = editing();
        let assignee_id = if assignee().is_empty() {
            None
        } else {
            Some(assignee())
        };
        let request = ChoreRequest {
            title: title(),
            description: description(),
            assignee_id,
            timezone: zone(),
            due_local: local_time(&start()),
            rrule,
            exdates: dates,
            enabled: Some(enabled()),
            version: current.as_ref().map(|item| item.version),
        };
        let api = api.read().clone();
        let house = save_house.clone();
        spawn(async move {
            let result = match current {
                Some(item) => api.update_chore(&house, &item.id, &request).await,
                None => api.create_chore(&house, &request).await,
            };
            match result {
                Ok(saved) => {
                    let mut values = chores();
                    if let Some(index) = values.iter().position(|item| item.id == saved.id) {
                        values[index] = saved;
                    } else {
                        values.push(saved);
                    }
                    chores.set(values);
                    editing.set(None);
                    title.set(String::new());
                    description.set(String::new());
                    assignee.set(String::new());
                    start.set(String::new());
                    exdates.set(String::new());
                    enabled.set(true);
                    status.set("Chore saved.".into());
                }
                Err(error) => status.set(format!("Could not save chore: {error}")),
            }
        });
    };
    let can_write = writable(&role);
    rsx! { section { h2 { "Chores" }
        if matches!(*state.read(), LoadState::Loading) { p { role: "status", "Loading chores…" } }
        if let LoadState::Error(error) = &*state.read() { p { class: "error", role: "alert", "Unable to load chores: {error}" } button { onclick: move |_| state.set(LoadState::Loading), "Retry chores" } }
        if matches!(*state.read(), LoadState::Ready) {
            if can_write { div { class: "card", h3 { if editing().is_some() { "Edit chore" } else { "Add chore" } }
                label { "Title", input { value: "{title}", oninput: move |event| title.set(event.value()) } } label { "Description", textarea { value: "{description}", oninput: move |event| description.set(event.value()) } }
                label { "Assignee", select { value: "{assignee}", oninput: move |event| assignee.set(event.value()), option { value: "", "Unassigned" } for member in members.iter() { option { value: "{member.user_id}", "{member.user_name}" } } } }
                label { "Due local", input { r#type: "datetime-local", value: "{start}", oninput: move |event| start.set(event.value()) } } label { "IANA timezone", input { value: "{zone}", oninput: move |event| zone.set(event.value()) } }
                label { "Frequency", select { value: "{frequency}", oninput: move |event| frequency.set(event.value()), option { value: "DAILY", "Daily" } option { value: "WEEKLY", "Weekly" } option { value: "MONTHLY", "Monthly" } } } label { "Interval (1–366)", input { r#type: "number", value: "{interval}", oninput: move |event| interval.set(event.value()) } } label { "Count (1–366)", input { r#type: "number", value: "{count}", oninput: move |event| count.set(event.value()) } } label { "EXDATE local times (comma-separated)", input { value: "{exdates}", oninput: move |event| exdates.set(event.value()) } } label { input { r#type: "checkbox", checked: enabled(), oninput: move |event| enabled.set(event.checked()) } " Enabled" }
                button { onclick: save, if editing().is_some() { "Save chore" } else { "Create chore" } } if editing().is_some() { button { onclick: move |_| { editing.set(None); status.set("Chore editing cancelled.".into()); }, "Cancel edit" } } p { role: "status", "{status}" }
            } } else { p { "Your monitor role is view-only." } }
            if chores.read().is_empty() { p { "No chores yet." } }
            for (chore, edit, chore_update, chore_delete, chore_complete, can_manage, update_house, delete_house, complete_house, exception_text) in chores.read().iter().cloned().map(|chore| { let can_manage = owns_or_admin(&role, &chore.creator_id, &user_id); let exception_text = chore.exdates.join(", "); (chore.clone(), chore.clone(), chore.clone(), chore.clone(), chore, can_manage, house_id.clone(), house_id.clone(), house_id.clone(), exception_text) }) { article { class: "card", h3 { "{chore.title}" } p { "Due {chore.due_local} · {chore.timezone} · {chore.rrule}" } p { if chore.enabled { "Enabled" } else { "Disabled" } } if !chore.description.is_empty() { p { "{chore.description}" } } if !chore.exdates.is_empty() { p { "Exceptions: {exception_text}" } }
                if can_manage { button { onclick: move |_| { title.set(edit.title.clone()); description.set(edit.description.clone()); assignee.set(edit.assignee_id.clone().unwrap_or_default()); start.set(input_time(&edit.due_local)); zone.set(edit.timezone.clone()); let fields = recurrence_fields(&edit.rrule); frequency.set(fields.0); interval.set(fields.1); count.set(fields.2); exdates.set(edit.exdates.join(", ")); enabled.set(edit.enabled); editing.set(Some(edit.clone())); status.set("Editing chore.".into()); }, "Edit" }
                    button { onclick: move |_| { let api = api.read().clone(); let chore = chore_update.clone(); let house = update_house.clone(); spawn(async move { let request = ChoreRequest { title: chore.title.clone(), description: chore.description.clone(), assignee_id: chore.assignee_id.clone(), timezone: chore.timezone.clone(), due_local: chore.due_local.clone(), rrule: chore.rrule.clone(), exdates: chore.exdates.clone(), enabled: Some(!chore.enabled), version: Some(chore.version) }; match api.update_chore(&house, &chore.id, &request).await { Ok(saved) => { let mut values = chores(); if let Some(index) = values.iter().position(|item| item.id == saved.id) { values[index] = saved; chores.set(values); } status.set("Chore enabled state saved.".into()); }, Err(error) => status.set(format!("Could not update chore: {error}")), } }); }, if chore.enabled { "Disable" } else { "Enable" } }
                    button { onclick: move |_| { let api = api.read().clone(); let chore = chore_delete.clone(); let house = delete_house.clone(); spawn(async move { match api.delete_chore(&house, &chore.id, chore.version).await { Ok(()) => { chores.set(chores().into_iter().filter(|item| item.id != chore.id).collect()); status.set("Chore deleted.".into()); }, Err(error) => status.set(format!("Could not delete chore: {error}")), } }); }, "Delete" }
                }
                if can_write { label { "Occurrence UTC (RFC3339)", input { value: "{occurrence}", placeholder: "2025-02-01T09:00:00Z", oninput: move |event| occurrence.set(event.value()) } } button { onclick: move |_| { let value = occurrence(); if !value.ends_with('Z') { status.set("Use an exact UTC occurrence in RFC3339 form (ending in Z).".into()); return; } let api = api.read().clone(); let chore = chore_complete.clone(); let house = complete_house.clone(); spawn(async move { match api.complete_chore(&house, &chore.id, &CompleteChoreRequest { occurrence_at: value }).await { Ok(_) => status.set("Chore occurrence completed.".into()), Err(error) => status.set(format!("Could not complete occurrence: {error}")), } }); }, "Complete occurrence" } }
            } }
        }
    } }
}

#[component]
pub fn CalendarSection(house_id: String, role: String, user_id: String) -> Element {
    let api = use_context::<Signal<ApiClient>>();
    let mut events = use_signal(Vec::<CalendarEvent>::new);
    let mut state = use_signal(|| LoadState::Loading);
    let mut title = use_signal(String::new);
    let mut description = use_signal(String::new);
    let mut start = use_signal(String::new);
    let mut end = use_signal(String::new);
    let mut zone = use_signal(|| "UTC".to_string());
    let mut all_day = use_signal(|| false);
    let mut frequency = use_signal(|| "WEEKLY".to_string());
    let mut interval = use_signal(|| "1".to_string());
    let mut count = use_signal(|| "1".to_string());
    let mut exdates = use_signal(String::new);
    let mut editing = use_signal(|| None::<CalendarEvent>);
    let mut status = use_signal(String::new);
    let load_house = house_id.clone();
    use_effect(move || {
        if !matches!(*state.read(), LoadState::Loading) {
            return;
        }
        let api = api.read().clone();
        let house = load_house.clone();
        spawn(async move {
            match api.get_calendar(&house).await {
                Ok(values) => {
                    events.set(values);
                    state.set(LoadState::Ready);
                }
                Err(error) => state.set(LoadState::Error(error.to_string())),
            }
        });
    });
    let save_house = house_id.clone();
    let save = move |_| {
        let (rrule, dates) = match recurrence_request(
            &title(),
            &start(),
            &zone(),
            &frequency(),
            &interval(),
            &count(),
            &exdates(),
        ) {
            Ok(value) => value,
            Err(error) => {
                status.set(error);
                return;
            }
        };
        if end.read().len() != 16 {
            status.set("An end time is required, including for all-day events.".into());
            return;
        }
        let current = editing();
        let request = CalendarEventRequest {
            title: title(),
            description: description(),
            timezone: zone(),
            start_local: local_time(&start()),
            end_local: local_time(&end()),
            all_day: all_day(),
            rrule,
            exdates: dates,
            version: current.as_ref().map(|item| item.version),
        };
        let api = api.read().clone();
        let house = save_house.clone();
        spawn(async move {
            let result = match current {
                Some(event) => api.update_calendar(&house, &event.id, &request).await,
                None => api.create_calendar(&house, &request).await,
            };
            match result {
                Ok(saved) => {
                    let mut values = events();
                    if let Some(index) = values.iter().position(|event| event.id == saved.id) {
                        values[index] = saved;
                    } else {
                        values.push(saved);
                    }
                    events.set(values);
                    editing.set(None);
                    title.set(String::new());
                    description.set(String::new());
                    start.set(String::new());
                    end.set(String::new());
                    exdates.set(String::new());
                    status.set("Calendar event saved.".into());
                }
                Err(error) => status.set(format!("Could not save calendar event: {error}")),
            }
        });
    };
    let can_write = writable(&role);
    rsx! {
        section {
            h2 { "Calendar" }
            p { "Upcoming events are an accessible day-oriented list." }
            if matches!(*state.read(), LoadState::Loading) { p { role: "status", "Loading calendar…" } }
            if let LoadState::Error(error) = &*state.read() {
                p { class: "error", role: "alert", "Unable to load calendar: {error}" }
                button { onclick: move |_| state.set(LoadState::Loading), "Retry calendar" }
            }
            if matches!(*state.read(), LoadState::Ready) {
                if can_write {
                    div { class: "card",
                        h3 { if editing().is_some() { "Edit calendar event" } else { "Add calendar event" } }
                        label { "Title", input { value: "{title}", oninput: move |event| title.set(event.value()) } }
                        label { "Description", textarea { value: "{description}", oninput: move |event| description.set(event.value()) } }
                        label { input { r#type: "checkbox", checked: all_day(), oninput: move |event| all_day.set(event.checked()) } " All day" }
                        label { "Start local", input { r#type: "datetime-local", value: "{start}", oninput: move |event| start.set(event.value()) } }
                        label { "End local", input { r#type: "datetime-local", value: "{end}", oninput: move |event| end.set(event.value()) } }
                        label { "IANA timezone", input { value: "{zone}", oninput: move |event| zone.set(event.value()) } }
                        label { "Frequency", select { value: "{frequency}", oninput: move |event| frequency.set(event.value()), option { value: "DAILY", "Daily" } option { value: "WEEKLY", "Weekly" } option { value: "MONTHLY", "Monthly" } } }
                        label { "Interval (1–366)", input { r#type: "number", value: "{interval}", oninput: move |event| interval.set(event.value()) } }
                        label { "Count (1–366)", input { r#type: "number", value: "{count}", oninput: move |event| count.set(event.value()) } }
                        label { "EXDATE local times (comma-separated)", input { value: "{exdates}", oninput: move |event| exdates.set(event.value()) } }
                        button { onclick: save, if editing().is_some() { "Save calendar event" } else { "Create calendar event" } }
                        if editing().is_some() { button { onclick: move |_| { editing.set(None); status.set("Calendar editing cancelled.".into()); }, "Cancel edit" } }
                        p { role: "status", "{status}" }
                    }
                } else { p { "Your monitor role is view-only." } }
                if events.read().is_empty() { p { "No calendar events yet." } }
                for (event, edit, event_delete, can_manage, kind, exception_text, delete_house) in events.read().iter().cloned().map(|event| { let can_manage = owns_or_admin(&role, &event.creator_id, &user_id); let kind = if event.all_day { "all day" } else { "timed" }; let exception_text = event.exdates.join(", "); (event.clone(), event.clone(), event, can_manage, kind, exception_text, house_id.clone()) }) {
                    article { class: "card",
                        h3 { "{event.title}" }
                        p { "{event.start_local} to {event.end_local} · {kind} · {event.timezone}" }
                        p { "{event.rrule}" }
                        if !event.description.is_empty() { p { "{event.description}" } }
                        if !event.exdates.is_empty() { p { "Exceptions: {exception_text}" } }
                        if can_manage {
                            button { onclick: move |_| { title.set(edit.title.clone()); description.set(edit.description.clone()); start.set(input_time(&edit.start_local)); end.set(input_time(&edit.end_local)); zone.set(edit.timezone.clone()); all_day.set(edit.all_day); let fields = recurrence_fields(&edit.rrule); frequency.set(fields.0); interval.set(fields.1); count.set(fields.2); exdates.set(edit.exdates.join(", ")); editing.set(Some(edit.clone())); status.set("Editing calendar event.".into()); }, "Edit" }
                            button { onclick: move |_| { let api = api.read().clone(); let event = event_delete.clone(); let house = delete_house.clone(); spawn(async move { match api.delete_calendar(&house, &event.id, event.version).await { Ok(()) => { events.set(events().into_iter().filter(|value| value.id != event.id).collect()); status.set("Calendar event deleted.".into()); }, Err(error) => status.set(format!("Could not delete calendar event: {error}")), } }); }, "Delete" }
                        }
                    }
                }
            }
        }
    }
}

#[component]
pub fn ChatSection(house_id: String, role: String, user_id: String) -> Element {
    let api = use_context::<Signal<ApiClient>>();
    let mut messages = use_signal(Vec::<ChatMessage>::new);
    let mut next = use_signal(String::new);
    let mut state = use_signal(|| LoadState::Loading);
    let mut body = use_signal(String::new);
    let mut editing = use_signal(|| None::<ChatMessage>);
    let mut status = use_signal(String::new);
    let load_house = house_id.clone();
    use_effect(move || {
        if !matches!(*state.read(), LoadState::Loading) {
            return;
        }
        let api = api.read().clone();
        let house = load_house.clone();
        spawn(async move {
            match api.get_chat(&house, None).await {
                Ok(page) => {
                    messages.set(page.messages);
                    next.set(page.next_cursor);
                    state.set(LoadState::Ready);
                }
                Err(error) => state.set(LoadState::Error(error.to_string())),
            }
        });
    });
    let send_house = house_id.clone();
    let send = move |_| {
        if body.read().trim().is_empty() {
            status.set("A message is required.".into());
            return;
        }
        let current = editing();
        let request = ChatMessageRequest { body: body() };
        let api = api.read().clone();
        let house = send_house.clone();
        spawn(async move {
            let result = match current {
                Some(message) => api.update_chat(&house, &message.id, &request).await,
                None => api.create_chat(&house, &request).await,
            };
            match result {
                Ok(saved) => {
                    let mut values = messages();
                    if let Some(index) = values.iter().position(|message| message.id == saved.id) {
                        values[index] = saved;
                        status.set("Message edited.".into());
                    } else {
                        values.insert(0, saved);
                        status.set("Message sent.".into());
                    }
                    messages.set(values);
                    body.set(String::new());
                    editing.set(None);
                }
                Err(error) => status.set(format!("Could not save message: {error}")),
            }
        });
    };
    let refresh_house = house_id.clone();
    let refresh = move |_| {
        let api = api.read().clone();
        let house = refresh_house.clone();
        spawn(async move {
            match api.get_chat(&house, None).await {
                Ok(page) => {
                    messages.set(page.messages);
                    next.set(page.next_cursor);
                    status.set("Chat refreshed.".into());
                }
                Err(error) => status.set(format!("Could not refresh chat: {error}")),
            }
        });
    };
    let older_house = house_id.clone();
    let older = move |_| {
        let before = match next().parse::<i64>() {
            Ok(value) => value,
            Err(_) => {
                status.set("The older-message cursor was invalid; refresh chat.".into());
                return;
            }
        };
        let api = api.read().clone();
        let house = older_house.clone();
        spawn(async move {
            match api.get_chat(&house, Some(before)).await {
                Ok(page) => {
                    let mut values = messages();
                    for message in page.messages {
                        if !values.iter().any(|existing| existing.id == message.id) {
                            values.push(message);
                        }
                    }
                    next.set(page.next_cursor);
                    messages.set(values);
                    status.set("Older messages loaded.".into());
                }
                Err(error) => status.set(format!("Could not load older messages: {error}")),
            }
        });
    };
    let can_write = writable(&role);
    rsx! {
        section {
            h2 { "Chat" }
            p { "Use Refresh to fetch updates; messages are never queued for offline sending or persisted by this client." }
            if matches!(*state.read(), LoadState::Loading) { p { role: "status", "Loading chat…" } }
            if let LoadState::Error(error) = &*state.read() { p { class: "error", role: "alert", "Unable to load chat: {error}" } button { onclick: move |_| state.set(LoadState::Loading), "Retry chat" } }
            if matches!(*state.read(), LoadState::Ready) {
                button { onclick: refresh, "Refresh chat" }
                if can_write { div { class: "card", h3 { if editing().is_some() { "Edit message" } else { "Send message" } } label { "Message", textarea { value: "{body}", oninput: move |event| body.set(event.value()) } } button { onclick: send, if editing().is_some() { "Save message" } else { "Send message" } } if editing().is_some() { button { onclick: move |_| { editing.set(None); body.set(String::new()); status.set("Message editing cancelled.".into()); }, "Cancel edit" } } } } else { p { "Your monitor role is view-only." } }
                if !next().is_empty() { button { onclick: older, "Load older messages" } }
                if messages.read().is_empty() { p { "No chat messages yet." } }
                for (message, edit, message_delete, own, delete_house) in messages.read().iter().cloned().map(|message| { let own = message.author_id == user_id; (message.clone(), message.clone(), message, own, house_id.clone()) }) {
                    article { class: "card", small { "{message.created_at}" }
                        if message.deleted_at.is_some() { p { "This message was deleted." } } else if message.redacted_at.is_some() { p { "This message was redacted." } } else if let Some(text) = message.body.as_ref() { p { "{text}" } } else { p { "This message is unavailable." } }
                        if can_write && own && message.deleted_at.is_none() && message.redacted_at.is_none() { button { onclick: move |_| { body.set(edit.body.clone().unwrap_or_default()); editing.set(Some(edit.clone())); status.set("Editing message.".into()); }, "Edit message" } button { onclick: move |_| { let api = api.read().clone(); let message = message_delete.clone(); let house = delete_house.clone(); spawn(async move { match api.delete_chat(&house, &message.id).await { Ok(()) => { let mut values = messages(); if let Some(index) = values.iter().position(|value| value.id == message.id) { values[index].body = None; values[index].deleted_at = Some("deleted".into()); messages.set(values); } status.set("Message deleted.".into()); }, Err(error) => status.set(format!("Could not delete message: {error}")), } }); }, "Delete message" } }
                    }
                }
                p { role: "status", "{status}" }
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn recurrence_enforces_client_bounds_and_preserves_exdates() {
        assert!(recurrence_request(
            "Bins",
            "2025-01-02T09:00",
            "UTC",
            "WEEKLY",
            "1",
            "2",
            "2025-01-09T09:00"
        )
        .is_ok());
        assert!(
            recurrence_request("Bins", "2025-01-02T09:00", "UTC", "WEEKLY", "367", "1", "")
                .is_err()
        );
        assert!(
            recurrence_request("Bins", "2025-01-02T09:00", "UTC", "WEEKLY", "1", "367", "")
                .is_err()
        );
    }
    #[test]
    fn recurrence_edit_fields_default_interval() {
        assert_eq!(
            recurrence_fields("FREQ=WEEKLY;COUNT=2"),
            ("WEEKLY".into(), "1".into(), "2".into())
        );
    }
    #[test]
    fn mutations_are_withheld_for_monitors() {
        assert!(!writable("monitor"));
        assert!(writable("member"));
        assert!(owns_or_admin("admin", "other", "me"));
        assert!(owns_or_admin("member", "me", "me"));
        assert!(!owns_or_admin("member", "other", "me"));
    }
}
