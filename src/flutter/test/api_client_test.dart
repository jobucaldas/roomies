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

  test('resolveApiBaseUrl ignores baked localhost on public web origin', () {
    expect(
      resolveApiBaseUrl(
        configured: 'http://localhost:8080/api',
        webOrigin: 'https://roomies-dev.example',
        isWeb: true,
      ),
      'https://roomies-dev.example/api',
    );
    expect(
      resolveApiBaseUrl(
        configured: 'http://127.0.0.1:8080/api',
        webOrigin: 'https://roomies-dev.example',
        isWeb: true,
      ),
      'https://roomies-dev.example/api',
    );
  });

  test('resolveApiBaseUrl keeps localhost when page is also loopback', () {
    expect(
      resolveApiBaseUrl(
        configured: 'http://localhost:8080/api',
        webOrigin: 'http://localhost:58000',
        isWeb: true,
      ),
      'http://localhost:8080/api',
    );
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

  test('workos logout urls are limited to the hosted logout endpoint', () {
    expect(
      isWorkOSLogoutUrl(
        'https://api.workos.com/user_management/sessions/logout?session_id=s_1',
      ),
      isTrue,
    );
    expect(
      isWorkOSLogoutUrl('https://evil.example/user_management/sessions/logout'),
      isFalse,
    );
    expect(
      isWorkOSLogoutUrl(
        'https://api.workos.com/user_management/authorize?client_id=c',
      ),
      isFalse,
    );
  });
}
