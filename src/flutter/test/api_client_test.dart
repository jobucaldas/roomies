import 'package:flutter_test/flutter_test.dart';
import 'package:roomies/api/api_client.dart';

void main() {
  test('resolveApiBaseUrl uses origin for relative web paths', () {
    expect(
      resolveApiBaseUrl(
        configured: '/api',
        webOrigin: 'https://app.example',
        isWeb: true,
      ),
      'https://app.example/api',
    );
  });

  test('resolveApiBaseUrl defaults to relative api on web', () {
    expect(resolveApiBaseUrl(isWeb: true), '/api');
  });
}
