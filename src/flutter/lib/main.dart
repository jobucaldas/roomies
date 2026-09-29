import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_web_plugins/url_strategy.dart';
import 'package:provider/provider.dart';

import 'api/api_client.dart';
import 'router.dart';
import 'state/app_state.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  usePathUrlStrategy();
  final api = ApiClient(webOrigin: kIsWeb ? Uri.base.origin : null);
  final appState = AppState(api);
  await appState.restoreSession();
  runApp(RoomiesApp(appState: appState));
}

class RoomiesApp extends StatefulWidget {
  const RoomiesApp({super.key, required this.appState});

  final AppState appState;

  @override
  State<RoomiesApp> createState() => _RoomiesAppState();
}

class _RoomiesAppState extends State<RoomiesApp> {
  late final _router = createRouter(widget.appState);

  @override
  Widget build(BuildContext context) {
    return ChangeNotifierProvider.value(
      value: widget.appState,
      child: MaterialApp.router(
        title: 'Roomies',
        theme: ThemeData(
          scaffoldBackgroundColor: const Color(0xFFF0F2F5),
          colorScheme: ColorScheme.fromSeed(seedColor: const Color(0xFF1A73E8)),
          useMaterial3: true,
        ),
        routerConfig: _router,
        builder: (context, child) {
          if (!widget.appState.sessionReady) {
            return const Scaffold(
              body: Center(child: Text('Restoring session…')),
            );
          }
          return child ?? const SizedBox.shrink();
        },
      ),
    );
  }
}
