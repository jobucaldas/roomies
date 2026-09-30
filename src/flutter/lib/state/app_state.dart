import 'package:flutter/foundation.dart';

import '../api/api_client.dart';
import '../api/api_error.dart';
import '../models/models.dart';

class AppState extends ChangeNotifier {
  AppState(this.api);

  final ApiClient api;

  User? user;
  bool sessionReady = false;

  Future<void> restoreSession() async {
    await api.clearLegacyWebSession();
    await api.loadPersistedToken();
    if (api.hasSessionHint) {
      try {
        user = await api.me();
        api.adoptCookieSession();
      } on ApiError catch (error) {
        if (error.status == 401) {
          await api.logout();
        } else {
          api.invalidateSession();
        }
        user = null;
      } catch (_) {
        api.invalidateSession();
        user = null;
      }
    } else if (api.hasSavedToken) {
      try {
        user = await api.me();
      } on ApiError catch (error) {
        if (error.status == 401) {
          await api.logout();
        } else {
          api.invalidateSession();
        }
        user = null;
      } catch (_) {
        api.invalidateSession();
        user = null;
      }
    }
    sessionReady = true;
    notifyListeners();
  }

  Future<void> setUser(User? value) async {
    user = value;
    notifyListeners();
  }

  Future<void> logout() async {
    await api.logout();
    user = null;
    notifyListeners();
  }
}
