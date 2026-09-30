import 'package:web/web.dart' as web;

const _stateKey = 'roomies.oauth.state';
const _verifierKey = 'roomies.oauth.code_verifier';

Future<void> saveOAuthPending({
  required String state,
  required String codeVerifier,
}) async {
  web.window.sessionStorage.setItem(_stateKey, state);
  web.window.sessionStorage.setItem(_verifierKey, codeVerifier);
}

Future<({String? state, String? codeVerifier})> loadOAuthPending() async {
  return (
    state: web.window.sessionStorage.getItem(_stateKey),
    codeVerifier: web.window.sessionStorage.getItem(_verifierKey),
  );
}

Future<void> clearOAuthPending() async {
  web.window.sessionStorage.removeItem(_stateKey);
  web.window.sessionStorage.removeItem(_verifierKey);
}
