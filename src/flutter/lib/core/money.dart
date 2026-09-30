int? parseMoneyCents(String value) {
  final trimmed = value.trim();
  if (trimmed.isEmpty || trimmed.startsWith('-')) {
    return null;
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

String formatMoney(double amount) => amount.toStringAsFixed(2);

List<Map<String, dynamic>> parseSplitEntries(String value, int totalCents) {
  final entries = <Map<String, dynamic>>[];
  var sum = 0;
  for (final item in value.split(',')) {
    final trimmed = item.trim();
    if (trimmed.isEmpty) continue;
    final parts = trimmed.split(':');
    if (parts.length != 2) {
      throw FormatException('Custom splits use user-id:amount, separated by commas');
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
    throw FormatException('Custom split amounts must exactly equal the expense amount');
  }
  return entries;
}
