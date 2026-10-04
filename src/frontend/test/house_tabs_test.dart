import 'package:flutter_test/flutter_test.dart';
import 'package:roomies/core/house_tabs.dart';

void main() {
  test('HouseTabs indexOf maps known keys and defaults unknown', () {
    expect(HouseTabs.indexOf('notes'), 1);
    expect(HouseTabs.indexOf('groceries'), 2);
    expect(HouseTabs.indexOf('NOTES'), 1);
    expect(HouseTabs.indexOf(null), 0);
    expect(HouseTabs.indexOf('nope'), 0);
    expect(HouseTabs.keyAt(1), 'notes');
  });
}
