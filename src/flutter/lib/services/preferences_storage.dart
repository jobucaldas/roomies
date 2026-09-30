import 'package:shared_preferences/shared_preferences.dart';

const localeOverrideKey = 'roomies.locale.override';
const defaultHouseKey = 'roomies.default.house';

/// Client preferences that survive sessions (locale override + default house).
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
