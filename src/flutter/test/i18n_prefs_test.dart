import 'package:flutter_test/flutter_test.dart';
import 'package:roomies/l10n/strings.dart';
import 'package:roomies/services/preferences_storage.dart';
import 'package:roomies/state/app_state.dart';
import 'package:roomies/api/api_client.dart';
import 'package:shared_preferences/shared_preferences.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  setUp(() {
    SharedPreferences.setMockInitialValues({});
  });

  test('device locale defaults to Portuguese for pt tags', () {
    expect(RoomiesStrings.resolveDeviceLocale('pt-BR'), 'pt');
    expect(RoomiesStrings.resolveDeviceLocale('pt_BR'), 'pt');
    expect(RoomiesStrings.resolveDeviceLocale('en-US'), 'en');
    expect(RoomiesStrings.resolveDeviceLocale(null), 'en');
  });

  test('Portuguese strings cover AuthKit CTA', () {
    final pt = RoomiesStrings('pt');
    expect(pt.continueWorkOS, 'Continuar com WorkOS');
    expect(pt.dashboard, 'Dashboard');
    expect(pt.settings, 'Configurações');
  });

  test('locale override beats device locale', () async {
    final app = AppState(
      ApiClient(baseUrl: 'http://example/api'),
      preferences: PreferencesStorage(),
      deviceLocale: 'en-US',
    );
    await app.loadPreferences();
    expect(app.localeCode, 'en');
    await app.setLocaleOverride('pt');
    expect(app.localeCode, 'pt');
    expect(app.strings.continueWorkOS, 'Continuar com WorkOS');
    await app.setLocaleOverride(null);
    expect(app.localeCode, 'en');
  });

  test('default house preference persists', () async {
    final prefs = PreferencesStorage();
    await prefs.saveDefaultHouseId('house-1');
    expect(await prefs.loadDefaultHouseId(), 'house-1');
    await prefs.saveDefaultHouseId(null);
    expect(await prefs.loadDefaultHouseId(), isNull);
  });

  test('restoredHomePath prefers default house', () {
    final app = AppState(
      ApiClient(baseUrl: 'http://example/api'),
      deviceLocale: 'en',
    );
    app.restoredExistingSession = true;
    app.defaultHouseId = 'h1';
    app.houses = [];
    // Without membership list yet, still route to the stored default.
    expect(app.restoredHomePath(), '/house/h1');
  });
}
