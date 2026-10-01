import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:roomies/api/api_client.dart';
import 'package:roomies/l10n/strings.dart';
import 'package:roomies/services/preferences_storage.dart';
import 'package:roomies/state/app_state.dart';
import 'package:roomies/theme/roomies_theme.dart';
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

  test('Portuguese strings cover hosted AuthKit CTA', () {
    final pt = RoomiesStrings('pt');
    expect(pt.signIn, 'Entrar');
    expect(pt.createAccount, 'Criar conta');
    expect(pt.settings, 'Configurações');
    expect(pt.dashboard, 'Painel');
    expect(pt.appearance, 'Aparência');
    expect(pt.themeDark, 'Escuro');
    expect(pt.houseTabs.contains('Chat'), isFalse);
    expect(pt.recentEvents, 'Eventos recentes');
    expect(pt.monthMoney, 'Gastos do mês');
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
    expect(app.strings.signIn, 'Entrar');
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
    expect(app.restoredHomePath(), '/house/h1');
  });

  test('theme override persists and maps to ThemeMode', () async {
    final app = AppState(
      ApiClient(baseUrl: 'http://example/api'),
      preferences: PreferencesStorage(),
      deviceLocale: 'en-US',
    );
    await app.loadPreferences();
    expect(app.themeMode, ThemeMode.system);
    await app.setThemeOverride('dark');
    expect(app.themeOverride, 'dark');
    expect(app.themeMode, ThemeMode.dark);
    await app.setThemeOverride('light');
    expect(app.themeMode, ThemeMode.light);
    await app.setThemeOverride(null);
    expect(app.themeMode, ThemeMode.system);
  });

  test('brand accent persists mint and plum', () async {
    final app = AppState(
      ApiClient(baseUrl: 'http://example/api'),
      preferences: PreferencesStorage(),
      deviceLocale: 'en-US',
    );
    await app.loadPreferences();
    expect(app.brandAccent, BrandAccent.mint);
    await app.setBrandAccent(BrandAccent.plum);
    expect(app.brandAccent, BrandAccent.plum);
    expect(await PreferencesStorage().loadBrandAccent(), 'plum');
    await app.setBrandAccent(BrandAccent.mint);
    expect(app.brandAccent, BrandAccent.mint);
  });
}
