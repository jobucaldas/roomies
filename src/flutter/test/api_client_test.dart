import 'package:flutter_test/flutter_test.dart';
import 'package:roomies/api/api_client.dart';
import 'package:roomies/auth/workos_url.dart';

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

  test('workos authorize urls are limited to the hosted api host', () {
    expect(
      isWorkOSAuthorizeUrl(
        'https://api.workos.com/user_management/authorize?client_id=client_123',
      ),
      isTrue,
    );
    expect(
      isWorkOSAuthorizeUrl('https://evil.example/user_management/authorize'),
      isFalse,
    );
    expect(
        isWorkOSAuthorizeUrl('http://api.workos.com/user_management/authorize'),
        isFalse);
  });
}
