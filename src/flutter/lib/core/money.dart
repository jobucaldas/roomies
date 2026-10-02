import 'package:intl/intl.dart';

int? parseMoneyCents(String value) {
  var trimmed = value.trim();
  if (trimmed.isEmpty || trimmed.startsWith('-')) {
    return null;
  }
  // Accept Brazilian decimal comma when there is no thousands-dot ambiguity.
  if (trimmed.contains(',') && !trimmed.contains('.')) {
    trimmed = trimmed.replaceAll(',', '.');
  } else if (trimmed.contains(',') && trimmed.contains('.')) {
    // pt-BR style 1.234,56 → 1234.56
    trimmed = trimmed.replaceAll('.', '').replaceAll(',', '.');
  }
  final parts = trimmed.split('.');
  if (parts.length > 2) return null;
  final whole = int.tryParse(parts[0]);
  if (whole == null) return null;
  final fraction = parts.length == 2 ? parts[1] : '';
  if (fraction.length > 2 || !RegExp(r'^\d*$').hasMatch(fraction)) {
    return null;
  }
  final fracCents = fraction.isEmpty
      ? 0
      : int.parse(fraction.padRight(2, '0').substring(0, 2));
  return whole * 100 + fracCents;
}

double centsToApiAmount(int cents) => cents / 100.0;

/// Format money for display. Portuguese uses BRL (reais); English uses USD;
/// Spanish uses Latin American number formatting with a `$` symbol.
String formatMoney(double amount, {String localeCode = 'en'}) {
  return switch (localeCode) {
    'pt' =>
      NumberFormat.currency(locale: 'pt_BR', symbol: r'R$ ').format(amount),
    'es' => NumberFormat.currency(locale: 'es_419', symbol: r'$').format(amount),
    _ => NumberFormat.currency(locale: 'en_US', symbol: r'$').format(amount),
  };
}

List<Map<String, dynamic>> parseSplitEntries(String value, int totalCents) {
  final entries = <Map<String, dynamic>>[];
  var sum = 0;
  for (final item in value.split(',')) {
    final trimmed = item.trim();
    if (trimmed.isEmpty) continue;
    final parts = trimmed.split(':');
    if (parts.length != 2) {
      throw FormatException(
          'Custom splits use user-id:amount, separated by commas');
    }
    final userId = parts[0].trim();
    if (userId.isEmpty) {
      throw FormatException('A split user id is required');
    }
    final cents = parseMoneyCents(parts[1]);
    if (cents == null) {
      throw FormatException('invalid amount');
    }
    sum += cents;
    entries.add({'user_id': userId, 'amount': centsToApiAmount(cents)});
  }
  if (entries.isEmpty || sum != totalCents) {
    throw FormatException(
        'Custom split amounts must exactly equal the expense amount');
  }
  return entries;
}
