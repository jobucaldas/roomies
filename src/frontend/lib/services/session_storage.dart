import 'package:shared_preferences/shared_preferences.dart';

/// Native builds may persist a bearer token. Web builds must not: the
/// browser session is an HttpOnly cookie, and this key is only cleared so
/// older localStorage values cannot be read by page script.
const sessionTokenKey = 'roomies.session.token';
const pendingInvitationKey = 'roomies.pending.invitation';

class SessionStorage {
  Future<String?> loadToken() async {
    final prefs = await SharedPreferences.getInstance();
    final token = prefs.getString(sessionTokenKey);
    if (token == null || token.isEmpty) return null;
    return token;
  }

  Future<void> saveToken(String token) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(sessionTokenKey, token);
  }

  Future<void> clearToken() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.remove(sessionTokenKey);
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
