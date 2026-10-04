import 'dart:async';
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:http/http.dart' as http;
import 'package:provider/provider.dart';
import 'package:roomies/api/api_client.dart';
import 'package:roomies/models/models.dart';
import 'package:roomies/screens/dashboard_screen.dart';
import 'package:roomies/state/app_state.dart';

/// Holds expense GETs until the test releases them, in the order they arrived.
class _GatedExpenses extends http.BaseClient {
  final List<_HeldExpense> held = [];

  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    final path = request.url.path;
    if (path.endsWith('/expenses')) {
      final houseId = path.split('/houses/').last.split('/').first;
      final held = _HeldExpense(houseId);
      this.held.add(held);
      final body = await held.completer.future;
      return _json(body);
    }
    if (path.endsWith('/houses')) {
      return _json(jsonEncode([
        _house('alpha', 'Alpha'),
        _house('beta', 'Beta'),
      ]));
    }
    return _json('[]');
  }

  static Map<String, dynamic> _house(String id, String name) => {
        'id': id,
        'name': name,
        'created_at': '2026-01-01T00:00:00Z',
      };

  static http.StreamedResponse _json(String body) {
    return http.StreamedResponse(
      Stream<List<int>>.value(utf8.encode(body)),
      200,
      headers: const {'content-type': 'application/json; charset=utf-8'},
    );
  }
}

class _HeldExpense {
  _HeldExpense(this.houseId);

  final String houseId;
  final Completer<String> completer = Completer<String>();

  void release(String description) {
    final now = DateTime.now();
    final month =
        '${now.year.toString().padLeft(4, '0')}-${now.month.toString().padLeft(2, '0')}';
    completer.complete(jsonEncode([
      {
        'id': 'expense-$houseId',
        'house_id': houseId,
        'payer_id': 'u1',
        'amount': houseId == 'alpha' ? 10.0 : 80.0,
        'description': description,
        'category': '',
        'date': '$month-01',
        'visibility': 'shared',
        'created_at': '2026-01-02T00:00:00Z',
        'payer_name': 'João',
      }
    ]));
  }
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  testWidgets(
    'a slower summary cannot overwrite the house the user switched to',
    (tester) async {
      final client = _GatedExpenses();
      final api = ApiClient(
        httpClient: client,
        baseUrl: 'http://example.test/api',
      );
      api.adoptCookieSession();
      final appState = AppState(api, deviceLocale: 'en');
      appState.sessionReady = true;
      appState.user = User(
        id: 'u1',
        email: 'joao@example.com',
        name: 'João',
        createdAt: '2026-01-01T00:00:00Z',
      );

      final router = GoRouter(
        initialLocation: '/dashboard',
        routes: [
          GoRoute(
            path: '/dashboard',
            builder: (context, state) => const DashboardScreen(),
          ),
          GoRoute(
            path: '/house/:id',
            builder: (context, state) => const SizedBox.shrink(),
          ),
          GoRoute(
            path: '/settings',
            builder: (context, state) => const SizedBox.shrink(),
          ),
        ],
      );
      addTearDown(router.dispose);

      await tester.pumpWidget(
        ChangeNotifierProvider.value(
          value: appState,
          child: MaterialApp.router(routerConfig: router),
        ),
      );
      await tester.pump();

      expect(client.held, hasLength(1));
      expect(client.held.single.houseId, 'alpha');

      // House switch is the top PopupMenu dropdown (selected label also appears in body).
      expect(find.text('Alpha'), findsWidgets);
      await tester.tap(find.byTooltip('Switch house'));
      // Avoid pumpAndSettle: dashboard note rotation uses a periodic Timer.
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 300));

      expect(find.text('Beta'), findsOneWidget);
      await tester.tap(find.text('Beta'));
      await tester.pump();
      await tester.pump();

      expect(client.held, hasLength(2));
      expect(client.held[1].houseId, 'beta');

      client.held[1].release('Beta rent');
      await tester.pump();
      await tester.pump();

      expect(find.text('Beta rent'), findsOneWidget);
      expect(find.text('Alpha rent'), findsNothing);

      client.held[0].release('Alpha rent');
      await tester.pump();
      await tester.pump();

      expect(find.text('Beta rent'), findsOneWidget);
      expect(find.text('Alpha rent'), findsNothing);
      expect(find.text('Beta'), findsWidgets);
    },
  );
}
