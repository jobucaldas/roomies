import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';
import 'package:roomies/api/api_client.dart';
import 'package:roomies/models/models.dart';
import 'package:roomies/state/app_state.dart';
import 'package:roomies/widgets/app_shell.dart';

/// Regression: first-house name field must keep focus across the MediaQuery
/// churn that accompanies a soft keyboard on mobile web (viewInsets + height).
void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  testWidgets(
    'create-house name field retains focus across keyboard viewInsets',
    (tester) async {
      final appState = AppState(
        ApiClient(baseUrl: 'http://example/api'),
        deviceLocale: 'en',
      );
      appState.sessionReady = true;
      appState.user = User(
        id: 'u1',
        email: 'joao@example.com',
        name: 'João',
        createdAt: '2026-01-01T00:00:00Z',
      );
      appState.houses = const [];

      final focusNode = FocusNode(debugLabel: 'create-house-name');
      final controller = TextEditingController();
      addTearDown(() {
        focusNode.dispose();
        controller.dispose();
      });

      Future<void> pumpPhone({
        required double height,
        required double bottomInset,
      }) async {
        final router = GoRouter(
          initialLocation: '/dashboard',
          routes: [
            GoRoute(
              path: '/dashboard',
              builder: (context, state) => AppShell(
                child: ListView(
                  padding: const EdgeInsets.all(16),
                  children: [
                    TextField(
                      key: const ValueKey('create-house-name'),
                      controller: controller,
                      focusNode: focusNode,
                      decoration: const InputDecoration(
                        labelText: 'House name',
                        hintText: 'House name',
                      ),
                    ),
                  ],
                ),
              ),
            ),
            GoRoute(
              path: '/settings',
              builder: (context, state) => const SizedBox.shrink(),
            ),
          ],
        );

        await tester.pumpWidget(
          ChangeNotifierProvider.value(
            value: appState,
            child: MediaQuery(
              data: MediaQueryData(
                size: Size(390, height),
                viewInsets: EdgeInsets.only(bottom: bottomInset),
              ),
              child: MaterialApp.router(routerConfig: router),
            ),
          ),
        );
        await tester.pumpAndSettle();
      }

      // Phone-sized viewport, keyboard closed.
      await pumpPhone(height: 844, bottomInset: 0);

      await tester.tap(find.byKey(const ValueKey('create-house-name')));
      await tester.pump();
      expect(focusNode.hasFocus, isTrue);

      // Soft keyboard opens: shorter view + bottom inset (Android Chrome).
      await pumpPhone(height: 520, bottomInset: 320);
      expect(
        focusNode.hasFocus,
        isTrue,
        reason: 'keyboard inset rebuild must not drop house-name focus',
      );

      // AppState notify (post-auth / refreshHouses) while typing.
      appState.notifyListeners();
      await tester.pump();
      expect(
        focusNode.hasFocus,
        isTrue,
        reason: 'AppState rebuild must not drop house-name focus',
      );

      await tester.enterText(
        find.byKey(const ValueKey('create-house-name')),
        'Casa João',
      );
      expect(controller.text, 'Casa João');
      expect(focusNode.hasFocus, isTrue);
    },
  );

  testWidgets(
    'AppShell keeps drawer chrome when only height changes (narrow)',
    (tester) async {
      final appState = AppState(
        ApiClient(baseUrl: 'http://example/api'),
        deviceLocale: 'en',
      );
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
            builder: (context, state) => AppShell(
              child: const SizedBox.expand(child: Text('body')),
            ),
          ),
          GoRoute(
            path: '/settings',
            builder: (context, state) => const SizedBox.shrink(),
          ),
        ],
      );

      await tester.pumpWidget(
        ChangeNotifierProvider.value(
          value: appState,
          child: MediaQuery(
            data: const MediaQueryData(size: Size(390, 844)),
            child: MaterialApp.router(routerConfig: router),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.byType(Scaffold), findsOneWidget);
      expect(find.byTooltip('Menu'), findsOneWidget);

      // Height shrinks as if the soft keyboard opened; width stays phone-sized.
      await tester.pumpWidget(
        ChangeNotifierProvider.value(
          value: appState,
          child: MediaQuery(
            data: const MediaQueryData(
              size: Size(390, 520),
              viewInsets: EdgeInsets.only(bottom: 320),
            ),
            child: MaterialApp.router(routerConfig: router),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.byType(Scaffold), findsOneWidget);
      expect(find.byTooltip('Menu'), findsOneWidget);
      final scaffold = tester.widget<Scaffold>(find.byType(Scaffold));
      expect(scaffold.resizeToAvoidBottomInset, isFalse);
    },
  );
}
