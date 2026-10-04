import 'datetime_format.dart';

class ScheduledEventRequest {
  ScheduledEventRequest({
    required this.title,
    required this.timezone,
    required this.dtstartLocal,
    required this.rrule,
    required this.exdates,
    this.enabled = true,
  });

  final String title;
  final String timezone;
  final String dtstartLocal;
  final String rrule;
  final List<String> exdates;
  final bool enabled;

  Map<String, dynamic> toJson() => {
        'title': title,
        'timezone': timezone,
        'dtstart_local': dtstartLocal,
        'rrule': rrule,
        'exdates': exdates,
        'enabled': enabled,
      };
}

(String freq, String interval, String count, String until) recurrenceFields(
  String rule,
) {
  String field(String name) {
    for (final part in rule.split(';')) {
      if (part.startsWith(name)) {
        return part.substring(name.length);
      }
    }
    return '';
  }

  final interval = field('INTERVAL=');
  return (
    field('FREQ='),
    interval.isEmpty ? '1' : interval,
    field('COUNT='),
    field('UNTIL='),
  );
}

/// RRULE parts the form cannot edit (BYDAY, BYMONTHDAY, ...), kept verbatim
/// so a summary never hides a qualifier that changes when events occur.
String recurrenceExtras(String rule) => rule
    .split(';')
    .where((part) => part.isNotEmpty)
    .where((part) =>
        !const ['FREQ=', 'INTERVAL=', 'COUNT=', 'UNTIL='].any(part.startsWith))
    .join(';');

String minutesTime(int? minutes) {
  if (minutes == null) return '';
  return '${(minutes ~/ 60).toString().padLeft(2, '0')}:${(minutes % 60).toString().padLeft(2, '0')}';
}

int? timeMinutes(String value) {
  final parts = value.split(':');
  if (parts.length != 2) return null;
  final hours = int.tryParse(parts[0]);
  final mins = int.tryParse(parts[1]);
  if (hours == null || mins == null) return null;
  final total = hours * 60 + mins;
  if (total >= 1440) return null;
  return total;
}

ScheduledEventRequest buildScheduledEventRequest({
  required String title,
  required String start,
  required String zone,
  required String freq,
  required String interval,
  required String count,
  required String until,
  required String exdates,
}) {
  final trimmedTitle = title.trim();
  if (trimmedTitle.isEmpty || trimmedTitle.length > 120) {
    throw FormatException('Title must contain 1 to 120 characters.');
  }
  final intervalValue = int.tryParse(interval);
  if (intervalValue == null || intervalValue < 1 || intervalValue > 366) {
    throw FormatException('Interval must be 1–366.');
  }
  var rule = 'FREQ=$freq;INTERVAL=$intervalValue';
  if (count.trim().isNotEmpty) {
    final countValue = int.tryParse(count);
    if (countValue == null || countValue < 1 || countValue > 366) {
      throw FormatException('Count must be 1–366.');
    }
    rule += ';COUNT=$countValue';
  } else if (until.trim().isEmpty) {
    throw FormatException('Provide a count or an until date.');
  } else {
    rule += ';UNTIL=${until.trim()}';
  }
  final dates = exdates
      .split(',')
      .map((value) => value.trim())
      .where((value) => value.isNotEmpty)
      .toList();
  if (dates.length > 100) {
    throw FormatException('At most 100 exception dates are allowed.');
  }
  if (start.length != 16 || zone.trim().isEmpty) {
    throw FormatException('Local start and IANA timezone are required.');
  }
  if (count.trim().isEmpty) {
    final untilAt = tryParseApiDateTime(until);
    if (untilAt == null) {
      throw FormatException('Until must be a valid date.');
    }
    // Compare wall-clock values; a trailing Z must not shift the comparison.
    final naive = DateTime(untilAt.year, untilAt.month, untilAt.day,
        untilAt.hour, untilAt.minute, untilAt.second);
    if (!naive.isAfter(DateTime.parse(start))) {
      throw FormatException('Until must be after the local start.');
    }
  }
  return ScheduledEventRequest(
    title: trimmedTitle,
    timezone: zone.trim(),
    dtstartLocal: '$start:00',
    rrule: rule,
    exdates: dates,
  );
}

bool notificationControlsReady({required bool loading, String? error}) {
  return !loading && error == null;
}

String browserPushInitialStatus({required bool isWeb}) {
  if (isWeb) return 'Browser push is disabled.';
  return 'Browser push is unsupported in the native desktop app.';
}
