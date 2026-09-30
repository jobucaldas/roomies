import 'package:shared_preferences/shared_preferences.dart';

const sessionTokenKey = 'roomies.session.token';
const pendingInvitationKey = 'roomies.pending.invitation';
const oauthPendingStateKey = 'roomies.oauth.state';
const oauthPendingVerifierKey = 'roomies.oauth.code_verifier';

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

  /// Stores AuthKit PKCE material for native targets.
  /// Web uses sessionStorage via [oauth_pending_web.dart] instead.
  Future<void> saveOAuthPending({
    required String state,
    required String codeVerifier,
  }) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(oauthPendingStateKey, state);
    await prefs.setString(oauthPendingVerifierKey, codeVerifier);
  }

  Future<({String? state, String? codeVerifier})> loadOAuthPending() async {
    final prefs = await SharedPreferences.getInstance();
    return (
      state: prefs.getString(oauthPendingStateKey),
      codeVerifier: prefs.getString(oauthPendingVerifierKey),
    );
  }

  Future<void> clearOAuthPending() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.remove(oauthPendingStateKey);
    await prefs.remove(oauthPendingVerifierKey);
  }
}
