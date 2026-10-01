import 'package:flutter_test/flutter_test.dart';
import 'package:roomies/core/money.dart';

void main() {
  test('parseMoneyCents accepts two decimal places', () {
    expect(parseMoneyCents('21.00'), 2100);
    expect(parseMoneyCents('10.00'), 1000);
  });

  test('parseMoneyCents accepts Brazilian comma decimals', () {
    expect(parseMoneyCents('21,50'), 2150);
    expect(parseMoneyCents('1.234,56'), 123456);
  });

  test('parseSplitEntries requires exact total', () {
    final entries = parseSplitEntries('a:7.00, b:3.00', 1000);
    expect(entries.length, 2);
    expect(entries[0]['amount'], 7.0);
  });

  test('formatMoney uses USD for English and BRL for Portuguese', () {
    expect(formatMoney(3, localeCode: 'en'), contains('3.00'));
    expect(formatMoney(3, localeCode: 'en'), contains(r'$'));
    expect(formatMoney(3, localeCode: 'pt'), contains('3,00'));
    expect(formatMoney(3, localeCode: 'pt'), contains('R\$'));
  });
}
