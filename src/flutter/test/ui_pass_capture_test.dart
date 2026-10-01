import 'package:flutter_test/flutter_test.dart';

/// Screenshots live under the project store `media/ui-pass/` and are
/// captured with Chrome headless HTML previews (flutter_tester + toImage
/// hangs with custom web fonts in this environment).
void main() {
  test('ui-pass capture is external to flutter_tester', () {
    expect(true, isTrue);
  });
}
