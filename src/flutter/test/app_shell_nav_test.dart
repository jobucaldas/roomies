import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';
import 'package:roomies/api/api_client.dart';
import 'package:roomies/models/models.dart';
import 'package:roomies/state/app_state.dart';
import 'package:roomies/widgets/app_shell.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  AppState makeApp() {
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
    appState.houses = [
      House(
        id: 'h1',
        name: 'Casa João',
        createdAt: '2026-01-01T00:00:00Z',
      ),
      House(
        id: 'h2',
        name: 'Other place',
        createdAt: '2026-01-02T00:00:00Z',
      ),
    ];
    appState.defaultHouseId = 'h1';
    return appState;
  }

  testWidgets('wide shell shows action nav, not house list as primary',
      (tester) async {
    final appState = makeApp();
    final router = GoRouter(
      initialLocation: '/dashboard',
      routes: [
        GoRoute(
          path: '/dashboard',
          builder: (context, state) => AppShell(
            currentHouseId: 'h1',
            child: const Text('dashboard-body'),
          ),
        ),
        GoRoute(
          path: '/house/:id',
          builder: (context, state) => Text('house ${state.pathParameters['id']}'),
        ),
        GoRoute(
          path: '/settings',
          builder: (context, state) => const SizedBox.shrink(),
        ),
      ],
    );

    await tester.binding.setSurfaceSize(const Size(1200, 900));
    addTearDown(() => tester.binding.setSurfaceSize(null));

    await tester.pumpWidget(
      ChangeNotifierProvider.value(
        value: appState,
        child: MediaQuery(
          data: const MediaQueryData(size: Size(1200, 900)),
          child: MaterialApp.router(routerConfig: router),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('HOUSE'), findsOneWidget);
    expect(find.text('Notes'), findsOneWidget);
    expect(find.text('Groceries'), findsOneWidget);
    expect(find.text('Notifications'), findsOneWidget);
    expect(find.text('Members'), findsOneWidget);
    // Nav labels and the brand appear once — the top bar does not repeat them.
    expect(find.text('Home'), findsOneWidget);
    expect(find.text('Roomies'), findsOneWidget);
    // House names appear in the top switcher, not as a sidebar list of many homes.
    expect(find.text('Casa João'), findsWidgets);
    expect(find.byTooltip('Menu'), findsNothing);
    expect(find.byTooltip('Switch house'), findsOneWidget);
  });

  testWidgets('narrow shell exposes same actions via hamburger drawer',
      (tester) async {
    final appState = makeApp();
    final router = GoRouter(
      initialLocation: '/dashboard',
      routes: [
        GoRoute(
          path: '/dashboard',
          builder: (context, state) => AppShell(
            currentHouseId: 'h1',
            child: const Text('dashboard-body'),
          ),
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

    await tester.binding.setSurfaceSize(const Size(390, 844));
    addTearDown(() => tester.binding.setSurfaceSize(null));

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
    expect(find.byTooltip('Switch house'), findsOneWidget);

    await tester.tap(find.byTooltip('Menu'));
    await tester.pumpAndSettle();

    expect(find.text('HOUSE'), findsOneWidget);
    expect(find.text('Notes'), findsWidgets);
    expect(find.text('Groceries'), findsWidgets);
    expect(find.text('Expenses'), findsWidgets);
  });

  testWidgets('narrow shell centers house switcher in the top bar',
      (tester) async {
    final appState = makeApp();
    final router = GoRouter(
      initialLocation: '/dashboard',
      routes: [
        GoRoute(
          path: '/dashboard',
          builder: (context, state) => AppShell(
            currentHouseId: 'h1',
            child: const Text('dashboard-body'),
          ),
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

    const size = Size(390, 844);
    await tester.binding.setSurfaceSize(size);
    addTearDown(() => tester.binding.setSurfaceSize(null));

    await tester.pumpWidget(
      ChangeNotifierProvider.value(
        value: appState,
        child: MediaQuery(
          data: const MediaQueryData(size: size),
          child: MaterialApp.router(routerConfig: router),
        ),
      ),
    );
    await tester.pumpAndSettle();

    final switcher = tester.getCenter(find.byTooltip('Switch house'));
    expect(switcher.dx, closeTo(size.width / 2, 20));
  });

  testWidgets('wide shell centers house switcher in the main content bar',
      (tester) async {
    final appState = makeApp();
    final router = GoRouter(
      initialLocation: '/dashboard',
      routes: [
        GoRoute(
          path: '/dashboard',
          builder: (context, state) => AppShell(
            currentHouseId: 'h1',
            child: const Text('dashboard-body'),
          ),
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

    const size = Size(1200, 900);
    await tester.binding.setSurfaceSize(size);
    addTearDown(() => tester.binding.setSurfaceSize(null));

    await tester.pumpWidget(
      ChangeNotifierProvider.value(
        value: appState,
        child: MediaQuery(
          data: const MediaQueryData(size: size),
          child: MaterialApp.router(routerConfig: router),
        ),
      ),
    );
    await tester.pumpAndSettle();

    // Sidebar is 248 wide; main content starts after it. Switcher should be
    // centered in that main column, not the full window.
    const sidebarWidth = 248.0;
    final mainCenterX = sidebarWidth + (size.width - sidebarWidth) / 2;
    final switcher = tester.getCenter(find.byTooltip('Switch house'));
    expect(switcher.dx, closeTo(mainCenterX, 24));
  });
}
