//! Persistent session-token storage. Passwords are never stored.
//!
//! Web keeps the existing browser localStorage behavior. Linux desktop and
//! Android use their platform credential stores; the old token file is read
//! only for a one-time, secure-first migration.

#[cfg(target_arch = "wasm32")]
const TOKEN_KEY: &str = "roomies.session.token";
#[cfg(target_arch = "wasm32")]
const PENDING_INVITATION_KEY: &str = "roomies.pending.invitation";
#[cfg(not(target_arch = "wasm32"))]
const SECRET_SERVICE: &str = "app.roomies";
#[cfg(not(target_arch = "wasm32"))]
const SECRET_ACCOUNT: &str = "session";
#[cfg(not(target_arch = "wasm32"))]
const MAX_TOKEN_BYTES: usize = 8 * 1024;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum StorageError {
    Unavailable,
    Corrupt,
    WriteFailed,
    DeleteFailed,
}

impl std::fmt::Display for StorageError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.write_str(match self {
            Self::Unavailable => "Secure session storage is unavailable.",
            Self::Corrupt => "The saved session could not be read safely.",
            Self::WriteFailed => "The session could not be saved securely.",
            Self::DeleteFailed => "The saved session could not be fully removed.",
        })
    }
}

impl std::error::Error for StorageError {}

pub(crate) trait SessionTokenStore: Send + Sync {
    fn load(&self) -> Result<Option<String>, StorageError>;
    fn save(&self, token: &str) -> Result<(), StorageError>;
    fn clear(&self) -> Result<(), StorageError>;
}

pub(crate) fn production_store() -> std::sync::Arc<dyn SessionTokenStore> {
    #[cfg(target_arch = "wasm32")]
    return std::sync::Arc::new(WebSessionTokenStore);

    #[cfg(not(target_arch = "wasm32"))]
    return std::sync::Arc::new(NativeSessionTokenStore::production());
}

#[cfg(target_arch = "wasm32")]
struct WebSessionTokenStore;

#[cfg(target_arch = "wasm32")]
impl SessionTokenStore for WebSessionTokenStore {
    fn load(&self) -> Result<Option<String>, StorageError> {
        Ok(web_sys::window()
            .and_then(|window| window.local_storage().ok().flatten())
            .and_then(|storage| storage.get_item(TOKEN_KEY).ok().flatten())
            .filter(|token| !token.is_empty()))
    }

    fn save(&self, token: &str) -> Result<(), StorageError> {
        if !token.is_empty() {
            if let Some(storage) =
                web_sys::window().and_then(|window| window.local_storage().ok().flatten())
            {
                let _ = storage.set_item(TOKEN_KEY, token);
            }
        }
        Ok(())
    }

    fn clear(&self) -> Result<(), StorageError> {
        if let Some(storage) =
            web_sys::window().and_then(|window| window.local_storage().ok().flatten())
        {
            let _ = storage.remove_item(TOKEN_KEY);
        }
        Ok(())
    }
}

#[cfg(not(target_arch = "wasm32"))]
trait SessionSecretStore: Send + Sync {
    fn get(&self) -> Result<Option<String>, StorageError>;
    fn set(&self, token: &str) -> Result<(), StorageError>;
    fn delete(&self) -> Result<(), StorageError>;
}

#[cfg(not(target_arch = "wasm32"))]
trait LegacyTokenFile: Send + Sync {
    fn read(&self) -> Result<Option<Vec<u8>>, StorageError>;
    fn delete(&self) -> Result<(), StorageError>;
}

#[cfg(not(target_arch = "wasm32"))]
struct NativeSessionTokenStore {
    secret: std::sync::Arc<dyn SessionSecretStore>,
    legacy: std::sync::Arc<dyn LegacyTokenFile>,
}

#[cfg(not(target_arch = "wasm32"))]
impl NativeSessionTokenStore {
    fn production() -> Self {
        Self {
            secret: std::sync::Arc::new(PlatformSecretStore::new()),
            legacy: std::sync::Arc::new(FilesystemLegacyTokenFile {
                path: legacy_token_path(),
            }),
        }
    }

    #[cfg(test)]
    fn with_backends(
        secret: std::sync::Arc<dyn SessionSecretStore>,
        legacy: std::sync::Arc<dyn LegacyTokenFile>,
    ) -> Self {
        Self { secret, legacy }
    }
}

#[cfg(not(target_arch = "wasm32"))]
impl SessionTokenStore for NativeSessionTokenStore {
    fn load(&self) -> Result<Option<String>, StorageError> {
        if let Some(token) = self.secret.get()? {
            let token = validate_token(token.as_bytes())?;
            // A prior migration may have stored the secret immediately before a
            // crash or failed unlink. Never read that plaintext again.
            self.legacy.delete()?;
            return Ok(Some(token));
        }

        let Some(bytes) = self.legacy.read()? else {
            return Ok(None);
        };
        let token = validate_token(&bytes)?;
        self.secret.set(&token)?;
        match self.secret.get()? {
            Some(saved) if saved == token => {}
            Some(_) => return Err(StorageError::Corrupt),
            None => return Err(StorageError::WriteFailed),
        }
        self.legacy.delete()?;
        Ok(Some(token))
    }

    fn save(&self, token: &str) -> Result<(), StorageError> {
        let token = validate_token(token.as_bytes()).map_err(|_| StorageError::WriteFailed)?;
        self.secret.set(&token)
    }

    fn clear(&self) -> Result<(), StorageError> {
        let secret_result = self.secret.delete();
        let legacy_result = self.legacy.delete();
        secret_result.and(legacy_result)
    }
}

#[cfg(not(target_arch = "wasm32"))]
fn validate_token(bytes: &[u8]) -> Result<String, StorageError> {
    if bytes.is_empty()
        || bytes.len() > MAX_TOKEN_BYTES
        || !bytes.iter().all(|byte| byte.is_ascii_graphic())
    {
        return Err(StorageError::Corrupt);
    }
    String::from_utf8(bytes.to_vec()).map_err(|_| StorageError::Corrupt)
}

#[cfg(not(target_arch = "wasm32"))]
struct PlatformSecretStore {
    entry: Result<keyring_core::Entry, StorageError>,
}

#[cfg(not(target_arch = "wasm32"))]
impl PlatformSecretStore {
    fn new() -> Self {
        use keyring_core::api::CredentialStoreApi;

        #[cfg(target_os = "linux")]
        let store =
            zbus_secret_service_keyring_store::Store::new().map_err(|_| StorageError::Unavailable);

        #[cfg(target_os = "android")]
        let store = {
            let configuration = std::collections::HashMap::from([("name", SECRET_SERVICE)]);
            android_native_keyring_store::Store::new_with_configuration(&configuration)
                .map_err(|_| StorageError::Unavailable)
        };

        #[cfg(not(any(target_os = "linux", target_os = "android")))]
        let store: Result<std::sync::Arc<dyn keyring_core::CredentialStore>, StorageError> =
            Err(StorageError::Unavailable);

        let entry = store.and_then(|store| {
            store
                .build(SECRET_SERVICE, SECRET_ACCOUNT, None)
                .map_err(|_| StorageError::Unavailable)
        });
        Self { entry }
    }

    fn entry(&self) -> Result<&keyring_core::Entry, StorageError> {
        self.entry.as_ref().map_err(|error| *error)
    }
}

#[cfg(not(target_arch = "wasm32"))]
impl SessionSecretStore for PlatformSecretStore {
    fn get(&self) -> Result<Option<String>, StorageError> {
        match self.entry()?.get_password() {
            Ok(token) => Ok(Some(token)),
            Err(keyring_core::Error::NoEntry) => Ok(None),
            Err(
                keyring_core::Error::BadEncoding(_)
                | keyring_core::Error::BadDataFormat(_, _)
                | keyring_core::Error::BadStoreFormat(_),
            ) => Err(StorageError::Corrupt),
            Err(_) => Err(StorageError::Unavailable),
        }
    }

    fn set(&self, token: &str) -> Result<(), StorageError> {
        self.entry()?
            .set_password(token)
            .map_err(|_| StorageError::WriteFailed)
    }

    fn delete(&self) -> Result<(), StorageError> {
        match self.entry()?.delete_credential() {
            Ok(()) | Err(keyring_core::Error::NoEntry) => Ok(()),
            Err(_) => Err(StorageError::DeleteFailed),
        }
    }
}

#[cfg(not(target_arch = "wasm32"))]
struct FilesystemLegacyTokenFile {
    path: Option<std::path::PathBuf>,
}

#[cfg(not(target_arch = "wasm32"))]
impl LegacyTokenFile for FilesystemLegacyTokenFile {
    fn read(&self) -> Result<Option<Vec<u8>>, StorageError> {
        use std::io::Read;

        let Some(path) = &self.path else {
            return Ok(None);
        };
        let file = match std::fs::File::open(path) {
            Ok(file) => file,
            Err(error) if error.kind() == std::io::ErrorKind::NotFound => return Ok(None),
            Err(_) => return Err(StorageError::Unavailable),
        };
        let mut bytes = Vec::new();
        file.take((MAX_TOKEN_BYTES + 1) as u64)
            .read_to_end(&mut bytes)
            .map_err(|_| StorageError::Unavailable)?;
        if bytes.len() > MAX_TOKEN_BYTES {
            return Err(StorageError::Corrupt);
        }
        Ok(Some(bytes))
    }

    fn delete(&self) -> Result<(), StorageError> {
        let Some(path) = &self.path else {
            return Ok(());
        };
        match std::fs::remove_file(path) {
            Ok(()) => Ok(()),
            Err(error) if error.kind() == std::io::ErrorKind::NotFound => Ok(()),
            Err(_) => Err(StorageError::DeleteFailed),
        }
    }
}

#[cfg(all(not(target_arch = "wasm32"), target_os = "android"))]
fn legacy_token_path() -> Option<std::path::PathBuf> {
    Some(std::path::PathBuf::from(
        "/data/user/0/app.roomies/files/roomies/session.token",
    ))
}

#[cfg(all(not(target_arch = "wasm32"), not(target_os = "android")))]
fn legacy_token_path() -> Option<std::path::PathBuf> {
    legacy_token_path_from_dirs(
        std::env::var_os("XDG_DATA_HOME").map(std::path::PathBuf::from),
        std::env::var_os("HOME").map(std::path::PathBuf::from),
    )
}

#[cfg(all(not(target_arch = "wasm32"), not(target_os = "android")))]
fn legacy_token_path_from_dirs(
    xdg_data_home: Option<std::path::PathBuf>,
    home: Option<std::path::PathBuf>,
) -> Option<std::path::PathBuf> {
    let base = xdg_data_home.or_else(|| home.map(|home| home.join(".local/share")))?;
    Some(base.join("roomies/session.token"))
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

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::{Arc, Mutex};

    #[derive(Default)]
    struct FakeSecret {
        value: Mutex<Option<String>>,
        get_error: Mutex<Option<StorageError>>,
        set_error: Mutex<Option<StorageError>>,
        delete_error: Mutex<Option<StorageError>>,
        sets: Mutex<usize>,
        deletes: Mutex<usize>,
    }

    impl SessionSecretStore for FakeSecret {
        fn get(&self) -> Result<Option<String>, StorageError> {
            if let Some(error) = *self.get_error.lock().unwrap() {
                return Err(error);
            }
            Ok(self.value.lock().unwrap().clone())
        }
        fn set(&self, token: &str) -> Result<(), StorageError> {
            *self.sets.lock().unwrap() += 1;
            if let Some(error) = *self.set_error.lock().unwrap() {
                return Err(error);
            }
            *self.value.lock().unwrap() = Some(token.to_owned());
            Ok(())
        }
        fn delete(&self) -> Result<(), StorageError> {
            *self.deletes.lock().unwrap() += 1;
            if let Some(error) = *self.delete_error.lock().unwrap() {
                return Err(error);
            }
            *self.value.lock().unwrap() = None;
            Ok(())
        }
    }

    #[derive(Default)]
    struct FakeLegacy {
        value: Mutex<Option<Vec<u8>>>,
        reads: Mutex<usize>,
        deletes: Mutex<usize>,
    }

    impl LegacyTokenFile for FakeLegacy {
        fn read(&self) -> Result<Option<Vec<u8>>, StorageError> {
            *self.reads.lock().unwrap() += 1;
            Ok(self.value.lock().unwrap().clone())
        }
        fn delete(&self) -> Result<(), StorageError> {
            *self.deletes.lock().unwrap() += 1;
            *self.value.lock().unwrap() = None;
            Ok(())
        }
    }

    fn store(secret: Arc<FakeSecret>, legacy: Arc<FakeLegacy>) -> NativeSessionTokenStore {
        NativeSessionTokenStore::with_backends(secret, legacy)
    }

    #[test]
    fn secure_get_set_and_delete_use_the_secret_store() {
        let secret = Arc::new(FakeSecret::default());
        let legacy = Arc::new(FakeLegacy::default());
        let store = store(secret.clone(), legacy);
        store.save("token").unwrap();
        assert_eq!(store.load().unwrap().as_deref(), Some("token"));
        store.clear().unwrap();
        assert_eq!(*secret.value.lock().unwrap(), None);
    }

    #[test]
    fn migrates_legacy_token_then_deletes_plaintext() {
        let secret = Arc::new(FakeSecret::default());
        let legacy = Arc::new(FakeLegacy::default());
        *legacy.value.lock().unwrap() = Some(b"legacy-token".to_vec());
        let store = store(secret.clone(), legacy.clone());
        assert_eq!(store.load().unwrap().as_deref(), Some("legacy-token"));
        assert_eq!(
            secret.value.lock().unwrap().as_deref(),
            Some("legacy-token")
        );
        assert_eq!(*legacy.value.lock().unwrap(), None);
    }

    #[test]
    fn unavailable_secure_store_never_reads_plaintext() {
        let secret = Arc::new(FakeSecret::default());
        *secret.get_error.lock().unwrap() = Some(StorageError::Unavailable);
        let legacy = Arc::new(FakeLegacy::default());
        *legacy.value.lock().unwrap() = Some(b"legacy-token".to_vec());
        assert_eq!(
            store(secret, legacy.clone()).load(),
            Err(StorageError::Unavailable)
        );
        assert_eq!(*legacy.reads.lock().unwrap(), 0);
        assert!(legacy.value.lock().unwrap().is_some());
    }

    #[test]
    fn failed_secure_write_retains_legacy_file() {
        let secret = Arc::new(FakeSecret::default());
        *secret.set_error.lock().unwrap() = Some(StorageError::WriteFailed);
        let legacy = Arc::new(FakeLegacy::default());
        *legacy.value.lock().unwrap() = Some(b"legacy-token".to_vec());
        assert_eq!(
            store(secret, legacy.clone()).load(),
            Err(StorageError::WriteFailed)
        );
        assert!(legacy.value.lock().unwrap().is_some());
    }

    #[test]
    fn malformed_and_oversized_legacy_tokens_are_rejected_and_retained() {
        for value in [
            b"token with spaces".to_vec(),
            vec![b'x'; MAX_TOKEN_BYTES + 1],
        ] {
            let secret = Arc::new(FakeSecret::default());
            let legacy = Arc::new(FakeLegacy::default());
            *legacy.value.lock().unwrap() = Some(value);
            assert_eq!(
                store(secret, legacy.clone()).load(),
                Err(StorageError::Corrupt)
            );
            assert!(legacy.value.lock().unwrap().is_some());
        }
    }

    #[test]
    fn migration_is_secure_first_and_idempotent() {
        let secret = Arc::new(FakeSecret::default());
        *secret.value.lock().unwrap() = Some("secure-token".into());
        let legacy = Arc::new(FakeLegacy::default());
        *legacy.value.lock().unwrap() = Some(b"stale-token".to_vec());
        let store = store(secret.clone(), legacy.clone());
        assert_eq!(store.load().unwrap().as_deref(), Some("secure-token"));
        assert_eq!(*legacy.reads.lock().unwrap(), 0);
        assert_eq!(*secret.sets.lock().unwrap(), 0);
        assert_eq!(*legacy.value.lock().unwrap(), None);
        assert_eq!(store.load().unwrap().as_deref(), Some("secure-token"));
    }

    #[test]
    fn clear_removes_secure_and_legacy_values() {
        let secret = Arc::new(FakeSecret::default());
        *secret.value.lock().unwrap() = Some("secure-token".into());
        let legacy = Arc::new(FakeLegacy::default());
        *legacy.value.lock().unwrap() = Some(b"legacy-token".to_vec());
        store(secret.clone(), legacy.clone()).clear().unwrap();
        assert_eq!(*secret.value.lock().unwrap(), None);
        assert_eq!(*legacy.value.lock().unwrap(), None);
    }

    #[test]
    fn clear_still_removes_legacy_file_when_secure_delete_fails() {
        let secret = Arc::new(FakeSecret::default());
        *secret.delete_error.lock().unwrap() = Some(StorageError::DeleteFailed);
        let legacy = Arc::new(FakeLegacy::default());
        *legacy.value.lock().unwrap() = Some(b"legacy-token".to_vec());

        assert_eq!(
            store(secret, legacy.clone()).clear(),
            Err(StorageError::DeleteFailed)
        );
        assert_eq!(*legacy.value.lock().unwrap(), None);
    }

    #[cfg(not(target_os = "android"))]
    #[test]
    fn legacy_path_is_exactly_the_previous_app_data_path() {
        assert_eq!(
            legacy_token_path_from_dirs(
                Some(std::path::PathBuf::from("/tmp/data")),
                Some(std::path::PathBuf::from("/tmp/home")),
            )
            .unwrap(),
            std::path::PathBuf::from("/tmp/data/roomies/session.token")
        );
        assert_eq!(
            legacy_token_path_from_dirs(None, Some(std::path::PathBuf::from("/tmp/home"))).unwrap(),
            std::path::PathBuf::from("/tmp/home/.local/share/roomies/session.token")
        );
    }
}
