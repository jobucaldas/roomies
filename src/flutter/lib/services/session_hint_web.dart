import 'package:web/web.dart' as web;

bool hasSessionHint() {
  for (final part in web.document.cookie.split(';')) {
    final cookie = part.trim();
    if (cookie.startsWith('roomies_session_hint=')) {
      return cookie.length > 'roomies_session_hint='.length;
    }
  }
  return false;
}
