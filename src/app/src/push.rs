#[cfg(target_arch = "wasm32")]
use crate::api::ApiClient;

#[cfg(target_arch = "wasm32")]
use wasm_bindgen::prelude::*;

#[cfg(target_arch = "wasm32")]
#[wasm_bindgen(inline_js = r#"
export async function subscribePush(vapid, label) {
  if (!('serviceWorker' in navigator) || !('PushManager' in window)) throw new Error('Browser push is unsupported.');
  const registration = await navigator.serviceWorker.register('/roomies-sw.js', {scope: '/'});
  const permission = await Notification.requestPermission();
  if (permission !== 'granted') throw new Error('Notification permission was denied.');
  const key = Uint8Array.from(atob(vapid.replace(/-/g,'+').replace(/_/g,'/')), c => c.charCodeAt(0));
  const subscription = await registration.pushManager.subscribe({userVisibleOnly:true, applicationServerKey:key});
  const json = subscription.toJSON();
  return { endpoint: json.endpoint, p256dh: json.keys.p256dh, auth: json.keys.auth, label: label };
}
export async function unsubscribePush() {
  if (!('serviceWorker' in navigator)) return;
  const registration = await navigator.serviceWorker.getRegistration('/');
  const subscription = registration && await registration.pushManager.getSubscription();
  if (subscription) await subscription.unsubscribe();
}
"#)]
extern "C" {
    #[wasm_bindgen(catch, js_name = subscribePush)]
    async fn subscribe_push(vapid: String, label: String) -> Result<JsValue, JsValue>;
    #[wasm_bindgen(catch, js_name = unsubscribePush)]
    async fn unsubscribe_push() -> Result<(), JsValue>;
}

#[cfg(target_arch = "wasm32")]
pub async fn enable(api: &ApiClient, house_id: &str) -> Result<(), String> {
    let key = api
        .get_vapid_public_key()
        .await
        .map_err(|e| e.to_string())?;
    if key.public_key.is_empty() {
        return Err("Browser push is not configured by this server.".into());
    }
    let value = subscribe_push(key.public_key, "Browser".into())
        .await
        .map_err(js_error)?;
    let request: crate::models::CreateNotificationSubscriptionRequest =
        serde_wasm_bindgen::from_value(value).map_err(|e| e.to_string())?;
    api.create_notification_subscription(house_id, &request)
        .await
        .map(|_| ())
        .map_err(|e| e.to_string())
}
#[cfg(target_arch = "wasm32")]
pub async fn disable(api: &ApiClient, house_id: &str, ids: &[String]) -> Result<(), String> {
    for id in ids {
        api.delete_notification_subscription(house_id, id)
            .await
            .map_err(|e| e.to_string())?;
    }
    unsubscribe_push().await.map_err(js_error)
}
#[cfg(target_arch = "wasm32")]
fn js_error(value: JsValue) -> String {
    value
        .as_string()
        .unwrap_or_else(|| "Browser push failed.".into())
}

#[cfg(not(target_arch = "wasm32"))]
pub async fn enable(_: &crate::api::ApiClient, _: &str) -> Result<(), String> {
    Err("Browser push is unsupported in the native desktop app.".into())
}
#[cfg(not(target_arch = "wasm32"))]
pub async fn disable(_: &crate::api::ApiClient, _: &str, _: &[String]) -> Result<(), String> {
    Err("Browser push is unsupported in the native desktop app.".into())
}
