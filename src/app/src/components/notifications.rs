use crate::api::ApiClient;
use crate::core::Role;
use crate::models::{
    NotificationPreferences, NotificationSubscription, ScheduledEventRequest, ScheduledHouseEvent,
};
use dioxus::prelude::*;

#[derive(Clone, Debug, PartialEq)]
enum NotificationLoadState {
    Loading,
    Ready,
    Error(String),
}

fn notification_controls_ready(state: &NotificationLoadState) -> bool {
    matches!(state, NotificationLoadState::Ready)
}

#[component]
pub fn NotificationsSection(house_id: String, role: String, user_id: String) -> Element {
    let api = use_context::<Signal<ApiClient>>();
    let mut preferences = use_signal(NotificationPreferences::default);
    let mut subscriptions = use_signal(Vec::<NotificationSubscription>::new);
    let mut events = use_signal(Vec::<ScheduledHouseEvent>::new);
    let mut load_state = use_signal(|| NotificationLoadState::Loading);
    let can_create = Role::parse(&role)
        .map(|value| value != Role::Monitor)
        .unwrap_or(false);
    let load_house = house_id.clone();
    use_effect(move || {
        if !matches!(*load_state.read(), NotificationLoadState::Loading) {
            return;
        }
        let api = api.read().cloned();
        let house_id = load_house.clone();
        spawn(async move {
            match (
                api.get_notification_preferences(&house_id).await,
                api.get_notification_subscriptions(&house_id).await,
                api.get_scheduled_events(&house_id).await,
            ) {
                (Ok(p), Ok(s), Ok(e)) => {
                    preferences.set(p);
                    subscriptions.set(s);
                    events.set(e);
                    load_state.set(NotificationLoadState::Ready);
                }
                (Err(error), _, _) | (_, Err(error), _) | (_, _, Err(error)) => {
                    load_state.set(NotificationLoadState::Error(error.to_string()))
                }
            }
        });
    });
    rsx! {
        section {
            h2 { "Notifications & schedule" }
            if notification_controls_ready(&load_state.read()) {
                PreferencesForm { house_id: house_id.clone(), preferences: preferences.read().clone(), on_saved: move |value| preferences.set(value) }
                BrowserPush { house_id: house_id.clone(), subscriptions: subscriptions.read().clone(), on_changed: move |value| subscriptions.set(value) }
                Schedule { house_id, user_id, events: events.read().clone(), can_create, admin: role == "admin" }
            } else if let NotificationLoadState::Error(error) = &*load_state.read() {
                p { role: "status", class: "error", "Unable to load notification settings: {error}" }
                button { onclick: move |_| load_state.set(NotificationLoadState::Loading), "Retry loading notification settings" }
            } else {
                p { role: "status", "Loading notification settings…" }
            }
        }
    }
}

#[component]
fn PreferencesForm(
    house_id: String,
    preferences: NotificationPreferences,
    on_saved: EventHandler<NotificationPreferences>,
) -> Element {
    let api = use_context::<Signal<ApiClient>>();
    let mut value = use_signal(|| preferences.clone());
    let preferences_for_effect = preferences.clone();
    use_effect(move || {
        if value().house_id != preferences_for_effect.house_id
            || value().user_id != preferences_for_effect.user_id
        {
            value.set(preferences_for_effect.clone());
        }
    });
    let mut status = use_signal(String::new);
    let save_house = house_id.clone();
    let save = move |_| {
        let candidate = value.read().clone();
        if candidate.timezone.trim().is_empty() {
            status.set("Use an IANA timezone and times between 00:00 and 23:59.".into());
            return;
        }
        let api = api.read().cloned();
        let house_id = save_house.clone();
        spawn(async move {
            match api
                .put_notification_preferences(&house_id, &candidate)
                .await
            {
                Ok(saved) => {
                    status.set("Notification preferences saved.".into());
                    on_saved.call(saved);
                }
                Err(e) => status.set(format!("Could not save preferences: {e}")),
            }
        });
    };
    rsx! { div { class: "card",
        h3 { "Your notification preferences" }
        label { input { r#type: "checkbox", checked: value.read().expense_created_enabled, oninput: move |event| { let mut v = value.read().clone(); v.expense_created_enabled = event.checked(); value.set(v); } } " Shared expense alerts" }
        label { input { r#type: "checkbox", checked: value.read().reminder_enabled, oninput: move |event| { let mut v = value.read().clone(); v.reminder_enabled = event.checked(); value.set(v); } } " Scheduled reminder alerts" }
        label { "Delivery", select { value: "{value.read().cadence}", onchange: move |e| { let mut v = value.read().clone(); v.cadence=e.value(); value.set(v); }, option { value: "immediate", "Immediate" } option { value: "daily_digest", "Daily digest" } } }
        label { "IANA timezone", input { value: "{value.read().timezone}", placeholder: "Europe/Lisbon", oninput: move |e| { let mut v=value.read().clone(); v.timezone=e.value(); value.set(v); } } }
        label { "Quiet start (local)", input { r#type: "time", value: "{minutes_time(value.read().quiet_start_minutes)}", oninput: move |e| { let mut v=value.read().clone(); v.quiet_start_minutes=time_minutes(&e.value()); value.set(v); } } }
        label { "Quiet end (local)", input { r#type: "time", value: "{minutes_time(value.read().quiet_end_minutes)}", oninput: move |e| { let mut v=value.read().clone(); v.quiet_end_minutes=time_minutes(&e.value()); value.set(v); } } }
        label { "Daily digest time (local)", input { r#type: "time", value: "{minutes_time(Some(value.read().digest_minutes))}", oninput: move |e| { let mut v=value.read().clone(); v.digest_minutes=time_minutes(&e.value()).unwrap_or(0); value.set(v); } } }
        button { onclick: save, "Save preferences" } p { role: "status", "{status}" }
    } }
}

#[component]
fn BrowserPush(
    house_id: String,
    subscriptions: Vec<NotificationSubscription>,
    on_changed: EventHandler<Vec<NotificationSubscription>>,
) -> Element {
    let api = use_context::<Signal<ApiClient>>();
    let mut status = use_signal(browser_push_state);
    let enable_house = house_id.clone();
    let enable = move |_| {
        let api = api.read().cloned();
        let house_id = enable_house.clone();
        spawn(async move {
            match crate::push::enable(&api, &house_id).await {
                Ok(()) => match api.get_notification_subscriptions(&house_id).await {
                    Ok(v) => {
                        on_changed.call(v);
                        status.set("Browser push enabled.".into());
                    }
                    Err(e) => status.set(format!("Subscription saved but refresh failed: {e}")),
                },
                Err(e) => status.set(e),
            }
        });
    };
    let disable_house = house_id.clone();
    let disable_subscriptions = subscriptions.clone();
    let disable = move |_| {
        let api = api.read().cloned();
        let house_id = disable_house.clone();
        let after = disable_subscriptions.clone();
        let ids: Vec<String> = after
            .iter()
            .filter(|s| s.platform == "web_push")
            .map(|s| s.id.clone())
            .collect();
        spawn(async move {
            match crate::push::disable(&api, &house_id, &ids).await {
                Ok(()) => {
                    on_changed.call(
                        after
                            .iter()
                            .filter(|s| s.platform != "web_push")
                            .cloned()
                            .collect(),
                    );
                    status.set("Browser push disabled.".into());
                }
                Err(e) => status.set(e),
            }
        });
    };
    let enabled = subscriptions.iter().any(|s| s.platform == "web_push");
    rsx! { div { class: "card", h3 { "Browser push" } p { "Browser push is only available in a supported web browser." }
        if enabled { button { onclick: disable, "Disable browser push" } } else { button { onclick: enable, "Enable browser push" } }
        p { role: "status", "{status}" }
        h4 { "Your devices" } for subscription in subscriptions { p { "{subscription.platform}: {subscription.device_label} (last seen {subscription.last_seen_at})" } }
    } }
}

#[component]
fn Schedule(
    house_id: String,
    user_id: String,
    events: Vec<ScheduledHouseEvent>,
    can_create: bool,
    admin: bool,
) -> Element {
    let api = use_context::<Signal<ApiClient>>();
    let mut items = use_signal(|| events);
    let mut status = use_signal(String::new);
    let mut saving = use_signal(|| false);
    let mut editing = use_signal(|| Option::<String>::None);
    let mut title = use_signal(String::new);
    let mut start = use_signal(String::new);
    let mut zone = use_signal(|| "UTC".to_string());
    let mut frequency = use_signal(|| "DAILY".to_string());
    let mut interval = use_signal(|| "1".to_string());
    let mut count = use_signal(|| "1".to_string());
    let mut until = use_signal(String::new);
    let mut exdates = use_signal(String::new);
    let save_house = house_id.clone();
    let save = move |_| {
        let request = match event_request(
            &title(),
            &start(),
            &zone(),
            &frequency(),
            &interval(),
            &count(),
            &until(),
            &exdates(),
        ) {
            Ok(v) => v,
            Err(e) => {
                status.set(e);
                return;
            }
        };
        saving.set(true);
        let api = api.read().cloned();
        let house_id = save_house.clone();
        let event_id = editing();
        spawn(async move {
            let result = if let Some(id) = event_id {
                api.update_scheduled_event(&house_id, &id, &request).await
            } else {
                api.create_scheduled_event(&house_id, &request).await
            };
            match result {
                Ok(event) => {
                    let mut values = items.read().clone();
                    if let Some(position) = values.iter().position(|value| value.id == event.id) {
                        values[position] = event;
                        status.set("Scheduled event updated.".into());
                    } else {
                        values.push(event);
                        status.set("Scheduled event created.".into());
                    }
                    items.set(values);
                    editing.set(None);
                }
                Err(e) => status.set(format!("Could not save event: {e}")),
            }
            saving.set(false);
        });
    };
    let cancel = move |_| {
        editing.set(None);
        status.set("Event editing cancelled.".into());
    };
    let delete_house = house_id.clone();
    let delete_event = move |deleted_id: String| {
        let api = api.read().cloned();
        let house_id = delete_house.clone();
        spawn(async move {
            match api.delete_scheduled_event(&house_id, &deleted_id).await {
                Ok(()) => {
                    let values = items
                        .read()
                        .iter()
                        .filter(|event| event.id != deleted_id)
                        .cloned()
                        .collect();
                    items.set(values);
                    status.set("Scheduled event deleted.".into());
                }
                Err(error) => status.set(format!("Could not delete event: {error}")),
            }
        });
    };
    let exceptions = |event: &ScheduledHouseEvent| event.exdates.join(", ");
    rsx! { div { class: "card", h3 { "Scheduled events" }
        for event in items.read().iter() { article { class: "card", h4 { "{event.title}" } p { "{event.dtstart_local} · {event.timezone} · {event.rrule}" } if !event.exdates.is_empty() { p { "Exceptions: {exceptions(event)}" } }
            if event.creator_id == user_id || admin {
                EventEditorButton { event: event.clone(), on_edit: move |event: ScheduledHouseEvent| {
                    title.set(event.title.clone()); start.set(event.dtstart_local.get(..16).unwrap_or(&event.dtstart_local).to_string()); zone.set(event.timezone.clone());
                    let rule = recurrence_fields(&event.rrule); frequency.set(rule.0); interval.set(rule.1); count.set(rule.2); until.set(rule.3); exdates.set(event.exdates.join(", ")); editing.set(Some(event.id)); status.set("Editing scheduled event.".into());
                } }
                EventDelete { event_id: event.id.clone(), on_delete: delete_event.clone() }
            }
        } }
        if can_create { h4 { if editing().is_some() { "Edit scheduled event" } else { "Add scheduled event" } }
            label { "Title", input { value: "{title}", oninput: move |e| title.set(e.value()) } } label { "Local start", input { r#type:"datetime-local", value:"{start}", oninput:move|e|start.set(e.value()) } } label { "IANA timezone", input { value:"{zone}", oninput:move|e|zone.set(e.value()) } } label { "Frequency", select { value:"{frequency}", oninput:move|e|frequency.set(e.value()), option { value:"DAILY", "Daily" } option { value:"WEEKLY", "Weekly" } option { value:"MONTHLY", "Monthly" } } } label { "Interval (1–366)", input { r#type:"number", value:"{interval}", oninput:move|e|interval.set(e.value()) } } label { "Count (1–366; leave blank to use until)", input { r#type:"number", value:"{count}", oninput:move|e|count.set(e.value()) } } label { "Until (optional, YYYYMMDDTHHMMSS)", input { value:"{until}", oninput:move|e|until.set(e.value()) } } label { "EXDATE local times (comma-separated)", input { value:"{exdates}", oninput:move|e|exdates.set(e.value()) } }
            button { disabled: saving(), onclick:save, if saving() { "Saving…" } else if editing().is_some() { "Save scheduled event" } else { "Create scheduled event" } }
            if editing().is_some() { button { disabled: saving(), onclick:cancel, "Cancel edit" } }
        } else { p { "Monitors can view scheduled events but cannot create, edit, or delete them." } }
        p { role:"status", "{status}" }
    } }
}
#[component]
fn EventEditorButton(
    event: ScheduledHouseEvent,
    on_edit: EventHandler<ScheduledHouseEvent>,
) -> Element {
    rsx! { button { onclick: move |_| on_edit.call(event.clone()), "Edit" } }
}
fn recurrence_fields(rule: &str) -> (String, String, String, String) {
    let field = |name| {
        rule.split(';')
            .find_map(|part| part.strip_prefix(name))
            .unwrap_or("")
            .to_string()
    };
    (
        field("FREQ="),
        {
            let value = field("INTERVAL=");
            if value.is_empty() {
                "1".into()
            } else {
                value
            }
        },
        field("COUNT="),
        field("UNTIL="),
    )
}

#[component]
fn EventDelete(event_id: String, on_delete: EventHandler<String>) -> Element {
    rsx! { button { onclick: move |_| on_delete.call(event_id.clone()), "Delete" } }
}
fn minutes_time(value: Option<u16>) -> String {
    value
        .map(|v| format!("{:02}:{:02}", v / 60, v % 60))
        .unwrap_or_default()
}
fn time_minutes(value: &str) -> Option<u16> {
    let (h, m) = value.split_once(':')?;
    let n = h
        .parse::<u16>()
        .ok()?
        .checked_mul(60)?
        .checked_add(m.parse().ok()?)?;
    (n < 1440).then_some(n)
}
#[allow(clippy::too_many_arguments)]
fn event_request(
    title: &str,
    start: &str,
    zone: &str,
    freq: &str,
    interval: &str,
    count: &str,
    until: &str,
    exdates: &str,
) -> Result<ScheduledEventRequest, String> {
    if title.trim().is_empty() || title.len() > 120 {
        return Err("Title must contain 1 to 120 characters.".into());
    }
    let i = interval
        .parse::<u16>()
        .map_err(|_| "Interval must be 1–366.")?;
    if !(1..=366).contains(&i) {
        return Err("Interval must be 1–366.".into());
    }
    let mut rule = format!("FREQ={freq};INTERVAL={i}");
    if !count.trim().is_empty() {
        let c = count.parse::<u16>().map_err(|_| "Count must be 1–366.")?;
        if !(1..=366).contains(&c) {
            return Err("Count must be 1–366.".into());
        }
        rule.push_str(&format!(";COUNT={c}"));
    } else if until.trim().is_empty() {
        return Err("Provide a count or an until date.".into());
    } else {
        rule.push_str(&format!(";UNTIL={}", until.trim()))
    }
    let dates = exdates
        .split(',')
        .map(str::trim)
        .filter(|s| !s.is_empty())
        .map(String::from)
        .collect::<Vec<_>>();
    if dates.len() > 100 {
        return Err("At most 100 exception dates are allowed.".into());
    }
    if start.len() != 16 || zone.trim().is_empty() {
        return Err("Local start and IANA timezone are required.".into());
    }
    Ok(ScheduledEventRequest {
        title: title.trim().into(),
        timezone: zone.trim().into(),
        dtstart_local: format!("{}:00", start),
        rrule: rule,
        exdates: dates,
        enabled: Some(true),
    })
}
fn browser_push_state() -> String {
    if cfg!(target_arch = "wasm32") {
        "Browser push is disabled.".into()
    } else {
        "Browser push is unsupported in the native desktop app.".into()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn notification_controls_stay_hidden_until_loading_succeeds() {
        assert!(!notification_controls_ready(
            &NotificationLoadState::Loading
        ));
        assert!(!notification_controls_ready(&NotificationLoadState::Error(
            "offline".into()
        )));
        assert!(notification_controls_ready(&NotificationLoadState::Ready));
    }

    #[test]
    fn service_worker_uses_generic_deduplicated_payloads() {
        let source = include_str!("../../public/roomies-sw.js");
        assert!(source.contains("body: 'You have a Roomies notification.'"));
        assert!(source.contains("tag: `roomies-${id}`"));
        assert!(source.contains("`/house/${encodeURIComponent(houseId)}`"));
        assert!(source.contains("const path = houseId ?"));
        assert!(!source.contains("/houses/"));
        assert!(!source.contains("endpoint"));
    }
    #[test]
    fn recurrence_fields_prefill_edit_form() {
        assert_eq!(
            recurrence_fields("FREQ=WEEKLY;INTERVAL=2;COUNT=3"),
            ("WEEKLY".into(), "2".into(), "3".into(), String::new())
        );
        assert_eq!(
            recurrence_fields("FREQ=MONTHLY;UNTIL=20251231T090000"),
            (
                "MONTHLY".into(),
                "1".into(),
                String::new(),
                "20251231T090000".into()
            )
        );
    }
    #[test]
    fn recurrence_request_enforces_client_bounds() {
        assert!(event_request(
            "Bins",
            "2025-01-02T09:00",
            "UTC",
            "DAILY",
            "367",
            "1",
            "",
            ""
        )
        .is_err());
        assert!(
            event_request("Bins", "2025-01-02T09:00", "UTC", "DAILY", "1", "", "", "").is_err()
        );
        assert!(event_request(
            "Bins",
            "2025-01-02T09:00",
            "UTC",
            "WEEKLY",
            "1",
            "2",
            "",
            "2025-01-09T09:00"
        )
        .is_ok());
    }
}
