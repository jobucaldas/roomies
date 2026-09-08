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
trait LegacyTokenCandidate: Send {
    fn read(&mut self) -> Result<Vec<u8>, StorageError>;
    fn delete_verified(self: Box<Self>) -> Result<(), StorageError>;
}

#[cfg(not(target_arch = "wasm32"))]
trait LegacyTokenFile: Send + Sync {
    fn open_candidate(&self) -> Result<Option<Box<dyn LegacyTokenCandidate>>, StorageError>;
    fn delete_path(&self) -> Result<(), StorageError>;
}

#[cfg(not(target_arch = "wasm32"))]
static NATIVE_OPERATION_LOCK: std::sync::Mutex<()> = std::sync::Mutex::new(());

#[cfg(not(target_arch = "wasm32"))]
struct NativeSessionTokenStore {
    secret: std::sync::Arc<dyn SessionSecretStore>,
    legacy: std::sync::Arc<dyn LegacyTokenFile>,
    #[cfg(target_os = "linux")]
    linux_interprocess_lock: bool,
}

#[cfg(not(target_arch = "wasm32"))]
impl NativeSessionTokenStore {
    fn production() -> Self {
        Self {
            secret: std::sync::Arc::new(PlatformSecretStore::new()),
            legacy: std::sync::Arc::new(FilesystemLegacyTokenFile {
                path: legacy_token_path(),
            }),
            #[cfg(target_os = "linux")]
            linux_interprocess_lock: true,
        }
    }

    #[cfg(test)]
    fn with_backends(
        secret: std::sync::Arc<dyn SessionSecretStore>,
        legacy: std::sync::Arc<dyn LegacyTokenFile>,
    ) -> Self {
        Self {
            secret,
            legacy,
            #[cfg(target_os = "linux")]
            linux_interprocess_lock: false,
        }
    }

    fn operation_guard(&self) -> Result<NativeOperationGuard<'_>, StorageError> {
        let process_guard = NATIVE_OPERATION_LOCK
            .lock()
            .map_err(|_| StorageError::Unavailable)?;
        #[cfg(target_os = "linux")]
        let interprocess_guard = if self.linux_interprocess_lock {
            Some(LinuxInterprocessLock::acquire()?)
        } else {
            None
        };
        Ok(NativeOperationGuard {
            _process_guard: process_guard,
            #[cfg(target_os = "linux")]
            _interprocess_guard: interprocess_guard,
        })
    }
}

#[cfg(not(target_arch = "wasm32"))]
struct NativeOperationGuard<'a> {
    _process_guard: std::sync::MutexGuard<'a, ()>,
    #[cfg(target_os = "linux")]
    _interprocess_guard: Option<LinuxInterprocessLock>,
}

#[cfg(not(target_arch = "wasm32"))]
impl SessionTokenStore for NativeSessionTokenStore {
    fn load(&self) -> Result<Option<String>, StorageError> {
        let _guard = self.operation_guard()?;
        if let Some(token) = self.secret.get()? {
            let token = validate_token(token.as_bytes())?;
            // A prior migration may have stored the secret immediately before a
            // crash or failed unlink. Never read that plaintext again.
            self.legacy.delete_path()?;
            return Ok(Some(token));
        }

        let Some(mut candidate) = self.legacy.open_candidate()? else {
            return Ok(None);
        };
        let token = validate_token(&candidate.read()?)?;
        self.secret.set(&token)?;
        match self.secret.get()? {
            Some(saved) if saved == token => {}
            Some(_) => return Err(StorageError::WriteFailed),
            None => return Err(StorageError::WriteFailed),
        }
        candidate.delete_verified()?;
        Ok(Some(token))
    }

    fn save(&self, token: &str) -> Result<(), StorageError> {
        let token = validate_token(token.as_bytes()).map_err(|_| StorageError::WriteFailed)?;
        let _guard = self.operation_guard()?;
        self.secret.set(&token)?;
        match self.secret.get()? {
            Some(saved) if saved == token => Ok(()),
            _ => Err(StorageError::WriteFailed),
        }
    }

    fn clear(&self) -> Result<(), StorageError> {
        let _guard = self.operation_guard()?;
        let secret_result = self
            .secret
            .delete()
            .and_then(|()| match self.secret.get()? {
                None => Ok(()),
                Some(_) => Err(StorageError::DeleteFailed),
            });
        let legacy_result = self.legacy.delete_path();
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

#[cfg(target_os = "linux")]
struct LinuxInterprocessLock(std::fs::File);

#[cfg(target_os = "linux")]
impl LinuxInterprocessLock {
    fn acquire() -> Result<Self, StorageError> {
        use std::os::fd::AsRawFd;
        use std::os::unix::fs::{MetadataExt, OpenOptionsExt, PermissionsExt};

        let directory = std::env::var_os("XDG_RUNTIME_DIR")
            .map(std::path::PathBuf::from)
            .unwrap_or_else(|| {
                std::path::PathBuf::from(format!("/run/user/{}", unsafe { libc::geteuid() }))
            });
        let directory_metadata =
            std::fs::metadata(&directory).map_err(|_| StorageError::Unavailable)?;
        if !directory_metadata.is_dir()
            || directory_metadata.uid() != unsafe { libc::geteuid() }
            || directory_metadata.permissions().mode() & 0o077 != 0
        {
            return Err(StorageError::Unavailable);
        }
        let path = directory.join("app.roomies-session.lock");
        let file = std::fs::OpenOptions::new()
            .read(true)
            .write(true)
            .create(true)
            .mode(0o600)
            .custom_flags(libc::O_CLOEXEC | libc::O_NOFOLLOW)
            .open(path)
            .map_err(|_| StorageError::Unavailable)?;
        let metadata = file.metadata().map_err(|_| StorageError::Unavailable)?;
        if !metadata.is_file()
            || metadata.uid() != unsafe { libc::geteuid() }
            || metadata.permissions().mode() & 0o077 != 0
        {
            return Err(StorageError::Unavailable);
        }

        let deadline = std::time::Instant::now() + std::time::Duration::from_secs(2);
        loop {
            let result = unsafe { libc::flock(file.as_raw_fd(), libc::LOCK_EX | libc::LOCK_NB) };
            if result == 0 {
                return Ok(Self(file));
            }
            let error = std::io::Error::last_os_error();
            if !matches!(error.raw_os_error(), Some(code) if code == libc::EWOULDBLOCK || code == libc::EAGAIN)
                || std::time::Instant::now() >= deadline
            {
                return Err(StorageError::Unavailable);
            }
            std::thread::sleep(std::time::Duration::from_millis(10));
        }
    }
}

#[cfg(target_os = "linux")]
impl Drop for LinuxInterprocessLock {
    fn drop(&mut self) {
        use std::os::fd::AsRawFd;
        unsafe {
            libc::flock(self.0.as_raw_fd(), libc::LOCK_UN);
        }
    }
}

#[cfg(not(target_arch = "wasm32"))]
struct FilesystemLegacyTokenFile {
    path: Option<std::path::PathBuf>,
}

#[cfg(not(target_arch = "wasm32"))]
struct FilesystemLegacyTokenCandidate {
    file: std::fs::File,
    path: std::path::PathBuf,
    device: u64,
    inode: u64,
}

#[cfg(not(target_arch = "wasm32"))]
impl LegacyTokenCandidate for FilesystemLegacyTokenCandidate {
    fn read(&mut self) -> Result<Vec<u8>, StorageError> {
        use std::io::Read;

        let mut bytes = Vec::new();
        self.file
            .by_ref()
            .take((MAX_TOKEN_BYTES + 1) as u64)
            .read_to_end(&mut bytes)
            .map_err(|_| StorageError::Unavailable)?;
        if bytes.len() > MAX_TOKEN_BYTES {
            return Err(StorageError::Corrupt);
        }
        Ok(bytes)
    }

    fn delete_verified(self: Box<Self>) -> Result<(), StorageError> {
        use std::os::unix::fs::MetadataExt;

        let metadata =
            std::fs::symlink_metadata(&self.path).map_err(|_| StorageError::DeleteFailed)?;
        if !metadata.is_file() || metadata.dev() != self.device || metadata.ino() != self.inode {
            return Err(StorageError::DeleteFailed);
        }
        std::fs::remove_file(&self.path).map_err(|_| StorageError::DeleteFailed)
    }
}

#[cfg(not(target_arch = "wasm32"))]
impl LegacyTokenFile for FilesystemLegacyTokenFile {
    fn open_candidate(&self) -> Result<Option<Box<dyn LegacyTokenCandidate>>, StorageError> {
        use std::os::unix::fs::{MetadataExt, OpenOptionsExt, PermissionsExt};

        let Some(path) = &self.path else {
            return Ok(None);
        };
        let file = match std::fs::OpenOptions::new()
            .read(true)
            .custom_flags(libc::O_CLOEXEC | libc::O_NOFOLLOW | libc::O_NONBLOCK)
            .open(path)
        {
            Ok(file) => file,
            Err(error) if error.kind() == std::io::ErrorKind::NotFound => return Ok(None),
            Err(_) => return Err(StorageError::Corrupt),
        };
        let metadata = file.metadata().map_err(|_| StorageError::Corrupt)?;
        if !metadata.is_file()
            || metadata.uid() != unsafe { libc::geteuid() }
            || metadata.permissions().mode() & 0o077 != 0
        {
            return Err(StorageError::Corrupt);
        }
        Ok(Some(Box::new(FilesystemLegacyTokenCandidate {
            device: metadata.dev(),
            inode: metadata.ino(),
            file,
            path: path.clone(),
        })))
    }

    fn delete_path(&self) -> Result<(), StorageError> {
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
        ignore_set: Mutex<bool>,
        ignore_delete: Mutex<bool>,
        set_started: Mutex<Option<std::sync::mpsc::Sender<()>>>,
        set_release: Mutex<Option<std::sync::mpsc::Receiver<()>>>,
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
            if let Some(started) = self.set_started.lock().unwrap().take() {
                let _ = started.send(());
            }
            if let Some(release) = self.set_release.lock().unwrap().take() {
                let _ = release.recv();
            }
            if let Some(error) = *self.set_error.lock().unwrap() {
                return Err(error);
            }
            if !*self.ignore_set.lock().unwrap() {
                *self.value.lock().unwrap() = Some(token.to_owned());
            }
            Ok(())
        }
        fn delete(&self) -> Result<(), StorageError> {
            *self.deletes.lock().unwrap() += 1;
            if let Some(error) = *self.delete_error.lock().unwrap() {
                return Err(error);
            }
            if !*self.ignore_delete.lock().unwrap() {
                *self.value.lock().unwrap() = None;
            }
            Ok(())
        }
    }

    #[derive(Default)]
    struct FakeLegacy {
        value: Arc<Mutex<Option<Vec<u8>>>>,
        reads: Arc<Mutex<usize>>,
        deletes: Arc<Mutex<usize>>,
    }

    struct FakeLegacyCandidate {
        value: Arc<Mutex<Option<Vec<u8>>>>,
        reads: Arc<Mutex<usize>>,
        deletes: Arc<Mutex<usize>>,
        bytes: Vec<u8>,
    }

    impl LegacyTokenCandidate for FakeLegacyCandidate {
        fn read(&mut self) -> Result<Vec<u8>, StorageError> {
            *self.reads.lock().unwrap() += 1;
            Ok(self.bytes.clone())
        }

        fn delete_verified(self: Box<Self>) -> Result<(), StorageError> {
            *self.deletes.lock().unwrap() += 1;
            *self.value.lock().unwrap() = None;
            Ok(())
        }
    }

    impl LegacyTokenFile for FakeLegacy {
        fn open_candidate(&self) -> Result<Option<Box<dyn LegacyTokenCandidate>>, StorageError> {
            Ok(self.value.lock().unwrap().clone().map(|bytes| {
                Box::new(FakeLegacyCandidate {
                    value: self.value.clone(),
                    reads: self.reads.clone(),
                    deletes: self.deletes.clone(),
                    bytes,
                }) as Box<dyn LegacyTokenCandidate>
            }))
        }

        fn delete_path(&self) -> Result<(), StorageError> {
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
    fn successful_set_without_durable_mutation_is_rejected() {
        let secret = Arc::new(FakeSecret::default());
        *secret.ignore_set.lock().unwrap() = true;
        let legacy = Arc::new(FakeLegacy::default());

        assert_eq!(
            store(secret, legacy).save("token"),
            Err(StorageError::WriteFailed)
        );
    }

    #[test]
    fn successful_delete_without_durable_mutation_is_rejected() {
        let secret = Arc::new(FakeSecret::default());
        *secret.value.lock().unwrap() = Some("token".into());
        *secret.ignore_delete.lock().unwrap() = true;
        let legacy = Arc::new(FakeLegacy::default());

        assert_eq!(
            store(secret, legacy).clear(),
            Err(StorageError::DeleteFailed)
        );
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
    fn successful_migration_write_without_durable_mutation_retains_legacy_file() {
        let secret = Arc::new(FakeSecret::default());
        *secret.ignore_set.lock().unwrap() = true;
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
    fn migration_waits_for_concurrent_save_and_does_not_overwrite_it() {
        let secret = Arc::new(FakeSecret::default());
        let legacy = Arc::new(FakeLegacy::default());
        *legacy.value.lock().unwrap() = Some(b"legacy-token".to_vec());
        let store = Arc::new(store(secret.clone(), legacy));
        let (started_tx, started_rx) = std::sync::mpsc::channel();
        let (release_tx, release_rx) = std::sync::mpsc::channel();
        *secret.set_started.lock().unwrap() = Some(started_tx);
        *secret.set_release.lock().unwrap() = Some(release_rx);

        let saving = {
            let store = store.clone();
            std::thread::spawn(move || store.save("new-token"))
        };
        started_rx.recv().unwrap();
        let loading = {
            let store = store.clone();
            std::thread::spawn(move || store.load())
        };
        release_tx.send(()).unwrap();

        assert_eq!(saving.join().unwrap(), Ok(()));
        assert_eq!(
            loading.join().unwrap().unwrap().as_deref(),
            Some("new-token")
        );
        assert_eq!(secret.value.lock().unwrap().as_deref(), Some("new-token"));
        assert_eq!(*secret.sets.lock().unwrap(), 1);
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

    #[cfg(all(unix, not(target_os = "android")))]
    fn filesystem_test_directory(name: &str) -> std::path::PathBuf {
        use std::os::unix::fs::PermissionsExt;

        let path = std::env::temp_dir().join(format!(
            "roomies-{name}-{}-{}",
            std::process::id(),
            std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap()
                .as_nanos()
        ));
        std::fs::create_dir(&path).unwrap();
        std::fs::set_permissions(&path, std::fs::Permissions::from_mode(0o700)).unwrap();
        path
    }

    #[cfg(all(unix, not(target_os = "android")))]
    #[test]
    fn legacy_migration_rejects_symlinks_without_deleting_them() {
        use std::os::unix::fs::{symlink, PermissionsExt};

        let directory = filesystem_test_directory("legacy-symlink");
        let target = directory.join("target");
        std::fs::write(&target, b"token").unwrap();
        std::fs::set_permissions(&target, std::fs::Permissions::from_mode(0o600)).unwrap();
        let path = directory.join("session.token");
        symlink(&target, &path).unwrap();
        let legacy = FilesystemLegacyTokenFile {
            path: Some(path.clone()),
        };

        assert!(matches!(
            legacy.open_candidate(),
            Err(StorageError::Corrupt)
        ));
        assert!(std::fs::symlink_metadata(&path)
            .unwrap()
            .file_type()
            .is_symlink());
        std::fs::remove_dir_all(directory).unwrap();
    }

    #[cfg(all(unix, not(target_os = "android")))]
    #[test]
    fn legacy_migration_rejects_group_or_other_permissions() {
        use std::os::unix::fs::PermissionsExt;

        let directory = filesystem_test_directory("legacy-mode");
        let path = directory.join("session.token");
        std::fs::write(&path, b"token").unwrap();
        std::fs::set_permissions(&path, std::fs::Permissions::from_mode(0o640)).unwrap();
        let legacy = FilesystemLegacyTokenFile {
            path: Some(path.clone()),
        };

        assert!(matches!(
            legacy.open_candidate(),
            Err(StorageError::Corrupt)
        ));
        assert!(path.exists());
        std::fs::remove_dir_all(directory).unwrap();
    }

    #[cfg(all(unix, not(target_os = "android")))]
    #[test]
    fn legacy_migration_rejects_fifo_without_blocking() {
        use std::ffi::CString;
        use std::os::unix::ffi::OsStrExt;

        let directory = filesystem_test_directory("legacy-fifo");
        let path = directory.join("session.token");
        let c_path = CString::new(path.as_os_str().as_bytes()).unwrap();
        assert_eq!(unsafe { libc::mkfifo(c_path.as_ptr(), 0o600) }, 0);
        let legacy = FilesystemLegacyTokenFile {
            path: Some(path.clone()),
        };

        assert!(matches!(
            legacy.open_candidate(),
            Err(StorageError::Corrupt)
        ));
        assert!(path.exists());
        std::fs::remove_dir_all(directory).unwrap();
    }

    #[cfg(all(unix, not(target_os = "android")))]
    #[test]
    fn legacy_candidate_refuses_to_unlink_a_replacement() {
        use std::os::unix::fs::PermissionsExt;

        let directory = filesystem_test_directory("legacy-replacement");
        let path = directory.join("session.token");
        std::fs::write(&path, b"old-token").unwrap();
        std::fs::set_permissions(&path, std::fs::Permissions::from_mode(0o600)).unwrap();
        let legacy = FilesystemLegacyTokenFile {
            path: Some(path.clone()),
        };
        let mut candidate = legacy.open_candidate().unwrap().unwrap();
        assert_eq!(candidate.read().unwrap(), b"old-token");
        std::fs::rename(&path, directory.join("original")).unwrap();
        std::fs::write(&path, b"new-token").unwrap();
        std::fs::set_permissions(&path, std::fs::Permissions::from_mode(0o600)).unwrap();

        assert_eq!(candidate.delete_verified(), Err(StorageError::DeleteFailed));
        assert_eq!(std::fs::read(&path).unwrap(), b"new-token");
        std::fs::remove_dir_all(directory).unwrap();
    }

    #[test]
    fn vendored_android_provider_does_not_describe_jni_exceptions() {
        let vault =
            include_str!("../../../vendor/android-native-keyring-store/src/by_store/vault.rs");
        let legacy =
            include_str!("../../../vendor/android-native-keyring-store/src/by_service/mod.rs");
        let preferences =
            include_str!("../../../vendor/android-native-keyring-store/src/shared_preferences.rs");
        let credentials =
            include_str!("../../../vendor/android-native-keyring-store/src/by_store/cred.rs");
        assert!(!vault.contains("exception_describe"));
        assert!(!legacy.contains("exception_describe"));
        assert!(!preferences.contains("tracing::error"));
        assert!(!preferences.contains("tracing::debug"));
        assert!(!credentials.contains(".commit(env)?;"));
        assert!(credentials.contains("CommitFailed"));
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
