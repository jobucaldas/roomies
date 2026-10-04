import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:provider/provider.dart';
import 'package:roomies/api/api_client.dart';
import 'package:roomies/state/app_state.dart';
import 'package:roomies/widgets/roomies_date_field.dart';

void main() {
  Future<TextEditingController> pump(
    WidgetTester tester,
    RoomiesDateFieldKind kind, {
    bool compact = false,
    bool multiple = false,
  }) async {
    final controller = TextEditingController();
    final app = AppState(
      ApiClient(baseUrl: 'http://example/api'),
      deviceLocale: 'en',
    );
    await tester.pumpWidget(
      ChangeNotifierProvider.value(
        value: app,
        child: MaterialApp(
          home: Scaffold(
            body: RoomiesDateField(
              controller: controller,
              kind: kind,
              compact: compact,
              multiple: multiple,
            ),
          ),
        ),
      ),
    );
    return controller;
  }

  testWidgets('date picker writes yyyy-MM-dd', (tester) async {
    final c = await pump(tester, RoomiesDateFieldKind.date);
    await tester.tap(find.byType(TextField));
    await tester.pumpAndSettle();
    await tester.tap(find.text('OK'));
    await tester.pumpAndSettle();
    expect(c.text, matches(r'^\d{4}-\d{2}-\d{2}$'));
  });

  testWidgets('date-time picker writes 16-char local value', (tester) async {
    final c = await pump(tester, RoomiesDateFieldKind.dateTime);
    await tester.tap(find.byType(TextField));
    await tester.pumpAndSettle();
    await tester.tap(find.text('OK'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('OK'));
    await tester.pumpAndSettle();
    expect(c.text, matches(r'^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$'));
  });

  testWidgets('compact writes RRULE UNTIL form', (tester) async {
    final c = await pump(tester, RoomiesDateFieldKind.dateTime, compact: true);
    await tester.tap(find.byType(TextField));
    await tester.pumpAndSettle();
    await tester.tap(find.text('OK'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('OK'));
    await tester.pumpAndSettle();
    expect(c.text, matches(r'^\d{8}T\d{6}$'));
  });
}
