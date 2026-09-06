//! Persistent session-token storage. Passwords are never stored.
//!
//! Web uses browser localStorage. Native targets use a per-user application-data
//! file with restrictive permissions; OS keychain/keystore integration is future
//! hardening, not a prerequisite for the MVP.

#[cfg(target_arch = "wasm32")]
const TOKEN_KEY: &str = "roomies.session.token";
#[cfg(target_arch = "wasm32")]
const PENDING_INVITATION_KEY: &str = "roomies.pending.invitation";

pub trait SessionStorage {
    fn load_token(&self) -> Option<String>;
    fn save_token(&mut self, token: &str);
    fn clear_token(&mut self);
}

#[derive(Default, Debug, Clone, PartialEq, Eq)]
pub struct MemorySessionStorage(Option<String>);
impl SessionStorage for MemorySessionStorage {
    fn load_token(&self) -> Option<String> {
        self.0.clone()
    }
    fn save_token(&mut self, token: &str) {
        self.0 = (!token.is_empty()).then(|| token.to_owned());
    }
    fn clear_token(&mut self) {
        self.0 = None;
    }
}

/// Loads the token for the current app user, if one was persisted.
pub fn load_token() -> Option<String> {
    #[cfg(target_arch = "wasm32")]
    {
        return web_sys::window()
            .and_then(|window| window.local_storage().ok().flatten())
            .and_then(|storage| storage.get_item(TOKEN_KEY).ok().flatten())
            .filter(|token| !token.is_empty());
    }

    #[cfg(not(target_arch = "wasm32"))]
    {
        std::fs::read_to_string(token_path()?)
            .ok()
            .map(|token| token.trim().to_owned())
            .filter(|token| !token.is_empty())
    }
}

/// Persists a bearer token. Errors are intentionally best-effort: the server
/// remains authoritative and the in-memory session still works for this run.
pub fn save_token(token: &str) {
    if token.is_empty() {
        return;
    }
    #[cfg(target_arch = "wasm32")]
    {
        if let Some(storage) =
            web_sys::window().and_then(|window| window.local_storage().ok().flatten())
        {
            let _ = storage.set_item(TOKEN_KEY, token);
        }
    }

    #[cfg(not(target_arch = "wasm32"))]
    {
        if let Some(path) = token_path() {
            if let Some(parent) = path.parent() {
                let _ = std::fs::create_dir_all(parent);
                set_private_permissions(parent, true);
            }
            if let Ok(mut file) = private_file(&path) {
                use std::io::Write;
                let _ = file.write_all(token.as_bytes());
                set_private_permissions(&path, false);
            }
        }
    }
}

/// Removes the persisted bearer token.
pub fn clear_token() {
    #[cfg(target_arch = "wasm32")]
    {
        if let Some(storage) =
            web_sys::window().and_then(|window| window.local_storage().ok().flatten())
        {
            let _ = storage.remove_item(TOKEN_KEY);
        }
    }

    #[cfg(not(target_arch = "wasm32"))]
    {
        if let Some(path) = token_path() {
            let _ = std::fs::remove_file(path);
        }
    }
}

/// Stores an invitation bearer token only for the current browser session while
/// the recipient signs in or registers. It is never rendered or persisted with
/// the normal session token.
pub fn save_pending_invitation(token: &str) {
    #[cfg(not(target_arch = "wasm32"))]
    let _ = token;
    #[cfg(target_arch = "wasm32")]
    if !token.is_empty() {
        if let Some(storage) =
            web_sys::window().and_then(|window| window.session_storage().ok().flatten())
        {
            let _ = storage.set_item(PENDING_INVITATION_KEY, token);
        }
    }
}

pub fn pending_invitation() -> Option<String> {
    #[cfg(target_arch = "wasm32")]
    {
        return web_sys::window()
            .and_then(|window| window.session_storage().ok().flatten())
            .and_then(|storage| storage.get_item(PENDING_INVITATION_KEY).ok().flatten())
            .filter(|token| !token.is_empty());
    }
    #[cfg(not(target_arch = "wasm32"))]
    {
        None
    }
}

pub fn clear_pending_invitation() {
    #[cfg(target_arch = "wasm32")]
    if let Some(storage) =
        web_sys::window().and_then(|window| window.session_storage().ok().flatten())
    {
        let _ = storage.remove_item(PENDING_INVITATION_KEY);
    }
}

#[cfg(all(not(target_arch = "wasm32"), target_os = "android"))]
fn token_path() -> Option<std::path::PathBuf> {
    // Dioxus.toml's app.roomies identifier maps to this private Android files directory.
    Some(std::path::PathBuf::from(
        "/data/user/0/app.roomies/files/roomies/session.token",
    ))
}

#[cfg(all(not(target_arch = "wasm32"), not(target_os = "android")))]
fn token_path() -> Option<std::path::PathBuf> {
    token_path_from_dirs(
        std::env::var_os("XDG_DATA_HOME").map(std::path::PathBuf::from),
        std::env::var_os("HOME").map(std::path::PathBuf::from),
    )
}

#[cfg(all(not(target_arch = "wasm32"), not(target_os = "android")))]
fn token_path_from_dirs(
    xdg_data_home: Option<std::path::PathBuf>,
    home: Option<std::path::PathBuf>,
) -> Option<std::path::PathBuf> {
    let base = xdg_data_home.or_else(|| home.map(|home| home.join(".local/share")))?;
    Some(base.join("roomies/session.token"))
}

#[cfg(all(not(target_arch = "wasm32"), unix))]
fn private_file(path: &std::path::Path) -> std::io::Result<std::fs::File> {
    use std::fs::OpenOptions;
    use std::os::unix::fs::OpenOptionsExt;
    OpenOptions::new()
        .create(true)
        .write(true)
        .truncate(true)
        .mode(0o600)
        .open(path)
}

#[cfg(all(not(target_arch = "wasm32"), not(unix)))]
fn private_file(path: &std::path::Path) -> std::io::Result<std::fs::File> {
    use std::fs::OpenOptions;
    OpenOptions::new()
        .create(true)
        .write(true)
        .truncate(true)
        .open(path)
}

#[cfg(all(not(target_arch = "wasm32"), unix))]
fn set_private_permissions(path: &std::path::Path, directory: bool) {
    use std::os::unix::fs::PermissionsExt;
    let mode = if directory { 0o700 } else { 0o600 };
    if let Ok(metadata) = std::fs::metadata(path) {
        let mut permissions = metadata.permissions();
        permissions.set_mode(mode);
        let _ = std::fs::set_permissions(path, permissions);
    }
}

#[cfg(all(not(target_arch = "wasm32"), not(unix)))]
fn set_private_permissions(_path: &std::path::Path, _directory: bool) {}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn credentials_can_be_replaced_and_cleared() {
        let mut store = MemorySessionStorage::default();
        assert_eq!(store.load_token(), None);
        store.save_token("token");
        assert_eq!(store.load_token().as_deref(), Some("token"));
        store.clear_token();
        assert_eq!(store.load_token(), None);
    }

    #[test]
    fn empty_tokens_are_not_persisted_in_memory() {
        let mut store = MemorySessionStorage::default();
        store.save_token("");
        assert_eq!(store.load_token(), None);
    }

    #[cfg(all(not(target_arch = "wasm32"), not(target_os = "android")))]
    #[test]
    fn native_token_path_uses_private_app_data_directory() {
        let from_xdg = token_path_from_dirs(
            Some(std::path::PathBuf::from("/tmp/data")),
            Some(std::path::PathBuf::from("/tmp/home")),
        )
        .unwrap();
        assert_eq!(
            from_xdg,
            std::path::PathBuf::from("/tmp/data/roomies/session.token")
        );

        let from_home =
            token_path_from_dirs(None, Some(std::path::PathBuf::from("/tmp/home"))).unwrap();
        assert_eq!(
            from_home,
            std::path::PathBuf::from("/tmp/home/.local/share/roomies/session.token")
        );
    }

    #[cfg(unix)]
    #[test]
    fn private_files_use_restrictive_permissions() {
        use std::os::unix::fs::PermissionsExt;

        let mut path = std::env::temp_dir();
        let unique = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap()
            .as_nanos();
        path.push(format!("roomies-session-{}-{}", std::process::id(), unique));
        std::fs::create_dir_all(&path).unwrap();
        let token_path = path.join("token");
        let file = private_file(&token_path).unwrap();
        drop(file);
        let mode = std::fs::metadata(&token_path).unwrap().permissions().mode() & 0o777;
        assert_eq!(mode, 0o600);
        let _ = std::fs::remove_file(&token_path);
        let _ = std::fs::remove_dir_all(&path);
    }
}
