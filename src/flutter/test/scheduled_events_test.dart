import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:roomies/core/scheduled_events.dart';

void main() {
  test('notification controls stay hidden until loading succeeds', () {
    expect(notificationControlsReady(loading: true, error: null), isFalse);
    expect(
      notificationControlsReady(loading: false, error: 'offline'),
      isFalse,
    );
    expect(notificationControlsReady(loading: false, error: null), isTrue);
  });

  test('service worker uses generic deduplicated payloads', () {
    final source = File('web/roomies-sw.js').readAsStringSync();
    expect(source, contains("body: 'You have a Roomies notification.'"));
    expect(source, contains(r'tag: `roomies-${id}`'));
    expect(source, contains(r'`/house/${encodeURIComponent(houseId)}`'));
    expect(source, contains('const path = houseId ?'));
    expect(source, isNot(contains('/houses/')));
    expect(source, isNot(contains('endpoint')));
  });

  test('recurrence fields prefill edit form', () {
    expect(
      recurrenceFields('FREQ=WEEKLY;INTERVAL=2;COUNT=3'),
      ('WEEKLY', '2', '3', ''),
    );
    expect(
      recurrenceFields('FREQ=MONTHLY;UNTIL=20251231T090000'),
      ('MONTHLY', '1', '', '20251231T090000'),
    );
  });

  test('recurrence request enforces client bounds', () {
    expect(
      () => buildScheduledEventRequest(
        title: 'Bins',
        start: '2025-01-02T09:00',
        zone: 'UTC',
        freq: 'DAILY',
        interval: '367',
        count: '1',
        until: '',
        exdates: '',
      ),
      throwsFormatException,
    );
    expect(
      () => buildScheduledEventRequest(
        title: 'Bins',
        start: '2025-01-02T09:00',
        zone: 'UTC',
        freq: 'DAILY',
        interval: '1',
        count: '',
        until: '',
        exdates: '',
      ),
      throwsFormatException,
    );
    expect(
      buildScheduledEventRequest(
        title: 'Bins',
        start: '2025-01-02T09:00',
        zone: 'UTC',
        freq: 'WEEKLY',
        interval: '1',
        count: '2',
        until: '',
        exdates: '2025-01-09T09:00',
      ).rrule,
      'FREQ=WEEKLY;INTERVAL=1;COUNT=2',
    );
  });
}
