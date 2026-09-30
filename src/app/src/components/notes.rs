use crate::api::ApiClient;
use crate::core::{can_create, can_mutate_note, Role};
use crate::models::{Note, UpdateNoteRequest, User};
use dioxus::prelude::*;

#[component]
pub fn NotesSection(house_id: String, role: String) -> Element {
    let api = use_context::<Signal<ApiClient>>();
    let user = use_context::<Signal<Option<User>>>();
    let mut notes = use_signal(Vec::<Note>::new);
    let mut title = use_signal(String::new);
    let mut content = use_signal(String::new);
    let mut selected = use_signal(|| Option::<Note>::None);
    let mut confirm_delete = use_signal(|| Option::<String>::None);
    let mut error = use_signal(String::new);
    let mut loading = use_signal(|| true);
    let parsed_role = Role::parse(&role).unwrap_or(Role::Monitor);
    let can_create_note = can_create(parsed_role, "note");
    let h = house_id.clone();

    use_effect(move || {
        let api = api.read().cloned();
        let house_id = h.clone();
        spawn(async move {
            loading.set(true);
            match api.get_notes(&house_id).await {
                Ok(value) => notes.set(value),
                Err(e) => error.set(e.to_string()),
            }
            loading.set(false);
        });
    });

    let create_house_id = house_id.clone();
    let create = move |_| {
        if title.read().trim().is_empty() || content.read().trim().is_empty() {
            error.set("Title and content are required".into());
            return;
        }
        let api = api.read().cloned();
        let house_id = create_house_id.clone();
        let title_value = title.read().clone();
        let content_value = content.read().clone();
        spawn(async move {
            loading.set(true);
            match api
                .create_note(&house_id, &title_value, &content_value)
                .await
            {
                Ok(_) => {
                    title.set(String::new());
                    content.set(String::new());
                    match api.get_notes(&house_id).await {
                        Ok(value) => notes.set(value),
                        Err(e) => error.set(e.to_string()),
                    }
                }
                Err(e) => error.set(e.to_string()),
            }
            loading.set(false);
        });
    };

    let saved_house_id = house_id.clone();
    let delete_house_id = house_id.clone();
    rsx! {
        section { class: "section",
            h2 { "Notes" }
            if !error.read().is_empty() { p { class: "error", role: "alert", "{error}" } }
            if can_create_note {
                div { class: "card", h3 { "New note" },
                    label { "Title", input { value: "{title}", oninput: move |e| title.set(e.value()) } }
                    label { "Content", textarea { value: "{content}", oninput: move |e| content.set(e.value()), rows: "5" } }
                    button { disabled: *loading.read(), onclick: create, "Save note" }
                }
            } else {
                p { "Your monitor role is view-only." }
            }
            if *loading.read() {
                p { "Loading notes…" }
            } else if notes.read().is_empty() {
                p { "No notes yet." }
            } else {
                for note in notes.read().iter() {
                    {
                        let note = note.clone();
                        let edit_note = note.clone();
                        let delete_id = note.id.clone();
                        let author = user.read().as_ref().map(|u| u.id == note.author_id).unwrap_or(false);
                        let can_edit = can_mutate_note(parsed_role, author);
                        rsx! {
                            article { class: "card note",
                                h3 { "{note.title}" }
                                p { "{note.content}" }
                                small { "By {note.author_name} · updated {note.updated_at}" }
                                if can_edit {
                                    button { onclick: move |_| selected.set(Some(edit_note.clone())), "Edit" }
                                    button { onclick: move |_| confirm_delete.set(Some(delete_id.clone())), "Delete" }
                                }
                            }
                        }
                    }
                }
            }
            if let Some(note) = selected.read().clone() {
                NoteEditor {
                    house_id: house_id.clone(),
                    note,
                    on_close: move |_| selected.set(None),
                    on_saved: move |_| {
                        selected.set(None);
                        let api = api.read().cloned();
                        let house_id = saved_house_id.clone();
                        spawn(async move {
                            match api.get_notes(&house_id).await {
                                Ok(value) => notes.set(value),
                                Err(e) => error.set(e.to_string()),
                            }
                        });
                    }
                }
            }
            if let Some(note_id) = confirm_delete.read().clone() {
                div { class: "card", role: "alertdialog",
                    h3 { "Delete note?" }
                    p { "This cannot be undone." }
                    button {
                        onclick: move |_| {
                            let api = api.read().cloned();
                            let house_id = delete_house_id.clone();
                            let note_id = note_id.clone();
                            spawn(async move {
                                match api.delete_note(&house_id, &note_id).await {
                                    Ok(()) => {
                                        confirm_delete.set(None);
                                        match api.get_notes(&house_id).await {
                                            Ok(value) => notes.set(value),
                                            Err(e) => error.set(e.to_string()),
                                        }
                                    }
                                    Err(e) => error.set(e.to_string()),
                                }
                            });
                        },
                        "Confirm delete"
                    }
                    button { onclick: move |_| confirm_delete.set(None), "Cancel" }
                }
            }
        }
    }
}

#[component]
fn NoteEditor(
    house_id: String,
    note: Note,
    on_close: EventHandler<()>,
    on_saved: EventHandler<()>,
) -> Element {
    let api = use_context::<Signal<ApiClient>>();
    let mut title = use_signal(|| note.title.clone());
    let mut content = use_signal(|| note.content.clone());
    let mut error = use_signal(String::new);
    let save = move |_| {
        if title.read().trim().is_empty() || content.read().trim().is_empty() {
            error.set("Title and content are required".into());
            return;
        }
        let api = api.read().cloned();
        let house_id = house_id.clone();
        let note_id = note.id.clone();
        let request = UpdateNoteRequest {
            title: title.read().trim().into(),
            content: content.read().trim().into(),
        };
        spawn(async move {
            match api.update_note(&house_id, &note_id, &request).await {
                Ok(_) => on_saved.call(()),
                Err(e) => error.set(e.to_string()),
            }
        });
    };
    rsx! {
        div { class: "card", role: "dialog",
            h3 { "Edit note" }
            if !error.read().is_empty() { p { class: "error", role: "alert", "{error}" } }
            label { "Title", input { value: "{title}", oninput: move |e| title.set(e.value()) } }
            label { "Content", textarea { value: "{content}", oninput: move |e| content.set(e.value()) } }
            button { onclick: save, "Save changes" }
            button { onclick: move |_| on_close.call(()), "Close" }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn update_note_serializes_required_fields() {
        let request = UpdateNoteRequest {
            title: "A".into(),
            content: "B".into(),
        };
        assert_eq!(
            serde_json::to_string(&request).unwrap(),
            r#"{"title":"A","content":"B"}"#
        );
    }
    #[test]
    fn monitor_cannot_create_notes() {
        assert!(!can_create(Role::Monitor, "note"));
        assert!(can_create(Role::Admin, "note"));
    }
}
