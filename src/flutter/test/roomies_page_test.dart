import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:roomies/widgets/roomies_ui.dart';

void main() {
  testWidgets('RoomiesPage provides Material for TextField', (tester) async {
    await tester.pumpWidget(
      const MaterialApp(
        home: RoomiesPage(
          child: TextField(decoration: InputDecoration(hintText: 'House name')),
        ),
      ),
    );
    expect(tester.takeException(), isNull);
    expect(find.byType(TextField), findsOneWidget);
    expect(find.byType(Material), findsWidgets);
  });
}
