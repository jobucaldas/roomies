import 'package:web/web.dart' as web;

import 'workos_url.dart';

void openExternalUrl(String url) {
  if (!isWorkOSAuthorizeUrl(url)) {
    throw ArgumentError(
        'Refusing to navigate to an untrusted authorization URL');
  }
  web.window.location.href = url;
}
