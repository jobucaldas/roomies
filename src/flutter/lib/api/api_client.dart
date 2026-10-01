import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;

import '../models/models.dart';
import '../services/session_hint_stub.dart'
    if (dart.library.js_interop) '../services/session_hint_web.dart'
    as session_hint;
import '../services/session_storage.dart';
import 'api_error.dart';
import 'http_client_factory_stub.dart'
    if (dart.library.js_interop) 'http_client_factory_web.dart';

bool _isLoopbackHost(String? host) {
  if (host == null || host.isEmpty) return false;
  final normalized = host.toLowerCase();
  return normalized == 'localhost' ||
      normalized == '127.0.0.1' ||
      normalized == '::1' ||
      normalized == '[::1]';
}

/// Resolves the API base for the current client.
///
/// Web builds should use relative `/api` (same-origin via Caddy). If a release
/// image was accidentally baked with a loopback `ROOMIES_API_URL` but is served
/// from a public origin (Tailscale, Ingress), ignore the loopback URL and use
/// `{webOrigin}/api` so phones on the tailnet never call the device's localhost.
String resolveApiBaseUrl({
  String? configured,
  String? webOrigin,
  bool isWeb = kIsWeb,
}) {
  const fallback = '/api';
  var value =
      (configured == null || configured.isEmpty) ? fallback : configured;
  final origin = (webOrigin == null || webOrigin.isEmpty)
      ? null
      : webOrigin.replaceAll(RegExp(r'/+$'), '');

  if (isWeb && origin != null) {
    final originHost = Uri.tryParse(origin)?.host;
    final configuredUri = Uri.tryParse(value);
    if (!_isLoopbackHost(originHost) &&
        configuredUri != null &&
        configuredUri.hasScheme &&
        _isLoopbackHost(configuredUri.host)) {
      value = fallback;
    }
    if (value.startsWith('/')) {
      return '$origin$value'.replaceAll(RegExp(r'/+$'), '');
    }
  } else if (isWeb && value.startsWith('/')) {
    return value.replaceAll(RegExp(r'/+$'), '');
  }
  return value.replaceAll(RegExp(r'/+$'), '');
}

class ApiClient {
  ApiClient({
    SessionStorage? storage,
    http.Client? httpClient,
    String? baseUrl,
    String? webOrigin,
  })  : _storage = storage ?? SessionStorage(),
        _http = httpClient ?? createPlatformClient(),
        baseUrl = resolveApiBaseUrl(
          configured: baseUrl ??
              const String.fromEnvironment('ROOMIES_API_URL', defaultValue: ''),
          webOrigin: webOrigin,
        );

  final SessionStorage _storage;
  final http.Client _http;
  final String baseUrl;

  String? _token;
  bool _cookieSession = false;
  bool _valid = false;
  int _generation = 0;

  bool get hasSavedToken => _token != null && _token!.isNotEmpty;

  bool get hasSessionHint => session_hint.hasSessionHint();

  bool get isAuthenticated => _valid && (_cookieSession || hasSavedToken);

  /// Web sessions live in an HttpOnly cookie. Drop any JWT left in
  /// localStorage by older builds so page script cannot read it.
  Future<void> clearLegacyWebSession() async {
    if (!kIsWeb) return;
    await _storage.clearToken();
  }

  Future<void> loadPersistedToken() async {
    if (kIsWeb) {
      _token = null;
      _valid = false;
      return;
    }
    _token = await _storage.loadToken();
    _valid = _token != null;
  }

  void adoptCookieSession() {
    _generation++;
    _token = null;
    _cookieSession = true;
    _valid = true;
  }

  Future<void> _setToken(String token) async {
    if (kIsWeb || token.isEmpty) {
      adoptCookieSession();
      return;
    }
    await _storage.saveToken(token);
    _generation++;
    _cookieSession = false;
    _token = token;
    _valid = true;
  }

  void invalidateSession() {
    _valid = false;
  }

  Future<void> logout() async {
    final gen = _generation;
    try {
      await _empty(
        () =>
            _http.post(Uri.parse('$baseUrl/auth/logout'), headers: _headers()),
        generation: gen,
      );
    } catch (_) {
      // Clearing the local session still logs the tab out if the network call fails.
    }
    _generation++;
    _token = null;
    _cookieSession = false;
    _valid = false;
    await _storage.clearToken();
  }

  void _handleUnauthorized(int? generation) {
    if (generation == null) return;
    if (generation != _generation) return;
    _generation++;
    _token = null;
    _cookieSession = false;
    _valid = false;
    _storage.clearToken();
  }

  Map<String, String> _headers() {
    final headers = <String, String>{'Content-Type': 'application/json'};
    if (kIsWeb) {
      headers['X-Roomies-Client'] = 'web';
    }
    if (_valid && !_cookieSession && _token != null) {
      headers['Authorization'] = 'Bearer $_token';
    }
    return headers;
  }

  Future<T> _send<T>(
    Future<http.Response> Function() request,
    T Function(Map<String, dynamic> json) decode, {
    int? generation,
  }) async {
    final response = await request().catchError((Object e) {
      throw ApiError.transport(e.toString());
    });
    if (response.statusCode >= 200 && response.statusCode < 300) {
      if (response.body.isEmpty) {
        throw ApiError.decode('empty body');
      }
      final decoded = jsonDecode(response.body);
      if (decoded is Map<String, dynamic>) {
        return decode(decoded);
      }
      throw ApiError.decode('expected object');
    }
    if (response.statusCode == 401) {
      _handleUnauthorized(generation);
    }
    var message = response.statusCode.toString();
    try {
      final body = jsonDecode(response.body);
      if (body is Map && body['error'] is String) {
        message = body['error'] as String;
      }
    } catch (_) {}
    throw ApiError.http(response.statusCode, message);
  }

  Future<void> _empty(Future<http.Response> Function() request,
      {int? generation}) async {
    final response = await request().catchError((Object e) {
      throw ApiError.transport(e.toString());
    });
    if (response.statusCode >= 200 && response.statusCode < 300) {
      return;
    }
    if (response.statusCode == 401) {
      _handleUnauthorized(generation);
    }
    throw ApiError.http(response.statusCode, response.body);
  }

  Future<AuthConfig> getAuthConfig() async {
    final gen = _generation;
    return _send(
      () => _http.get(Uri.parse('$baseUrl/auth/config'), headers: _headers()),
      AuthConfig.fromJson,
      generation: gen,
    );
  }

  Future<String> workosAuthorizeUrl({String screenHint = 'sign-in'}) async {
    final gen = _generation;
    final response = await _http
        .get(
          Uri.parse('$baseUrl/auth/workos/authorize').replace(
            queryParameters: {'screen_hint': screenHint},
          ),
          headers: _headers(),
        )
        .catchError((Object e) => throw ApiError.transport(e.toString()));
    if (response.statusCode >= 200 && response.statusCode < 300) {
      final body = jsonDecode(response.body) as Map<String, dynamic>;
      final url = body['url'] as String?;
      if (url == null || url.isEmpty) {
        throw ApiError.http(response.statusCode, 'missing authorization url');
      }
      return url;
    }
    if (response.statusCode == 401) {
      _handleUnauthorized(gen);
    }
    throw ApiError.http(response.statusCode, response.body);
  }

  Future<AuthResponse> completeWorkOSCallback(String code, String state) async {
    final gen = _generation;
    final result = await _send(
      () => _http.post(
        Uri.parse('$baseUrl/auth/workos/callback'),
        headers: _headers(),
        body: jsonEncode({'code': code, 'state': state}),
      ),
      AuthResponse.fromJson,
      generation: gen,
    );
    await _setToken(result.token);
    return result;
  }

  Future<AuthResponse> register(
      String name, String email, String password) async {
    final gen = _generation;
    final result = await _send(
      () => _http.post(
        Uri.parse('$baseUrl/auth/register'),
        headers: _headers(),
        body: jsonEncode({'name': name, 'email': email, 'password': password}),
      ),
      AuthResponse.fromJson,
      generation: gen,
    );
    await _setToken(result.token);
    return result;
  }

  Future<AuthResponse> login(String email, String password) async {
    final gen = _generation;
    final result = await _send(
      () => _http.post(
        Uri.parse('$baseUrl/auth/login'),
        headers: _headers(),
        body: jsonEncode({'email': email, 'password': password}),
      ),
      AuthResponse.fromJson,
      generation: gen,
    );
    await _setToken(result.token);
    return result;
  }

  Future<User> me() async {
    final gen = _generation;
    return _send(
      () => _http.get(Uri.parse('$baseUrl/auth/me'), headers: _headers()),
      User.fromJson,
      generation: gen,
    );
  }

  Future<List<House>> getHouses() async {
    final gen = _generation;
    final response = await _http
        .get(Uri.parse('$baseUrl/houses'), headers: _headers())
        .catchError((Object e) => throw ApiError.transport(e.toString()));
    if (response.statusCode == 401) {
      _handleUnauthorized(gen);
      throw ApiError.http(401, 'unauthorized');
    }
    if (response.statusCode >= 200 && response.statusCode < 300) {
      final list = jsonDecode(response.body) as List<dynamic>;
      return list
          .map((e) => House.fromJson(e as Map<String, dynamic>))
          .toList();
    }
    throw ApiError.http(response.statusCode, response.body);
  }

  Future<House> createHouse(String name) async {
    final gen = _generation;
    return _send(
      () => _http.post(
        Uri.parse('$baseUrl/houses'),
        headers: _headers(),
        body: jsonEncode({'name': name}),
      ),
      House.fromJson,
      generation: gen,
    );
  }

  Future<House> updateHouse(String id, String name) async {
    final gen = _generation;
    return _send(
      () => _http.put(
        Uri.parse('$baseUrl/houses/$id'),
        headers: _headers(),
        body: jsonEncode({'name': name}),
      ),
      House.fromJson,
      generation: gen,
    );
  }

  Future<House> getHouse(String id) async {
    final gen = _generation;
    return _send(
      () => _http.get(Uri.parse('$baseUrl/houses/$id'), headers: _headers()),
      House.fromJson,
      generation: gen,
    );
  }

  Future<List<HouseMember>> getMembers(String id) async {
    final gen = _generation;
    final response = await _http
        .get(Uri.parse('$baseUrl/houses/$id/members'), headers: _headers())
        .catchError((Object e) => throw ApiError.transport(e.toString()));
    if (response.statusCode == 401) {
      _handleUnauthorized(gen);
      throw ApiError.http(401, 'unauthorized');
    }
    if (response.statusCode >= 200 && response.statusCode < 300) {
      final list = jsonDecode(response.body) as List<dynamic>;
      return list
          .map((e) => HouseMember.fromJson(e as Map<String, dynamic>))
          .toList();
    }
    throw ApiError.http(response.statusCode, response.body);
  }

  Future<HouseMember> addMember(String id, String userId, String role) async {
    final gen = _generation;
    return _send(
      () => _http.post(
        Uri.parse('$baseUrl/houses/$id/members'),
        headers: _headers(),
        body: jsonEncode({'user_id': userId, 'role': role}),
      ),
      HouseMember.fromJson,
      generation: gen,
    );
  }

  Future<MessageResponse> updateMemberRole(
      String id, String userId, String role) async {
    final gen = _generation;
    return _send(
      () => _http.put(
        Uri.parse('$baseUrl/houses/$id/members/$userId'),
        headers: _headers(),
        body: jsonEncode({'role': role}),
      ),
      MessageResponse.fromJson,
      generation: gen,
    );
  }

  Future<void> removeMember(String id, String userId) async {
    final gen = _generation;
    await _empty(
      () => _http.delete(
        Uri.parse('$baseUrl/houses/$id/members/$userId'),
        headers: _headers(),
      ),
      generation: gen,
    );
  }

  Future<HouseInvitation> createInvitation(
      String houseId, String email, String role) async {
    final gen = _generation;
    return _send(
      () => _http.post(
        Uri.parse('$baseUrl/houses/$houseId/invites'),
        headers: _headers(),
        body: jsonEncode({'email': email, 'role': role}),
      ),
      HouseInvitation.fromJson,
      generation: gen,
    );
  }

  Future<List<HouseInvitation>> listInvitations(String houseId) async {
    final gen = _generation;
    final response = await _http
        .get(Uri.parse('$baseUrl/houses/$houseId/invites'), headers: _headers())
        .catchError((Object e) => throw ApiError.transport(e.toString()));
    if (response.statusCode == 401) {
      _handleUnauthorized(gen);
      throw ApiError.http(401, 'unauthorized');
    }
    if (response.statusCode >= 200 && response.statusCode < 300) {
      final list = jsonDecode(response.body) as List<dynamic>;
      return list
          .map((e) => HouseInvitation.fromJson(e as Map<String, dynamic>))
          .toList();
    }
    throw ApiError.http(response.statusCode, response.body);
  }

  Future<HouseInvitation> revokeInvitation(
      String houseId, String invitationId) async {
    final gen = _generation;
    return _send(
      () => _http.delete(
        Uri.parse('$baseUrl/houses/$houseId/invites/$invitationId'),
        headers: _headers(),
      ),
      HouseInvitation.fromJson,
      generation: gen,
    );
  }

  Future<InvitationAcceptanceResponse> acceptInvitation(String token) async {
    final gen = _generation;
    return _send(
      () => _http.post(
        Uri.parse('$baseUrl/invitations/accept'),
        headers: _headers(),
        body: jsonEncode({'token': token}),
      ),
      InvitationAcceptanceResponse.fromJson,
      generation: gen,
    );
  }

  Future<List<Expense>> getExpenses(String id) async {
    final gen = _generation;
    final response = await _http
        .get(Uri.parse('$baseUrl/houses/$id/expenses'), headers: _headers())
        .catchError((Object e) => throw ApiError.transport(e.toString()));
    if (response.statusCode == 401) {
      _handleUnauthorized(gen);
      throw ApiError.http(401, 'unauthorized');
    }
    if (response.statusCode >= 200 && response.statusCode < 300) {
      final list = jsonDecode(response.body) as List<dynamic>;
      return list
          .map((e) => Expense.fromJson(e as Map<String, dynamic>))
          .toList();
    }
    throw ApiError.http(response.statusCode, response.body);
  }

  Future<ExpenseDetail> getExpense(String houseId, String expenseId) async {
    final gen = _generation;
    return _send(
      () => _http.get(
        Uri.parse('$baseUrl/houses/$houseId/expenses/$expenseId'),
        headers: _headers(),
      ),
      ExpenseDetail.fromJson,
      generation: gen,
    );
  }

  Future<Expense> createExpense(String id, Map<String, dynamic> body) async {
    final gen = _generation;
    return _send(
      () => _http.post(
        Uri.parse('$baseUrl/houses/$id/expenses'),
        headers: _headers(),
        body: jsonEncode(body),
      ),
      Expense.fromJson,
      generation: gen,
    );
  }

  Future<Expense> updateExpense(
      String houseId, String expenseId, Map<String, dynamic> body) async {
    final gen = _generation;
    return _send(
      () => _http.put(
        Uri.parse('$baseUrl/houses/$houseId/expenses/$expenseId'),
        headers: _headers(),
        body: jsonEncode(body),
      ),
      Expense.fromJson,
      generation: gen,
    );
  }

  Future<void> deleteExpense(String houseId, String expenseId) async {
    final gen = _generation;
    await _empty(
      () => _http.delete(
        Uri.parse('$baseUrl/houses/$houseId/expenses/$expenseId'),
        headers: _headers(),
      ),
      generation: gen,
    );
  }

  Future<void> setExpenseVisibility(String houseId, String expenseId,
      String visibility, List<String> visibleTo) async {
    final gen = _generation;
    await _empty(
      () => _http.post(
        Uri.parse('$baseUrl/houses/$houseId/expenses/$expenseId/visibility'),
        headers: _headers(),
        body: jsonEncode({'visibility': visibility, 'visible_to': visibleTo}),
      ),
      generation: gen,
    );
  }

  Future<List<Note>> getNotes(String id) async {
    final gen = _generation;
    final response = await _http
        .get(Uri.parse('$baseUrl/houses/$id/notes'), headers: _headers())
        .catchError((Object e) => throw ApiError.transport(e.toString()));
    if (response.statusCode == 401) {
      _handleUnauthorized(gen);
      throw ApiError.http(401, 'unauthorized');
    }
    if (response.statusCode >= 200 && response.statusCode < 300) {
      final list = jsonDecode(response.body) as List<dynamic>;
      return list.map((e) => Note.fromJson(e as Map<String, dynamic>)).toList();
    }
    throw ApiError.http(response.statusCode, response.body);
  }

  Future<Note> createNote(String houseId, String title, String content) async {
    final gen = _generation;
    return _send(
      () => _http.post(
        Uri.parse('$baseUrl/houses/$houseId/notes'),
        headers: _headers(),
        body: jsonEncode({'title': title, 'content': content}),
      ),
      Note.fromJson,
      generation: gen,
    );
  }

  Future<Note> updateNote(
      String houseId, String noteId, String title, String content) async {
    final gen = _generation;
    return _send(
      () => _http.put(
        Uri.parse('$baseUrl/houses/$houseId/notes/$noteId'),
        headers: _headers(),
        body: jsonEncode({'title': title, 'content': content}),
      ),
      Note.fromJson,
      generation: gen,
    );
  }

  Future<void> deleteNote(String houseId, String noteId) async {
    final gen = _generation;
    await _empty(
      () => _http.delete(
        Uri.parse('$baseUrl/houses/$houseId/notes/$noteId'),
        headers: _headers(),
      ),
      generation: gen,
    );
  }

  Future<BalanceResponse> getBalances(String id) async {
    final gen = _generation;
    return _send(
      () => _http.get(
        Uri.parse('$baseUrl/houses/$id/balances'),
        headers: _headers(),
      ),
      BalanceResponse.fromJson,
      generation: gen,
    );
  }

  Future<NotificationPreferences> getNotificationPreferences(String id) async {
    final gen = _generation;
    return _send(
      () => _http.get(
        Uri.parse('$baseUrl/houses/$id/notification-preferences'),
        headers: _headers(),
      ),
      NotificationPreferences.fromJson,
      generation: gen,
    );
  }

  Future<NotificationPreferences> putNotificationPreferences(
      String id, NotificationPreferences prefs) async {
    final gen = _generation;
    return _send(
      () => _http.put(
        Uri.parse('$baseUrl/houses/$id/notification-preferences'),
        headers: _headers(),
        body: jsonEncode(prefs.toJson()),
      ),
      NotificationPreferences.fromJson,
      generation: gen,
    );
  }

  Future<List<NotificationSubscription>> getNotificationSubscriptions(
      String id) async {
    final gen = _generation;
    final response = await _http
        .get(Uri.parse('$baseUrl/houses/$id/notification-subscriptions'),
            headers: _headers())
        .catchError((Object e) => throw ApiError.transport(e.toString()));
    if (response.statusCode == 401) {
      _handleUnauthorized(gen);
      throw ApiError.http(401, 'unauthorized');
    }
    if (response.statusCode >= 200 && response.statusCode < 300) {
      final list = jsonDecode(response.body) as List<dynamic>;
      return list
          .map((e) =>
              NotificationSubscription.fromJson(e as Map<String, dynamic>))
          .toList();
    }
    throw ApiError.http(response.statusCode, response.body);
  }

  Future<NotificationSubscription> createNotificationSubscription(
      String id, Map<String, dynamic> body) async {
    final gen = _generation;
    return _send(
      () => _http.post(
        Uri.parse('$baseUrl/houses/$id/notification-subscriptions'),
        headers: _headers(),
        body: jsonEncode(body),
      ),
      NotificationSubscription.fromJson,
      generation: gen,
    );
  }

  Future<void> deleteNotificationSubscription(
      String houseId, String subscriptionId) async {
    final gen = _generation;
    await _empty(
      () => _http.delete(
        Uri.parse(
            '$baseUrl/houses/$houseId/notification-subscriptions/$subscriptionId'),
        headers: _headers(),
      ),
      generation: gen,
    );
  }

  Future<List<ScheduledHouseEvent>> getScheduledEvents(String id) async {
    final gen = _generation;
    final response = await _http
        .get(Uri.parse('$baseUrl/houses/$id/scheduled-events'),
            headers: _headers())
        .catchError((Object e) => throw ApiError.transport(e.toString()));
    if (response.statusCode == 401) {
      _handleUnauthorized(gen);
      throw ApiError.http(401, 'unauthorized');
    }
    if (response.statusCode >= 200 && response.statusCode < 300) {
      final list = jsonDecode(response.body) as List<dynamic>;
      return list
          .map((e) => ScheduledHouseEvent.fromJson(e as Map<String, dynamic>))
          .toList();
    }
    throw ApiError.http(response.statusCode, response.body);
  }

  Future<ScheduledHouseEvent> createScheduledEvent(
      String id, Map<String, dynamic> body) async {
    final gen = _generation;
    return _send(
      () => _http.post(
        Uri.parse('$baseUrl/houses/$id/scheduled-events'),
        headers: _headers(),
        body: jsonEncode(body),
      ),
      ScheduledHouseEvent.fromJson,
      generation: gen,
    );
  }

  Future<ScheduledHouseEvent> updateScheduledEvent(
      String houseId, String eventId, Map<String, dynamic> body) async {
    final gen = _generation;
    return _send(
      () => _http.put(
        Uri.parse('$baseUrl/houses/$houseId/scheduled-events/$eventId'),
        headers: _headers(),
        body: jsonEncode(body),
      ),
      ScheduledHouseEvent.fromJson,
      generation: gen,
    );
  }

  Future<void> deleteScheduledEvent(String houseId, String eventId) async {
    final gen = _generation;
    await _empty(
      () => _http.delete(
        Uri.parse('$baseUrl/houses/$houseId/scheduled-events/$eventId'),
        headers: _headers(),
      ),
      generation: gen,
    );
  }

  Future<List<GroceryItem>> getGroceries(String houseId) async {
    final gen = _generation;
    final response = await _http
        .get(Uri.parse('$baseUrl/houses/$houseId/groceries'),
            headers: _headers())
        .catchError((Object e) => throw ApiError.transport(e.toString()));
    if (response.statusCode == 401) {
      _handleUnauthorized(gen);
      throw ApiError.http(401, 'unauthorized');
    }
    if (response.statusCode >= 200 && response.statusCode < 300) {
      final list = jsonDecode(response.body) as List<dynamic>;
      return list
          .map((e) => GroceryItem.fromJson(e as Map<String, dynamic>))
          .toList();
    }
    throw ApiError.http(response.statusCode, response.body);
  }

  Future<GroceryItem> createGrocery(
      String houseId, Map<String, dynamic> body) async {
    final gen = _generation;
    return _send(
      () => _http.post(
        Uri.parse('$baseUrl/houses/$houseId/groceries'),
        headers: _headers(),
        body: jsonEncode(body),
      ),
      GroceryItem.fromJson,
      generation: gen,
    );
  }

  Future<GroceryItem> updateGrocery(
      String houseId, String id, Map<String, dynamic> body) async {
    final gen = _generation;
    return _send(
      () => _http.put(
        Uri.parse('$baseUrl/houses/$houseId/groceries/$id'),
        headers: _headers(),
        body: jsonEncode(body),
      ),
      GroceryItem.fromJson,
      generation: gen,
    );
  }

  Future<GroceryItem> toggleGrocery(
      String houseId, String id, bool checked, int version) async {
    final gen = _generation;
    return _send(
      () => _http.post(
        Uri.parse('$baseUrl/houses/$houseId/groceries/$id/toggle'),
        headers: _headers(),
        body: jsonEncode({'checked': checked, 'version': version}),
      ),
      GroceryItem.fromJson,
      generation: gen,
    );
  }

  Future<void> deleteGrocery(String houseId, String id, int version) async {
    final gen = _generation;
    await _empty(
      () => _http.delete(
        Uri.parse('$baseUrl/houses/$houseId/groceries/$id?version=$version'),
        headers: _headers(),
      ),
      generation: gen,
    );
  }

  Future<List<Chore>> getChores(String houseId) async {
    final gen = _generation;
    final response = await _http
        .get(Uri.parse('$baseUrl/houses/$houseId/chores'), headers: _headers())
        .catchError((Object e) => throw ApiError.transport(e.toString()));
    if (response.statusCode == 401) {
      _handleUnauthorized(gen);
      throw ApiError.http(401, 'unauthorized');
    }
    if (response.statusCode >= 200 && response.statusCode < 300) {
      final list = jsonDecode(response.body) as List<dynamic>;
      return list
          .map((e) => Chore.fromJson(e as Map<String, dynamic>))
          .toList();
    }
    throw ApiError.http(response.statusCode, response.body);
  }

  Future<Chore> createChore(String houseId, Map<String, dynamic> body) async {
    final gen = _generation;
    return _send(
      () => _http.post(
        Uri.parse('$baseUrl/houses/$houseId/chores'),
        headers: _headers(),
        body: jsonEncode(body),
      ),
      Chore.fromJson,
      generation: gen,
    );
  }

  Future<Chore> updateChore(
      String houseId, String id, Map<String, dynamic> body) async {
    final gen = _generation;
    return _send(
      () => _http.put(
        Uri.parse('$baseUrl/houses/$houseId/chores/$id'),
        headers: _headers(),
        body: jsonEncode(body),
      ),
      Chore.fromJson,
      generation: gen,
    );
  }

  Future<void> deleteChore(String houseId, String id, int version) async {
    final gen = _generation;
    await _empty(
      () => _http.delete(
        Uri.parse('$baseUrl/houses/$houseId/chores/$id?version=$version'),
        headers: _headers(),
      ),
      generation: gen,
    );
  }

  Future<void> completeChore(
      String houseId, String id, String occurrenceAt) async {
    final gen = _generation;
    await _empty(
      () => _http.post(
        Uri.parse('$baseUrl/houses/$houseId/chores/$id/completions'),
        headers: _headers(),
        body: jsonEncode({'occurrence_at': occurrenceAt}),
      ),
      generation: gen,
    );
  }

  Future<List<CalendarEvent>> getCalendar(String houseId) async {
    final gen = _generation;
    final response = await _http
        .get(Uri.parse('$baseUrl/houses/$houseId/calendar'),
            headers: _headers())
        .catchError((Object e) => throw ApiError.transport(e.toString()));
    if (response.statusCode == 401) {
      _handleUnauthorized(gen);
      throw ApiError.http(401, 'unauthorized');
    }
    if (response.statusCode >= 200 && response.statusCode < 300) {
      final list = jsonDecode(response.body) as List<dynamic>;
      return list
          .map((e) => CalendarEvent.fromJson(e as Map<String, dynamic>))
          .toList();
    }
    throw ApiError.http(response.statusCode, response.body);
  }

  Future<CalendarEvent> createCalendarEvent(
      String houseId, Map<String, dynamic> body) async {
    final gen = _generation;
    return _send(
      () => _http.post(
        Uri.parse('$baseUrl/houses/$houseId/calendar'),
        headers: _headers(),
        body: jsonEncode(body),
      ),
      CalendarEvent.fromJson,
      generation: gen,
    );
  }

  Future<CalendarEvent> updateCalendarEvent(
      String houseId, String id, Map<String, dynamic> body) async {
    final gen = _generation;
    return _send(
      () => _http.put(
        Uri.parse('$baseUrl/houses/$houseId/calendar/$id'),
        headers: _headers(),
        body: jsonEncode(body),
      ),
      CalendarEvent.fromJson,
      generation: gen,
    );
  }

  Future<void> deleteCalendarEvent(
      String houseId, String id, int version) async {
    final gen = _generation;
    await _empty(
      () => _http.delete(
        Uri.parse('$baseUrl/houses/$houseId/calendar/$id?version=$version'),
        headers: _headers(),
      ),
      generation: gen,
    );
  }

  Future<ChatPage> getChat(String houseId, {int? before}) async {
    final gen = _generation;
    final suffix = before != null ? '?before=$before&limit=50' : '?limit=50';
    return _send(
      () => _http.get(
        Uri.parse('$baseUrl/houses/$houseId/chat$suffix'),
        headers: _headers(),
      ),
      ChatPage.fromJson,
      generation: gen,
    );
  }

  Future<ChatMessage> createChat(String houseId, String body) async {
    final gen = _generation;
    return _send(
      () => _http.post(
        Uri.parse('$baseUrl/houses/$houseId/chat'),
        headers: _headers(),
        body: jsonEncode({'body': body}),
      ),
      ChatMessage.fromJson,
      generation: gen,
    );
  }

  Future<ChatMessage> updateChat(String houseId, String id, String body) async {
    final gen = _generation;
    return _send(
      () => _http.put(
        Uri.parse('$baseUrl/houses/$houseId/chat/$id'),
        headers: _headers(),
        body: jsonEncode({'body': body}),
      ),
      ChatMessage.fromJson,
      generation: gen,
    );
  }

  Future<void> deleteChat(String houseId, String id) async {
    final gen = _generation;
    await _empty(
      () => _http.delete(
        Uri.parse('$baseUrl/houses/$houseId/chat/$id'),
        headers: _headers(),
      ),
      generation: gen,
    );
  }

  Future<String?> loadPendingInvitation() => _storage.loadPendingInvitation();

  Future<void> savePendingInvitation(String token) =>
      _storage.savePendingInvitation(token);

  Future<void> clearPendingInvitation() => _storage.clearPendingInvitation();

  Future<VapidPublicKey> getVapidPublicKey() async {
    final gen = _generation;
    return _send(
      () => _http.get(
        Uri.parse('$baseUrl/notifications/vapid-public-key'),
        headers: _headers(),
      ),
      VapidPublicKey.fromJson,
      generation: gen,
    );
  }
}
