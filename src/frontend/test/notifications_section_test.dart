import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:provider/provider.dart';
import 'package:roomies/api/api_client.dart';
import 'package:roomies/core/roles.dart';
import 'package:roomies/models/models.dart';
import 'package:roomies/screens/house/notifications_section.dart';
import 'package:roomies/state/app_state.dart';

class _FakeApi extends ApiClient {
  _FakeApi({this.devices = const [], this.events = const []})
      : super(baseUrl: 'http://example/api');

  final List<NotificationSubscription> devices;
  final List<ScheduledHouseEvent> events;

  @override
  Future<NotificationPreferences> getNotificationPreferences(String id) async =>
      NotificationPreferences(houseId: id, userId: 'u1');

  @override
  Future<List<NotificationSubscription>> getNotificationSubscriptions(
          String id) async =>
      devices;

  @override
  Future<List<ScheduledHouseEvent>> getScheduledEvents(String id) async =>
      events;
}

Future<void> _pump(
  WidgetTester tester,
  _FakeApi api, {
  required Size size,
  HouseRole? role = HouseRole.admin,
}) async {
  tester.view.physicalSize = size;
  tester.view.devicePixelRatio = 1;
  addTearDown(tester.view.reset);
  await tester.pumpWidget(
    ChangeNotifierProvider(
      create: (_) => AppState(api, deviceLocale: 'en'),
      child: MaterialApp(
        home: Scaffold(
          body: SingleChildScrollView(
            padding: const EdgeInsets.all(16),
            child: NotificationsSection(houseId: 'h1', role: role, userId: 'u1'),
          ),
        ),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

void main() {
  final device = NotificationSubscription(
    id: 's1',
    houseId: 'h1',
    userId: 'u1',
    platform: 'web_push',
    deviceLabel: 'A very long browser label ' * 6,
    createdAt: '2026-01-01T00:00:00Z',
    lastSeenAt: '2026-01-02T00:00:00Z',
  );
  final event = ScheduledHouseEvent(
    id: 'e1',
    houseId: 'h1',
    creatorId: 'u1',
    title: 'Take out the trash ' * 6,
    timezone: 'Europe/Lisbon',
    dtstartLocal: '2026-01-05T09:00:00',
    rrule: 'FREQ=WEEKLY;INTERVAL=1;COUNT=4',
    enabled: true,
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
  );

  for (final size in const [Size(320, 700), Size(390, 800), Size(1200, 900)]) {
    testWidgets('renders without overflow at ${size.width.toInt()}px',
        (tester) async {
      await _pump(tester, _FakeApi(devices: [device], events: [event]),
          size: size);
      expect(tester.takeException(), isNull);
      expect(find.byType(Card), findsNWidgets(3));
    });
  }

  testWidgets('empty state and monitor role render cleanly', (tester) async {
    await _pump(tester, _FakeApi(),
        size: const Size(390, 800), role: HouseRole.monitor);
    expect(tester.takeException(), isNull);
    expect(find.byType(TextFormField), findsNWidgets(4));
  });
}
