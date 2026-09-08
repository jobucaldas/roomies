# Roomies patch provenance

Source: `android-native-keyring-store` 1.0.0 as published on crates.io, checksum `48c6349ddff23194f8fdce2ea8849380f5a4868c1648965b70e801e104cba9b3`.

Roomies carries this narrow patch because 1.0.0 calls JNI `exception_describe`, which writes Java/provider details directly to logcat, and ignores a false result from synchronous `SharedPreferences.commit`. The patch clears pending exceptions and maps an otherwise-successful JNI call to the provider's existing `JavaExceptionThrow` error without describing it. It also removes two provider-detail logging sites and maps failed named-vault commits to a new generic provider error. No cryptography, Keystore, vault format, or credential naming is changed.

Remove this override once an upstream release provides equivalent non-logging exception handling and has been verified against Roomies' Android build.
