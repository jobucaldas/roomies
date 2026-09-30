import 'package:flutter_test/flutter_test.dart';
import 'package:roomies/api/api_client.dart';
import 'package:roomies/main.dart';
import 'package:roomies/state/app_state.dart';

void main() {
  testWidgets('shows restoring session shell', (tester) async {
    final appState = AppState(
      ApiClient(baseUrl: 'http://example/api'),
      deviceLocale: 'en',
    );
    appState.sessionReady = false;
    await tester.pumpWidget(RoomiesApp(appState: appState));
    expect(find.text('Restoring session…'), findsOneWidget);
  });
}
