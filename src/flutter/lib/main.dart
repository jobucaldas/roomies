import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/semantics.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
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
  final appState = AppState(api, deviceLocale: platformLocaleTag());
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
      child: Consumer<AppState>(
        builder: (context, app, _) {
          final locale = app.localeCode == 'pt'
              ? const Locale('pt', 'BR')
              : const Locale('en');
          return MaterialApp.router(
            title: 'Roomies',
            theme: buildRoomiesTheme(),
            locale: locale,
            supportedLocales: const [
              Locale('en'),
              Locale('pt', 'BR'),
            ],
            localizationsDelegates: const [
              GlobalMaterialLocalizations.delegate,
              GlobalWidgetsLocalizations.delegate,
              GlobalCupertinoLocalizations.delegate,
            ],
            routerConfig: _router,
            builder: (context, child) {
              if (!app.sessionReady) {
                return RoomiesAtmosphere(
                  child: Scaffold(
                    backgroundColor: Colors.transparent,
                    body: Center(child: Text(app.strings.restoringSession)),
                  ),
                );
              }
              return child ?? const SizedBox.shrink();
            },
          );
        },
      ),
    );
  }
}
