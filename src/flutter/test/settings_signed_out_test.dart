import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';
import 'package:roomies/api/api_client.dart';
import 'package:roomies/screens/settings_screen.dart';
import 'package:roomies/state/app_state.dart';
import 'package:roomies/theme/roomies_theme.dart';
import 'package:shared_preferences/shared_preferences.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  setUp(() => SharedPreferences.setMockInitialValues({}));

  testWidgets('signed-out settings offer language incl. Spanish, no houses',
      (tester) async {
    final appState = AppState(
      ApiClient(baseUrl: 'http://example/api'),
      deviceLocale: 'en-US',
    )..sessionReady = true;
    final router = GoRouter(
      initialLocation: '/settings',
      routes: [
        GoRoute(path: '/', builder: (_, __) => const Text('sign-in')),
        GoRoute(path: '/settings', builder: (_, __) => const SettingsScreen()),
      ],
    );
    await tester.binding.setSurfaceSize(const Size(400, 1000));
    addTearDown(() => tester.binding.setSurfaceSize(null));
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

    expect(find.text('Español'), findsOneWidget);
    expect(find.text('Português'), findsOneWidget);
    expect(find.text('Houses'), findsNothing);
    expect(find.byTooltip('Menu'), findsNothing);

    // Currency: defaults to following the language, can be pinned.
    expect(find.text('Follow language'), findsOneWidget);
    await tester.tap(find.byKey(const ValueKey('currency-picker')));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Euro').last);
    await tester.pumpAndSettle();
    expect(appState.currencyOverride, 'EUR');

    await tester.tap(find.text('Español'));
    await tester.pumpAndSettle();
    expect(appState.localeCode, 'es');
    expect(find.text('Configuración'), findsOneWidget);

    await tester.tap(find.byTooltip('Volver'));
    await tester.pumpAndSettle();
    expect(find.text('sign-in'), findsOneWidget);
  });
}
