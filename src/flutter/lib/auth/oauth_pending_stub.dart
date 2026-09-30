import '../services/session_storage.dart';

Future<void> saveOAuthPending({
  required String state,
  required String codeVerifier,
}) async {
  final storage = SessionStorage();
  await storage.saveOAuthPending(state: state, codeVerifier: codeVerifier);
}

Future<({String? state, String? codeVerifier})> loadOAuthPending() async {
  final storage = SessionStorage();
  return storage.loadOAuthPending();
}

Future<void> clearOAuthPending() async {
  final storage = SessionStorage();
  await storage.clearOAuthPending();
}
