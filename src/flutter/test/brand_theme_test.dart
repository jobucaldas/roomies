import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:roomies/theme/roomies_theme.dart';

double _luminance(Color c) {
  double lin(double v) =>
      v <= 0.04045 ? v / 12.92 : math.pow((v + 0.055) / 1.055, 2.4).toDouble();
  return 0.2126 * lin(c.r) + 0.7152 * lin(c.g) + 0.0722 * lin(c.b);
}

double _contrast(Color a, Color b) {
  final la = _luminance(a);
  final lb = _luminance(b);
  return (math.max(la, lb) + 0.05) / (math.min(la, lb) + 0.05);
}

void main() {
  test('mint and plum swatches keep the brand hexes', () {
    expect(BrandAccent.mint.seed, const Color(0xFF21C68F));
    expect(BrandAccent.plum.seed, const Color(0xFF53134B));
  });

  test('buildRoomiesTheme wires the palette into ColorScheme', () {
    for (final brand in BrandAccent.values) {
      for (final brightness in Brightness.values) {
        final theme = buildRoomiesTheme(brightness: brightness, brand: brand);
        final p = RoomiesPalette.forBrand(brand, brightness);
        expect(theme.colorScheme.primary, p.teal);
        expect(theme.colorScheme.onPrimary, p.onTeal);
        expect(theme.scaffoldBackgroundColor, p.canvas);
        expect(theme.textTheme.bodyMedium?.fontFamily, roomiesBodyFamily);
        expect(theme.textTheme.headlineMedium?.fontFamily, roomiesDisplayFamily);
      }
    }
  });

  test('surfaces are pinned neutrals shared by every accent', () {
    for (final brightness in Brightness.values) {
      final mint = RoomiesPalette.forBrand(BrandAccent.mint, brightness);
      final plum = RoomiesPalette.forBrand(BrandAccent.plum, brightness);
      expect(mint.canvas, plum.canvas);
      expect(mint.surface, plum.surface);
      expect(mint.ink, plum.ink);
      expect(mint.teal, isNot(plum.teal));
    }
  });

  for (final brand in BrandAccent.values) {
    for (final brightness in Brightness.values) {
      test('${brand.id} $brightness meets WCAG AA', () {
        final p = RoomiesPalette.forBrand(brand, brightness);
        void aa(Color fg, Color bg, String what) => expect(
              _contrast(fg, bg),
              greaterThanOrEqualTo(4.5),
              reason: what,
            );
        aa(p.ink, p.canvas, 'ink on canvas');
        aa(p.inkMuted, p.canvas, 'muted ink on canvas');
        aa(p.inkMuted, p.surface, 'muted ink on surface');
        aa(p.onTeal, p.teal, 'label on accent fill');
        aa(p.onTealSoft, p.tealSoft, 'label on soft accent');
        aa(p.tealDeep, p.canvas, 'accent text on canvas');
        aa(p.tealDeep, p.surface, 'accent text on surface');
        aa(p.danger, p.surface, 'danger on surface');
      });
    }
  }
}
