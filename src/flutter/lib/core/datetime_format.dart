import 'package:intl/date_symbol_data_local.dart';
import 'package:intl/intl.dart';

bool _dateFormattingReady = false;

/// Load ICU date symbols for every UI language. Safe to call more than once.
Future<void> ensureDateFormatting() async {
  if (_dateFormattingReady) return;
  await Future.wait([
    initializeDateFormatting('en_US'),
    initializeDateFormatting('pt_BR'),
    initializeDateFormatting('es'),
  ]);
  _dateFormattingReady = true;
}

/// Parse API date / datetime strings into a [DateTime] when possible.
DateTime? tryParseApiDateTime(String raw) {
  final trimmed = raw.trim();
  if (trimmed.isEmpty) return null;
  final parsed = DateTime.tryParse(trimmed);
  if (parsed != null) return parsed;
  // YYYYMMDDTHHMMSS (scheduled until / EXDATE style)
  if (RegExp(r'^\d{8}T\d{6}$').hasMatch(trimmed)) {
    return DateTime.tryParse(
      '${trimmed.substring(0, 4)}-${trimmed.substring(4, 6)}-${trimmed.substring(6, 8)}'
      'T${trimmed.substring(9, 11)}:${trimmed.substring(11, 13)}:${trimmed.substring(13, 15)}',
    );
  }
  return null;
}

String _localeTag(String localeCode) => switch (localeCode) {
      'pt' => 'pt_BR',
      'es' => 'es',
      _ => 'en_US',
    };

String _fallbackDate(DateTime dt) =>
    '${dt.year}-${dt.month.toString().padLeft(2, '0')}-${dt.day.toString().padLeft(2, '0')}';

String _fallbackDateTime(DateTime dt) =>
    '${_fallbackDate(dt)} · ${dt.hour.toString().padLeft(2, '0')}:${dt.minute.toString().padLeft(2, '0')}';

/// Human-friendly date for money rows (e.g. `Oct 1, 2026` / `1 de out. de 2026`).
String formatDisplayDate(String raw, {String localeCode = 'en'}) {
  final dt = tryParseApiDateTime(raw);
  if (dt == null) return raw;
  try {
    return DateFormat.yMMMd(_localeTag(localeCode)).format(dt);
  } catch (_) {
    return _fallbackDate(dt);
  }
}

/// Human-friendly local datetime (e.g. `Oct 5, 2026 · 10:00`).
String formatDisplayDateTime(String raw, {String localeCode = 'en'}) {
  final dt = tryParseApiDateTime(raw);
  if (dt == null) return raw;
  try {
    final date = DateFormat.yMMMd(_localeTag(localeCode)).format(dt);
    final time = DateFormat.Hm(_localeTag(localeCode)).format(dt);
    return '$date · $time';
  } catch (_) {
    return _fallbackDateTime(dt);
  }
}
