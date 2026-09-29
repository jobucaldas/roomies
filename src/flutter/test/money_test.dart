import 'package:flutter_test/flutter_test.dart';
import 'package:roomies/core/money.dart';

void main() {
  test('parseMoneyCents accepts two decimal places', () {
    expect(parseMoneyCents('21.00'), 2100);
    expect(parseMoneyCents('10.00'), 1000);
  });

  test('parseSplitEntries requires exact total', () {
    final entries = parseSplitEntries('a:7.00, b:3.00', 1000);
    expect(entries.length, 2);
    expect(entries[0]['amount'], 7.0);
  });
}
