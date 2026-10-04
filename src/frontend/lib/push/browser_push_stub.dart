Future<Map<String, String>> subscribeBrowserPush(String vapidPublicKey) async {
  throw UnsupportedError('Browser push is unsupported in the native desktop app.');
}

Future<void> unsubscribeBrowserPush() async {}
