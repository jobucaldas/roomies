import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:roomies/services/session_storage.dart';
import 'package:shared_preferences/shared_preferences.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  test('session token lives in secure storage, not preferences', () async {
    FlutterSecureStorage.setMockInitialValues({});
    SharedPreferences.setMockInitialValues({});
    final storage = SessionStorage();
    await storage.saveToken('jwt-1');
    final prefs = await SharedPreferences.getInstance();
    expect(prefs.getString(sessionTokenKey), isNull);
    expect(await const FlutterSecureStorage().read(key: sessionTokenKey), 'jwt-1');
    expect(await storage.loadToken(), 'jwt-1');
    await storage.clearToken();
    expect(await storage.loadToken(), isNull);
  });

  test('a plaintext token from an older build moves to secure storage', () async {
    FlutterSecureStorage.setMockInitialValues({});
    SharedPreferences.setMockInitialValues({sessionTokenKey: 'legacy-jwt'});
    final storage = SessionStorage();
    expect(await storage.loadToken(), 'legacy-jwt');
    final prefs = await SharedPreferences.getInstance();
    expect(prefs.getString(sessionTokenKey), isNull);
    expect(await const FlutterSecureStorage().read(key: sessionTokenKey), 'legacy-jwt');
  });

  test('a plaintext token is removed even when the keystore fails', () async {
    FlutterSecureStorage.setMockInitialValues({});
    SharedPreferences.setMockInitialValues({sessionTokenKey: 'legacy-jwt'});
    final storage = SessionStorage(secureStorage: _BrokenSecureStorage());
    expect(await storage.loadToken(), isNull);
    final prefs = await SharedPreferences.getInstance();
    expect(prefs.getString(sessionTokenKey), isNull);
    await prefs.setString(sessionTokenKey, 'another-legacy-jwt');
    await storage.saveToken('fresh-jwt');
    expect(prefs.getString(sessionTokenKey), isNull);
  });
}

class _BrokenSecureStorage extends FlutterSecureStorage {
  @override
  Future<String?> read({
    required String key,
    IOSOptions? iOptions,
    AndroidOptions? aOptions,
    LinuxOptions? lOptions,
    WebOptions? webOptions,
    MacOsOptions? mOptions,
    WindowsOptions? wOptions,
  }) =>
      throw Exception('no keystore');

  @override
  Future<void> write({
    required String key,
    required String? value,
    IOSOptions? iOptions,
    AndroidOptions? aOptions,
    LinuxOptions? lOptions,
    WebOptions? webOptions,
    MacOsOptions? mOptions,
    WindowsOptions? wOptions,
  }) =>
      throw Exception('no keystore');
}
