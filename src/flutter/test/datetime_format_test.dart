import 'package:flutter_test/flutter_test.dart';
import 'package:roomies/core/datetime_format.dart';

void main() {
  setUpAll(() async {
    await ensureDateFormatting();
  });

  test('formatDisplayDateTime turns ISO into locale-friendly text', () {
    expect(
      formatDisplayDateTime('2026-10-05T10:00:00Z', localeCode: 'en'),
      'Oct 5, 2026 · 10:00',
    );
    final pt = formatDisplayDateTime('2026-10-05T10:00:00Z', localeCode: 'pt');
    expect(pt, contains('2026'));
    expect(pt, isNot(contains('T10:00')));
    expect(pt, contains('·'));
  });

  test('formatDisplayDate omits time for date-only values', () {
    expect(
      formatDisplayDate('2026-10-01', localeCode: 'en'),
      'Oct 1, 2026',
    );
    expect(
      formatDisplayDate('not-a-date', localeCode: 'en'),
      'not-a-date',
    );
  });

  test('tryParseApiDateTime accepts compact scheduled stamps', () {
    final parsed = tryParseApiDateTime('20251231T090000');
    expect(parsed, isNotNull);
    expect(parsed!.year, 2025);
    expect(parsed.month, 12);
    expect(parsed.day, 31);
    expect(parsed.hour, 9);
  });
}
