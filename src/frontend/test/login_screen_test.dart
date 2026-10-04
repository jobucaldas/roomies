import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:provider/provider.dart';
import 'package:roomies/api/api_client.dart';
import 'package:roomies/screens/login_screen.dart';
import 'package:roomies/state/app_state.dart';
import 'package:roomies/theme/roomies_theme.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  Future<void> pumpLogin(WidgetTester tester, http.Client client) async {
    final appState = AppState(
      ApiClient(baseUrl: 'http://example/api', httpClient: client),
      deviceLocale: 'en-US',
    )..sessionReady = true;
    final router = GoRouter(
      routes: [
        GoRoute(path: '/', builder: (_, __) => const LoginScreen()),
        GoRoute(path: '/settings', builder: (_, __) => const Text('settings')),
      ],
    );
    await tester.pumpWidget(
      ChangeNotifierProvider.value(
        value: appState,
        child: MaterialApp.router(
          theme: buildRoomiesTheme(),
          routerConfig: router,
        ),
      ),
    );
    await tester.pumpAndSettle();
  }

  testWidgets('sign-in offers Settings instead of a language toggle',
      (tester) async {
    await pumpLogin(
      tester,
      MockClient((_) async => http.Response(
            '{"authkit":true,"password":false}',
            200,
          )),
    );
    expect(find.text('Sign in'), findsOneWidget);
    expect(find.text('PT'), findsNothing);
    await tester.tap(find.byTooltip('Settings'));
    await tester.pumpAndSettle();
    expect(find.text('settings'), findsOneWidget);
  });

  testWidgets('server errors read as unreachable and can be retried',
      (tester) async {
    var calls = 0;
    await pumpLogin(
      tester,
      MockClient((_) async {
        calls++;
        return calls == 1
            ? http.Response('', 502)
            : http.Response('{"authkit":true,"password":false}', 200);
      }),
    );
    expect(find.textContaining("Can't reach Roomies"), findsOneWidget);
    expect(find.text('502'), findsNothing);
    // No guessed password form while the sign-in method is unknown.
    expect(find.text('Use email and password'), findsNothing);

    await tester.tap(find.text('Retry'));
    await tester.pumpAndSettle();
    expect(find.text('Sign in'), findsOneWidget);
    expect(find.textContaining("Can't reach Roomies"), findsNothing);
  });
}
