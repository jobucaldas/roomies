import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:roomies/theme/roomies_theme.dart';

void main() {
  test('mint and plum seed accents match João brand hexes', () {
    expect(BrandAccent.mint.seed, const Color(0xFF21C68F));
    expect(BrandAccent.plum.seed, const Color(0xFF53134B));
  });

  test('brand palettes wire seed into light primary teal slot', () {
    final mint = RoomiesPalette.forBrand(BrandAccent.mint, Brightness.light);
    final plum = RoomiesPalette.forBrand(BrandAccent.plum, Brightness.light);
    expect(mint.teal, const Color(0xFF21C68F));
    expect(plum.teal, const Color(0xFF53134B));
    expect(mint.teal, isNot(plum.teal));
  });

  test('buildRoomiesTheme exposes brand primary on ColorScheme', () {
    final mint = buildRoomiesTheme(
      brightness: Brightness.light,
      brand: BrandAccent.mint,
    );
    final plum = buildRoomiesTheme(
      brightness: Brightness.dark,
      brand: BrandAccent.plum,
    );
    expect(mint.colorScheme.primary, const Color(0xFF21C68F));
    expect(plum.colorScheme.primary, const Color(0xFFC45BB0));
  });
}
