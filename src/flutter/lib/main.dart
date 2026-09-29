import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/semantics.dart';
import 'package:flutter_web_plugins/url_strategy.dart';
import 'package:provider/provider.dart';

import 'api/api_client.dart';
import 'router.dart';
import 'state/app_state.dart';
import 'theme/roomies_theme.dart';
import 'widgets/roomies_ui.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  if (kIsWeb) {
    SemanticsBinding.instance.ensureSemantics();
  }
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
        theme: buildRoomiesTheme(),
        routerConfig: _router,
        builder: (context, child) {
          if (!widget.appState.sessionReady) {
            return RoomiesAtmosphere(
              child: const Scaffold(
                backgroundColor: Colors.transparent,
                body: Center(child: Text('Restoring session…')),
              ),
            );
          }
          return child ?? const SizedBox.shrink();
        },
      ),
    );
  }
}
