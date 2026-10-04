import 'dart:js_interop';

@JS('roomiesSubscribePush')
external JSPromise<JSAny?> roomiesSubscribePush(
  JSString vapid,
  JSString label,
);

@JS('roomiesUnsubscribePush')
external JSPromise<JSAny?> roomiesUnsubscribePush();

Future<Map<String, String>> subscribeBrowserPush(String vapidPublicKey) async {
  final result = await roomiesSubscribePush(vapidPublicKey.toJS, 'Browser'.toJS)
      .toDart;
  final map = result?.dartify();
  if (map is! Map) {
    throw StateError('Browser push failed.');
  }
  final values = map.cast<String, dynamic>();
  return {
    'endpoint': values['endpoint']?.toString() ?? '',
    'p256dh': values['p256dh']?.toString() ?? '',
    'auth': values['auth']?.toString() ?? '',
    'label': values['label']?.toString() ?? 'Browser',
  };
}

Future<void> unsubscribeBrowserPush() async {
  await roomiesUnsubscribePush().toDart;
}
