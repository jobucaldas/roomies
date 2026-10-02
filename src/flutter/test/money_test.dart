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
    expect(formatMoney(1234.5, localeCode: 'es'), r'$1,234.50');
  });

  test('explicit currency keeps the language number style', () {
    expect(formatMoney(1234.5, localeCode: 'en', currency: 'EUR'), '€1,234.50');
    // pt-BR separates the symbol with a non-breaking space.
    expect(
      formatMoney(1234.5, localeCode: 'pt', currency: 'USD'),
      '\$\u00A01.234,50',
    );
    expect(formatMoney(1234.5, localeCode: 'es', currency: 'BRL'), r'R$1,234.50');
    expect(formatMoney(1234.5, localeCode: 'en', currency: 'CLP'), r'$1,235');
  });

  test('following the language matches the per-language default', () {
    expect(currencyForLanguage('en'), 'USD');
    expect(currencyForLanguage('pt'), 'BRL');
    expect(currencyForLanguage('es'), 'USD');
    for (final lang in ['en', 'pt', 'es']) {
      expect(
        formatMoney(42, localeCode: lang),
        formatMoney(42, localeCode: lang, currency: currencyForLanguage(lang)),
      );
    }
  });
}
