import 'package:flutter/foundation.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:shared_preferences/shared_preferences.dart';

/// Native builds persist the bearer token in the platform's protected store
/// (Android Keystore, iOS/macOS Keychain, libsecret on Linux, DPAPI on
/// Windows), never in plain preferences. Web builds must not store it at all:
/// the browser session is an HttpOnly cookie, and this key is only cleared so
/// older localStorage values cannot be read by page script.
const sessionTokenKey = 'roomies.session.token';
const pendingInvitationKey = 'roomies.pending.invitation';

class SessionStorage {
  SessionStorage({FlutterSecureStorage? secureStorage})
      : _secure = secureStorage ?? const FlutterSecureStorage();

  final FlutterSecureStorage _secure;

  Future<String?> loadToken() async {
    if (kIsWeb) return null;
    // Older builds kept the token in plain preferences. Take it out first so
    // it is gone from disk even if the protected store is unavailable.
    final prefs = await SharedPreferences.getInstance();
    final legacy = prefs.getString(sessionTokenKey);
    if (legacy != null) await prefs.remove(sessionTokenKey);
    try {
      final token = await _secure.read(key: sessionTokenKey);
      if (token != null && token.isNotEmpty) return token;
      if (legacy == null || legacy.isEmpty) return null;
      await _secure.write(key: sessionTokenKey, value: legacy);
      return legacy;
    } catch (error) {
      // No usable keystore (e.g. a Linux session without a Secret Service):
      // sign in again rather than fall back to plaintext.
      debugPrint('secure session storage unavailable: $error');
      return null;
    }
  }

  Future<void> saveToken(String token) async {
    if (kIsWeb) return;
    final prefs = await SharedPreferences.getInstance();
    await prefs.remove(sessionTokenKey);
    try {
      await _secure.write(key: sessionTokenKey, value: token);
    } catch (error) {
      // The session still works for this run; it just is not remembered.
      debugPrint('secure session storage unavailable: $error');
    }
  }

  Future<void> clearToken() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.remove(sessionTokenKey);
    if (kIsWeb) return;
    try {
      await _secure.delete(key: sessionTokenKey);
    } catch (error) {
      debugPrint('secure session storage unavailable: $error');
    }
  }

  Future<String?> loadPendingInvitation() async {
    final prefs = await SharedPreferences.getInstance();
    return prefs.getString(pendingInvitationKey);
  }

  Future<void> savePendingInvitation(String token) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(pendingInvitationKey, token);
  }

  Future<void> clearPendingInvitation() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.remove(pendingInvitationKey);
  }
}
