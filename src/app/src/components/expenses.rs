use crate::api::{ApiClient, ExpenseDetail};
use crate::core::{
    can_change_expense_visibility, can_create, can_mutate_owned, cents_to_api_amount,
    parse_money_cents, Role,
};
use crate::models::{
    CreateExpenseRequest, Expense, HouseMember, SplitEntry, UpdateExpenseRequest, User,
};
use dioxus::prelude::*;

fn split_entries(value: &str, total: i64) -> Result<Vec<SplitEntry>, String> {
    let mut entries = Vec::new();
    let mut sum = 0_i64;
    for item in value.split(',').filter(|item| !item.trim().is_empty()) {
        let (user_id, amount) = item
            .trim()
            .split_once(':')
            .ok_or("Custom splits use user-id:amount, separated by commas")?;
        let cents = parse_money_cents(amount).map_err(str::to_owned)?;
        if user_id.trim().is_empty() {
            return Err("A split user id is required".into());
        }
        sum += cents;
        entries.push(SplitEntry {
            user_id: user_id.trim().into(),
            amount: cents_to_api_amount(cents),
        });
    }
    if entries.is_empty() || sum != total {
        return Err("Custom split amounts must exactly equal the expense amount".into());
    }
    Ok(entries)
}

#[component]
pub fn ExpensesSection(house_id: String) -> Element {
    let api = use_context::<Signal<ApiClient>>();
    let user = use_context::<Signal<Option<User>>>();
    let mut expenses = use_signal(Vec::<Expense>::new);
    let mut members = use_signal(Vec::<HouseMember>::new);
    let mut selected = use_signal(|| Option::<ExpenseDetail>::None);
    let mut show_form = use_signal(|| false);
    let mut amount = use_signal(String::new);
    let mut description = use_signal(String::new);
    let mut category = use_signal(String::new);
    let mut date = use_signal(|| chrono::Local::now().format("%Y-%m-%d").to_string());
    let mut visibility = use_signal(|| "shared".to_string());
    let mut recipients = use_signal(String::new);
    let mut custom_splits = use_signal(String::new);
    let mut error = use_signal(String::new);
    let mut loading = use_signal(|| true);
    let mut confirm_delete = use_signal(|| Option::<String>::None);
    let id = house_id.clone();
    use_effect(move || {
        let api = api.read().cloned();
        let id = id.clone();
        spawn(async move {
            loading.set(true);
            error.set(String::new());
            match (api.get_expenses(&id).await, api.get_members(&id).await) {
                (Ok(list), Ok(member_list)) => {
                    expenses.set(list);
                    members.set(member_list);
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
    let can_create_expense = role.map(|r| can_create(r, "expense")).unwrap_or(false);
    let create_id = house_id.clone();
    let create = move |_| {
        let cents = match parse_money_cents(&amount.read()) {
            Ok(v) if v > 0 => v,
            _ => {
                error.set("Enter a positive amount with at most two decimals".into());
                return;
            }
        };
        if description.read().trim().is_empty() {
            error.set("Description is required".into());
            return;
        }
        let split = match split_entries(&custom_splits.read(), cents) {
            Ok(v) => v,
            Err(_) if custom_splits.read().trim().is_empty() => vec![],
            Err(e) => {
                error.set(e);
                return;
            }
        };
        let visible_to = recipients
            .read()
            .split(',')
            .filter_map(|id| {
                let id = id.trim();
                (!id.is_empty()).then(|| id.to_string())
            })
            .collect::<Vec<_>>();
        if visibility.read().as_str() == "shared" && !visible_to.is_empty() {
            error.set("Recipients are only valid for private expenses".into());
            return;
        }
        let req = CreateExpenseRequest {
            amount: cents_to_api_amount(cents),
            description: description.read().trim().into(),
            category: category.read().trim().into(),
            date: date.read().clone(),
            visibility: visibility.read().clone(),
            visible_to,
            split,
        };
        let api = api.read().cloned();
        let id = create_id.clone();
        spawn(async move {
            loading.set(true);
            error.set(String::new());
            match api.create_expense(&id, &req).await {
                Ok(_) => {
                    show_form.set(false);
                    amount.set(String::new());
                    description.set(String::new());
                    custom_splits.set(String::new());
                    match api.get_expenses(&id).await {
                        Ok(list) => expenses.set(list),
                        Err(e) => error.set(e.to_string()),
                    }
                }
                Err(e) => error.set(e.to_string()),
            };
            loading.set(false);
        });
    };
    let saved_house_id = house_id.clone();
    rsx! { section { class: "section", "aria-live": "polite",
        h2 { "Expenses" }
        if !error.read().is_empty() { p { class: "error", role: "alert", "{error}" } }
        if can_create_expense { button { onclick: move |_| { let next = !*show_form.read(); show_form.set(next); }, if *show_form.read() { "Cancel" } else { "Add expense" } } } else { p { "Your monitor role is view-only." } }
        if *show_form.read() { div { class: "card", h3 { "New expense" }
            label { "Amount", input { r#type: "text", inputmode: "decimal", value: "{amount}", oninput: move |e| amount.set(e.value()) } }
            label { "Description", input { value: "{description}", oninput: move |e| description.set(e.value()) } }
            label { "Category (optional)", input { value: "{category}", oninput: move |e| category.set(e.value()) } }
            label { "Date", input { r#type: "date", value: "{date}", oninput: move |e| date.set(e.value()) } }
            label { "Visibility", select { value: "{visibility}", oninput: move |e| visibility.set(e.value()), option { value: "shared", "Shared" } option { value: "private", "Private" } } }
            if visibility.read().as_str() == "private" { label { "Recipient user IDs (comma separated; payer is always included)", input { value: "{recipients}", oninput: move |e| recipients.set(e.value()) } } }
            label { "Custom splits (optional: user-id:12.34, user-id:5.00). Leave empty to split equally among active members.", input { value: "{custom_splits}", oninput: move |e| custom_splits.set(e.value()) } }
            button { disabled: *loading.read(), onclick: create, "Save expense" }
        } }
        if *loading.read() { p { "Loading expenses…" } } else if expenses.read().is_empty() { p { "No expenses yet." } } else { div { class: "list",
            for exp in expenses.read().iter() { { let exp = exp.clone(); let detail_id = exp.id.clone(); let delete_id = exp.id.clone(); let detail_house = house_id.clone(); let owner = user.read().as_ref().map(|u| u.id == exp.payer_id).unwrap_or(false); let editable = role.map(|r| can_mutate_owned(r, owner) || matches!(r, Role::Admin)).unwrap_or(false); rsx! { article { class: "card", h3 { "{exp.description}" } p { "{exp.date} · ${exp.amount:.2} · {exp.payer_name} · {exp.visibility}" } if editable { button { onclick: move |_| { let api=api.read().cloned(); let h=detail_house.clone(); let eid=detail_id.clone(); spawn(async move { match api.get_expense(&h,&eid).await { Ok(detail) => selected.set(Some(detail)), Err(e) => error.set(e.to_string()) } }); }, "Details / edit" } button { onclick: move |_| confirm_delete.set(Some(delete_id.clone())), "Delete" } } } } } }
        } }
        if let Some(detail) = selected.read().clone() {
            ExpenseDialog {
                house_id: house_id.clone(),
                detail,
                members: members.read().clone(),
                on_close: move |_| selected.set(None),
                on_saved: move |_| {
                    selected.set(None);
                    let api = api.read().cloned();
                    let house_id = saved_house_id.clone();
                    spawn(async move {
                        match api.get_expenses(&house_id).await {
                            Ok(list) => expenses.set(list),
                            Err(e) => error.set(e.to_string()),
                        }
                    });
                }
            }
        }
        if let Some(expense_id) = confirm_delete.read().clone() { div { class: "card", role: "alertdialog", h3 { "Delete expense?" } p { "This cannot be undone." } button { onclick: move |_| { let api=api.read().cloned(); let h=house_id.clone(); let eid=expense_id.clone(); spawn(async move { match api.delete_expense(&h,&eid).await { Ok(()) => { confirm_delete.set(None); match api.get_expenses(&h).await { Ok(list)=>expenses.set(list), Err(e)=>error.set(e.to_string()) } }, Err(e)=>error.set(e.to_string()) } }); }, "Confirm delete" } button { onclick: move |_| confirm_delete.set(None), "Cancel" } } }
    } }
}

#[component]
fn ExpenseDialog(
    house_id: String,
    detail: ExpenseDetail,
    members: Vec<HouseMember>,
    on_close: EventHandler<()>,
    on_saved: EventHandler<()>,
) -> Element {
    let api = use_context::<Signal<ApiClient>>();
    let user = use_context::<Signal<Option<User>>>();
    let mut error = use_signal(String::new);
    let mut amount = use_signal(|| format!("{:.2}", detail.expense.amount));
    let mut description = use_signal(|| detail.expense.description.clone());
    let mut category = use_signal(|| detail.expense.category.clone());
    let mut date = use_signal(|| detail.expense.date.clone());
    let mut visibility = use_signal(|| detail.expense.visibility.clone());
    let mut recipients = use_signal(String::new);
    let detail_payer_id = detail.expense.payer_id.clone();
    let mut split = use_signal(|| {
        detail
            .splits
            .iter()
            .map(|s| format!("{}:{:.2}", s.user_id, s.share_amount))
            .collect::<Vec<_>>()
            .join(", ")
    });
    let is_payer = user
        .read()
        .as_ref()
        .map(|u| u.id == detail_payer_id)
        .unwrap_or(false);
    let can_change_visibility = user
        .read()
        .as_ref()
        .and_then(|u| {
            members
                .iter()
                .find(|m| m.user_id == u.id)
                .and_then(|m| Role::parse(&m.role))
        })
        .map(|role| can_change_expense_visibility(role, is_payer))
        .unwrap_or(false);
    let save = move |_| {
        let cents = match parse_money_cents(&amount.read()) {
            Ok(v) if v > 0 => v,
            _ => {
                error.set("Invalid amount".into());
                return;
            }
        };
        let old = (detail.expense.amount * 100.0).round() as i64;
        let replacement = if cents != old {
            match split_entries(&split.read(), cents) {
                Ok(v) => Some(v),
                Err(e) => {
                    error.set(format!("Changing amount requires a replacement split: {e}"));
                    return;
                }
            }
        } else {
            None
        };
        let req = UpdateExpenseRequest {
            amount: Some(cents_to_api_amount(cents)),
            description: Some(description.read().trim().into()),
            category: Some(category.read().trim().into()),
            date: Some(date.read().clone()),
            split: replacement,
        };
        let api = api.read().cloned();
        let h = house_id.clone();
        let eid = detail.expense.id.clone();
        let visibility = visibility.read().clone();
        let visible_to = recipients
            .read()
            .split(',')
            .filter_map(|id| {
                let id = id.trim();
                (!id.is_empty()).then(|| id.to_string())
            })
            .collect::<Vec<_>>();
        spawn(async move {
            match api.update_expense(&h, &eid, &req).await {
                Ok(_) if can_change_visibility => {
                    match api.set_visibility(&h, &eid, &visibility, &visible_to).await {
                        Ok(()) => on_saved.call(()),
                        Err(e) => error.set(e.to_string()),
                    }
                }
                Ok(_) => on_saved.call(()),
                Err(e) => error.set(e.to_string()),
            }
        });
    };
    let active_ids = members
        .iter()
        .filter(|m| m.role != "monitor")
        .map(|m| m.user_id.as_str())
        .collect::<Vec<_>>()
        .join(", ");
    rsx! { div { class: "card", role: "dialog", h3 { "Expense details" } if !error.read().is_empty() { p { class: "error", role: "alert", "{error}" } } label { "Amount", input { value: "{amount}", oninput: move |e| amount.set(e.value()) } } label { "Description", input { value: "{description}", oninput: move |e| description.set(e.value()) } } label { "Category", input { value: "{category}", oninput: move |e| category.set(e.value()) } } label { "Date", input { r#type:"date", value:"{date}", oninput:move |e|date.set(e.value()) } } label { "Replacement/custom split (required if amount changes)", input { value:"{split}", oninput:move |e|split.set(e.value()) } } p { "Active members: {active_ids}" } if can_change_visibility { label { "Visibility", select { value: "{visibility}", oninput: move |e| visibility.set(e.value()), option { value: "shared", "Shared" }, option { value: "private", "Private" } } } if visibility.read().as_str() == "private" { label { "Recipient user IDs (comma separated; payer is always included)", input { value: "{recipients}", oninput: move |e| recipients.set(e.value()) } } } } button { onclick:save, "Save changes" } button { onclick:move |_|on_close.call(()), "Close" } } }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn split_serialization_is_cent_exact() {
        let split = split_entries("a:1.01,b:0.99", 200).unwrap();
        assert_eq!(
            serde_json::to_string(&split).unwrap(),
            r#"[{"user_id":"a","amount":1.01},{"user_id":"b","amount":0.99}]"#
        );
    }
}
