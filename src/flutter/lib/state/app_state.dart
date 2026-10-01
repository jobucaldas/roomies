import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';

import '../api/api_client.dart';
import '../api/api_error.dart';
import '../l10n/strings.dart';
import '../models/models.dart';
import '../services/preferences_storage.dart';
import '../theme/roomies_theme.dart';

class AppState extends ChangeNotifier {
  AppState(
    this.api, {
    PreferencesStorage? preferences,
    String? deviceLocale,
  })  : preferences = preferences ?? PreferencesStorage(),
        _deviceLocale = RoomiesStrings.resolveDeviceLocale(deviceLocale);

  final ApiClient api;
  final PreferencesStorage preferences;

  User? user;
  bool sessionReady = false;

  /// Resolved locale code used by the UI (`en` or `pt`).
  String localeCode = 'en';

  /// Explicit Settings override; `null` means follow the device.
  String? localeOverride;

  /// Theme override: `null` (system), `light`, or `dark`.
  String? themeOverride;

  /// Brand accent family (`mint` / `plum`).
  BrandAccent brandAccent = BrandAccent.mint;

  /// Preferred house for later sessions.
  String? defaultHouseId;

  /// Cached membership list for the sidebar / Settings switcher.
  List<House> houses = const [];

  final String _deviceLocale;

  /// True when this process restored an existing session (not a fresh login).
  bool restoredExistingSession = false;

  RoomiesStrings get strings => RoomiesStrings(localeCode);

  ThemeMode get themeMode {
    switch (themeOverride) {
      case 'light':
        return ThemeMode.light;
      case 'dark':
        return ThemeMode.dark;
      default:
        return ThemeMode.system;
    }
  }

  Future<void> loadPreferences() async {
    localeOverride = await preferences.loadLocaleOverride();
    themeOverride = await preferences.loadThemeOverride();
    brandAccent = BrandAccentX.fromId(await preferences.loadBrandAccent());
    defaultHouseId = await preferences.loadDefaultHouseId();
    _applyLocale();
    notifyListeners();
  }

  void _applyLocale() {
    final override = localeOverride;
    if (override == 'pt' || override == 'en') {
      localeCode = override!;
    } else {
      localeCode = _deviceLocale;
    }
  }

  Future<void> setLocaleOverride(String? code) async {
    if (code != null && code != 'en' && code != 'pt') return;
    localeOverride = code;
    await preferences.saveLocaleOverride(code);
    _applyLocale();
    notifyListeners();
  }

  Future<void> setThemeOverride(String? mode) async {
    if (mode != null && mode != 'light' && mode != 'dark') return;
    themeOverride = mode;
    await preferences.saveThemeOverride(mode);
    notifyListeners();
  }

  Future<void> setBrandAccent(BrandAccent accent) async {
    brandAccent = accent;
    await preferences.saveBrandAccent(accent.id);
    notifyListeners();
  }

  void toggleLanguageQuick() {
    final next = localeCode == 'pt' ? 'en' : 'pt';
    // Fire-and-forget; callers that need await use [setLocaleOverride].
    setLocaleOverride(next);
  }

  Future<void> setDefaultHouseId(String? houseId) async {
    defaultHouseId = houseId;
    await preferences.saveDefaultHouseId(houseId);
    notifyListeners();
  }

  Future<void> refreshHouses() async {
    if (!api.isAuthenticated) {
      houses = const [];
      notifyListeners();
      return;
    }
    try {
      houses = await api.getHouses();
      if (defaultHouseId != null &&
          houses.every((house) => house.id != defaultHouseId)) {
        await setDefaultHouseId(null);
      }
      notifyListeners();
    } catch (_) {
      // Keep the last known list; screens surface their own errors.
    }
  }

  Future<House> createHouseAndSetDefault(String name) async {
    final house = await api.createHouse(name);
    await setDefaultHouseId(house.id);
    await refreshHouses();
    return house;
  }

  /// Destination after a restored session (not a fresh AuthKit/password login).
  /// Returns `null` for fresh logins so the caller can force `/dashboard`.
  String? restoredHomePath() {
    if (!restoredExistingSession) return null;
    final id = defaultHouseId;
    if (id == null || id.isEmpty) return '/dashboard';
    if (houses.isNotEmpty && houses.every((house) => house.id != id)) {
      return '/dashboard';
    }
    return '/house/$id';
  }

  Future<void> restoreSession() async {
    await loadPreferences();
    await api.clearLegacyWebSession();
    await api.loadPersistedToken();
    if (api.hasSessionHint) {
      try {
        user = await api.me();
        api.adoptCookieSession();
        restoredExistingSession = true;
        await refreshHouses();
      } on ApiError catch (error) {
        if (error.status == 401) {
          await api.logout();
        } else {
          api.invalidateSession();
        }
        user = null;
        restoredExistingSession = false;
      } catch (_) {
        api.invalidateSession();
        user = null;
        restoredExistingSession = false;
      }
    } else if (api.hasSavedToken) {
      try {
        user = await api.me();
        restoredExistingSession = true;
        await refreshHouses();
      } on ApiError catch (error) {
        if (error.status == 401) {
          await api.logout();
        } else {
          api.invalidateSession();
        }
        user = null;
        restoredExistingSession = false;
      } catch (_) {
        api.invalidateSession();
        user = null;
        restoredExistingSession = false;
      }
    }
    sessionReady = true;
    notifyListeners();
  }

  Future<void> setUser(User? value) async {
    user = value;
    restoredExistingSession = false;
    if (value != null) {
      await refreshHouses();
    } else {
      houses = const [];
    }
    notifyListeners();
  }

  Future<void> logout() async {
    await api.logout();
    user = null;
    houses = const [];
    restoredExistingSession = false;
    notifyListeners();
  }
}

/// Platform locale tag for bootstrap (e.g. `pt_BR`, `en_US`).
String platformLocaleTag() {
  return PlatformDispatcher.instance.locale.toLanguageTag();
}
