import 'package:shared_preferences/shared_preferences.dart';

const localeOverrideKey = 'roomies.locale.override';
const defaultHouseKey = 'roomies.default.house';
const themeOverrideKey = 'roomies.theme.override';
const brandAccentKey = 'roomies.brand.accent';

/// Client preferences that survive sessions (locale, theme, default house).
class PreferencesStorage {
  Future<String?> loadLocaleOverride() async {
    final prefs = await SharedPreferences.getInstance();
    final value = prefs.getString(localeOverrideKey);
    if (value == null || value.isEmpty) return null;
    return value;
  }

  Future<void> saveLocaleOverride(String? code) async {
    final prefs = await SharedPreferences.getInstance();
    if (code == null || code.isEmpty) {
      await prefs.remove(localeOverrideKey);
    } else {
      await prefs.setString(localeOverrideKey, code);
    }
  }

  /// `null` = follow device; otherwise `light` or `dark`.
  Future<String?> loadThemeOverride() async {
    final prefs = await SharedPreferences.getInstance();
    final value = prefs.getString(themeOverrideKey);
    if (value == null || value.isEmpty) return null;
    if (value == 'light' || value == 'dark') return value;
    return null;
  }

  Future<void> saveThemeOverride(String? mode) async {
    final prefs = await SharedPreferences.getInstance();
    if (mode == null || mode.isEmpty) {
      await prefs.remove(themeOverrideKey);
    } else {
      await prefs.setString(themeOverrideKey, mode);
    }
  }

  /// Brand accent id: `mint` (default) or `plum`.
  Future<String> loadBrandAccent() async {
    final prefs = await SharedPreferences.getInstance();
    final value = prefs.getString(brandAccentKey);
    if (value == 'plum' || value == 'mint') return value!;
    return 'mint';
  }

  Future<void> saveBrandAccent(String id) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(brandAccentKey, id);
  }

  Future<String?> loadDefaultHouseId() async {
    final prefs = await SharedPreferences.getInstance();
    final value = prefs.getString(defaultHouseKey);
    if (value == null || value.isEmpty) return null;
    return value;
  }

  Future<void> saveDefaultHouseId(String? houseId) async {
    final prefs = await SharedPreferences.getInstance();
    if (houseId == null || houseId.isEmpty) {
      await prefs.remove(defaultHouseKey);
    } else {
      await prefs.setString(defaultHouseKey, houseId);
    }
  }
}
