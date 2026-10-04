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
}
