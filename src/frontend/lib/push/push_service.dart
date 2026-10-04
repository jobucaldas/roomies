import 'package:flutter/foundation.dart';

import '../api/api_client.dart';
import 'browser_push.dart';

Future<String?> enableBrowserPush(ApiClient api, String houseId) async {
  if (!kIsWeb) {
    return 'Browser push is unsupported in the native desktop app.';
  }
  try {
    final key = await api.getVapidPublicKey();
    if (key.publicKey.isEmpty) {
      return 'Browser push is not configured by this server.';
    }
    final subscription = await subscribeBrowserPush(key.publicKey);
    await api.createNotificationSubscription(houseId, {
      'platform': 'web_push',
      'device_label': subscription['label'] ?? 'Browser',
      'endpoint': subscription['endpoint'],
      'p256dh': subscription['p256dh'],
      'auth': subscription['auth'],
    });
    return null;
  } catch (error) {
    return error.toString().replaceFirst('Exception: ', '');
  }
}

Future<String?> disableBrowserPush(
  ApiClient api,
  String houseId,
  List<String> subscriptionIds,
) async {
  if (!kIsWeb) {
    return 'Browser push is unsupported in the native desktop app.';
  }
  try {
    for (final id in subscriptionIds) {
      await api.deleteNotificationSubscription(houseId, id);
    }
    await unsubscribeBrowserPush();
    return null;
  } catch (error) {
    return error.toString().replaceFirst('Exception: ', '');
  }
}
