# Roomies patch notes

Source: `android-native-keyring-store` 1.0.0 (crates.io checksum `48c6349ddff23194f8fdce2ea8849380f5a4868c1648965b70e801e104cba9b3`).

## Changes
- Clear pending JNI exceptions instead of calling `exception_describe`.
- Treat a false `SharedPreferences.commit` result as `JavaExceptionThrow`.
- Remove provider-detail logging.
- Make named-store `Debug` output opaque.
- Map failed named-vault commits to a generic provider error.

Cryptography, Keystore usage, vault format, and credential naming are unchanged.

Remove this override when an upstream release provides equivalent exception handling and has been verified against the Roomies Android build.
