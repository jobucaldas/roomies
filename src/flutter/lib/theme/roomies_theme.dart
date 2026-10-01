import 'package:flutter/material.dart';

/// Brand accent families the user can pick in Settings.
enum BrandAccent {
  /// Teal-green `#21C68F`.
  mint,

  /// Deep plum/magenta `#53134B`.
  plum,
}

extension BrandAccentX on BrandAccent {
  String get id => switch (this) {
        BrandAccent.mint => 'mint',
        BrandAccent.plum => 'plum',
      };

  /// Seed accent used for swatches and primary actions.
  Color get seed => switch (this) {
        BrandAccent.mint => const Color(0xFF21C68F),
        BrandAccent.plum => const Color(0xFF53134B),
      };

  static BrandAccent fromId(String? raw) {
    switch (raw) {
      case 'plum':
        return BrandAccent.plum;
      case 'mint':
      default:
        return BrandAccent.mint;
    }
  }
}

/// Roomies visual system — brand accent + light/dark tokens.
///
/// Fonts load via `web/index.html` (Fraunces + Source Sans 3). Prefer
/// [RoomiesPalette.of] in widgets so light/dark tokens stay in sync.
/// Field names `teal*` are historical accent slots (mint or plum).
@immutable
class RoomiesPalette extends ThemeExtension<RoomiesPalette> {
  const RoomiesPalette({
    required this.ink,
    required this.inkMuted,
    required this.teal,
    required this.tealDeep,
    required this.tealSoft,
    required this.mist,
    required this.canvas,
    required this.surface,
    required this.surfaceRaised,
    required this.line,
    required this.danger,
    required this.dangerSoft,
    required this.success,
    required this.shadow,
  });

  final Color ink;
  final Color inkMuted;
  final Color teal;
  final Color tealDeep;
  final Color tealSoft;
  final Color mist;
  final Color canvas;
  final Color surface;
  final Color surfaceRaised;
  final Color line;
  final Color danger;
  final Color dangerSoft;
  final Color success;
  final Color shadow;

  static RoomiesPalette of(BuildContext context) {
    return Theme.of(context).extension<RoomiesPalette>() ??
        forBrand(BrandAccent.mint, Brightness.light);
  }

  static RoomiesPalette forBrand(BrandAccent accent, Brightness brightness) {
    return brightness == Brightness.dark
        ? _darkFor(accent)
        : _lightFor(accent);
  }

  /// Default light mint — kept for legacy static callers.
  static final light = _lightFor(BrandAccent.mint);

  /// Default dark mint — kept for legacy static callers.
  static final dark = _darkFor(BrandAccent.mint);

  static RoomiesPalette _lightFor(BrandAccent accent) {
    switch (accent) {
      case BrandAccent.mint:
        return const RoomiesPalette(
          ink: Color(0xFF14212B),
          inkMuted: Color(0xFF5A6B76),
          teal: Color(0xFF21C68F),
          tealDeep: Color(0xFF0F8A62),
          tealSoft: Color(0xFFD4F5E9),
          mist: Color(0xFFE6F3EE),
          canvas: Color(0xFFF3F8F6),
          surface: Color(0xFFFFFFF8),
          surfaceRaised: Color(0xFFFFFFFF),
          line: Color(0xFFD3E2DB),
          danger: Color(0xFFB42318),
          dangerSoft: Color(0xFFFCE8E6),
          success: Color(0xFF1B7A4A),
          shadow: Color(0x1414212B),
        );
      case BrandAccent.plum:
        return const RoomiesPalette(
          ink: Color(0xFF1C1420),
          inkMuted: Color(0xFF6B5A68),
          teal: Color(0xFF53134B),
          tealDeep: Color(0xFF3A0D34),
          tealSoft: Color(0xFFF3E4F0),
          mist: Color(0xFFF0E6ED),
          canvas: Color(0xFFF8F4F7),
          surface: Color(0xFFFFFFF8),
          surfaceRaised: Color(0xFFFFFFFF),
          line: Color(0xFFE0D3DE),
          danger: Color(0xFFB42318),
          dangerSoft: Color(0xFFFCE8E6),
          success: Color(0xFF1B7A4A),
          shadow: Color(0x141C1420),
        );
    }
  }

  static RoomiesPalette _darkFor(BrandAccent accent) {
    switch (accent) {
      case BrandAccent.mint:
        return const RoomiesPalette(
          ink: Color(0xFFE6EEF1),
          inkMuted: Color(0xFF9AADB8),
          teal: Color(0xFF21C68F),
          tealDeep: Color(0xFF7AD9B4),
          tealSoft: Color(0xFF16382C),
          mist: Color(0xFF1A2C26),
          canvas: Color(0xFF0B1412),
          surface: Color(0xFF121C1A),
          surfaceRaised: Color(0xFF182422),
          line: Color(0xFF2A3A36),
          danger: Color(0xFFFF8A7A),
          dangerSoft: Color(0xFF3A1C1A),
          success: Color(0xFF5DCE8E),
          shadow: Color(0x66000000),
        );
      case BrandAccent.plum:
        return const RoomiesPalette(
          ink: Color(0xFFF0E6EE),
          inkMuted: Color(0xFFB09AAD),
          teal: Color(0xFFC45BB0),
          tealDeep: Color(0xFFE8A8D8),
          tealSoft: Color(0xFF2A1528),
          mist: Color(0xFF241520),
          canvas: Color(0xFF120C11),
          surface: Color(0xFF1A1218),
          surfaceRaised: Color(0xFF241820),
          line: Color(0xFF3A2A38),
          danger: Color(0xFFFF8A7A),
          dangerSoft: Color(0xFF3A1C1A),
          success: Color(0xFF5DCE8E),
          shadow: Color(0x66000000),
        );
    }
  }

  @override
  RoomiesPalette copyWith({
    Color? ink,
    Color? inkMuted,
    Color? teal,
    Color? tealDeep,
    Color? tealSoft,
    Color? mist,
    Color? canvas,
    Color? surface,
    Color? surfaceRaised,
    Color? line,
    Color? danger,
    Color? dangerSoft,
    Color? success,
    Color? shadow,
  }) {
    return RoomiesPalette(
      ink: ink ?? this.ink,
      inkMuted: inkMuted ?? this.inkMuted,
      teal: teal ?? this.teal,
      tealDeep: tealDeep ?? this.tealDeep,
      tealSoft: tealSoft ?? this.tealSoft,
      mist: mist ?? this.mist,
      canvas: canvas ?? this.canvas,
      surface: surface ?? this.surface,
      surfaceRaised: surfaceRaised ?? this.surfaceRaised,
      line: line ?? this.line,
      danger: danger ?? this.danger,
      dangerSoft: dangerSoft ?? this.dangerSoft,
      success: success ?? this.success,
      shadow: shadow ?? this.shadow,
    );
  }

  @override
  RoomiesPalette lerp(ThemeExtension<RoomiesPalette>? other, double t) {
    if (other is! RoomiesPalette) return this;
    Color mix(Color a, Color b) => Color.lerp(a, b, t) ?? a;
    return RoomiesPalette(
      ink: mix(ink, other.ink),
      inkMuted: mix(inkMuted, other.inkMuted),
      teal: mix(teal, other.teal),
      tealDeep: mix(tealDeep, other.tealDeep),
      tealSoft: mix(tealSoft, other.tealSoft),
      mist: mix(mist, other.mist),
      canvas: mix(canvas, other.canvas),
      surface: mix(surface, other.surface),
      surfaceRaised: mix(surfaceRaised, other.surfaceRaised),
      line: mix(line, other.line),
      danger: mix(danger, other.danger),
      dangerSoft: mix(dangerSoft, other.dangerSoft),
      success: mix(success, other.success),
      shadow: mix(shadow, other.shadow),
    );
  }
}

/// Legacy static accessors — light mint tokens only. Prefer [RoomiesPalette.of].
abstract final class RoomiesColors {
  static const ink = Color(0xFF14212B);
  static const inkMuted = Color(0xFF5A6B76);
  static const teal = Color(0xFF21C68F);
  static const tealDeep = Color(0xFF0F8A62);
  static const tealSoft = Color(0xFFD4F5E9);
  static const mist = Color(0xFFE6F3EE);
  static const canvas = Color(0xFFF3F8F6);
  static const surface = Color(0xFFFFFFF8);
  static const line = Color(0xFFD3E2DB);
  static const danger = Color(0xFFB42318);
  static const dangerSoft = Color(0xFFFCE8E6);
  static const success = Color(0xFF1B7A4A);
}

const _displayFamily = 'Fraunces';
const _bodyFamily = 'Source Sans 3';

ThemeData buildRoomiesTheme({
  Brightness brightness = Brightness.light,
  BrandAccent brand = BrandAccent.mint,
}) {
  final palette = RoomiesPalette.forBrand(brand, brightness);
  final onPrimary = brightness == Brightness.dark
      ? const Color(0xFF0A1210)
      : Colors.white;

  final colorScheme = ColorScheme(
    brightness: brightness,
    primary: palette.teal,
    onPrimary: onPrimary,
    primaryContainer: palette.tealSoft,
    onPrimaryContainer: palette.tealDeep,
    secondary: palette.ink,
    onSecondary: palette.canvas,
    secondaryContainer: palette.mist,
    onSecondaryContainer: palette.ink,
    tertiary: palette.tealDeep,
    onTertiary: onPrimary,
    error: palette.danger,
    onError: onPrimary,
    errorContainer: palette.dangerSoft,
    onErrorContainer: palette.danger,
    surface: palette.surface,
    onSurface: palette.ink,
    onSurfaceVariant: palette.inkMuted,
    outline: palette.line,
    outlineVariant: palette.mist,
    shadow: palette.shadow,
    scrim: Colors.black,
    inverseSurface: palette.ink,
    onInverseSurface: palette.canvas,
    inversePrimary: palette.tealSoft,
    surfaceTint: palette.teal,
  );

  final base = ThemeData(useMaterial3: true, brightness: brightness).textTheme;
  final textTheme = base.apply(
    fontFamily: _bodyFamily,
    displayColor: palette.ink,
    bodyColor: palette.ink,
  );

  final themed = textTheme.copyWith(
    displayLarge: textTheme.displayLarge?.copyWith(
      fontFamily: _displayFamily,
      color: palette.ink,
      fontWeight: FontWeight.w700,
      letterSpacing: -0.6,
    ),
    displayMedium: textTheme.displayMedium?.copyWith(
      fontFamily: _displayFamily,
      color: palette.ink,
      fontWeight: FontWeight.w600,
    ),
    headlineLarge: textTheme.headlineLarge?.copyWith(
      fontFamily: _displayFamily,
      color: palette.tealDeep,
      fontWeight: FontWeight.w700,
      fontSize: 42,
      height: 1.02,
      letterSpacing: -1.0,
    ),
    headlineMedium: textTheme.headlineMedium?.copyWith(
      fontFamily: _displayFamily,
      color: palette.ink,
      fontWeight: FontWeight.w600,
      fontSize: 30,
      height: 1.12,
      letterSpacing: -0.4,
    ),
    headlineSmall: textTheme.headlineSmall?.copyWith(
      fontFamily: _displayFamily,
      color: palette.ink,
      fontWeight: FontWeight.w600,
      fontSize: 22,
    ),
    titleLarge: textTheme.titleLarge?.copyWith(
      fontFamily: _bodyFamily,
      color: palette.ink,
      fontWeight: FontWeight.w700,
      fontSize: 20,
      letterSpacing: -0.2,
    ),
    titleMedium: textTheme.titleMedium?.copyWith(
      fontFamily: _bodyFamily,
      color: palette.ink,
      fontWeight: FontWeight.w600,
    ),
    titleSmall: textTheme.titleSmall?.copyWith(
      fontFamily: _bodyFamily,
      color: palette.ink,
      fontWeight: FontWeight.w600,
    ),
    bodyLarge: textTheme.bodyLarge?.copyWith(
      fontFamily: _bodyFamily,
      color: palette.ink,
      fontSize: 16,
      height: 1.5,
    ),
    bodyMedium: textTheme.bodyMedium?.copyWith(
      fontFamily: _bodyFamily,
      color: palette.inkMuted,
      fontSize: 15,
      height: 1.5,
    ),
    bodySmall: textTheme.bodySmall?.copyWith(
      fontFamily: _bodyFamily,
      color: palette.inkMuted,
      height: 1.4,
    ),
    labelLarge: textTheme.labelLarge?.copyWith(
      fontFamily: _bodyFamily,
      fontWeight: FontWeight.w700,
      letterSpacing: 0.15,
    ),
  );

  return ThemeData(
    useMaterial3: true,
    brightness: brightness,
    colorScheme: colorScheme,
    scaffoldBackgroundColor: palette.canvas,
    textTheme: themed,
    fontFamily: _bodyFamily,
    extensions: [palette],
    appBarTheme: AppBarTheme(
      backgroundColor: Colors.transparent,
      elevation: 0,
      foregroundColor: palette.ink,
      titleTextStyle: themed.titleLarge,
    ),
    cardTheme: CardThemeData(
      color: palette.surface,
      elevation: 0,
      margin: const EdgeInsets.symmetric(vertical: 12),
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(20),
        side: BorderSide(color: palette.line),
      ),
    ),
    drawerTheme: DrawerThemeData(
      backgroundColor: palette.surface,
      surfaceTintColor: Colors.transparent,
    ),
    inputDecorationTheme: InputDecorationTheme(
      filled: true,
      fillColor: palette.surfaceRaised,
      hintStyle: themed.bodyMedium,
      contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 16),
      border: OutlineInputBorder(
        borderRadius: BorderRadius.circular(14),
        borderSide: BorderSide(color: palette.line),
      ),
      enabledBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(14),
        borderSide: BorderSide(color: palette.line),
      ),
      focusedBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(14),
        borderSide: BorderSide(color: palette.teal, width: 1.6),
      ),
    ),
    filledButtonTheme: FilledButtonThemeData(
      style: FilledButton.styleFrom(
        backgroundColor: palette.teal,
        foregroundColor: onPrimary,
        disabledBackgroundColor: palette.mist,
        disabledForegroundColor: palette.inkMuted,
        minimumSize: const Size(48, 48),
        padding: const EdgeInsets.symmetric(horizontal: 22, vertical: 14),
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(14)),
        textStyle: themed.labelLarge,
      ),
    ),
    elevatedButtonTheme: ElevatedButtonThemeData(
      style: ElevatedButton.styleFrom(
        backgroundColor: palette.teal,
        foregroundColor: onPrimary,
        disabledBackgroundColor: palette.mist,
        elevation: 0,
        minimumSize: const Size(48, 48),
        padding: const EdgeInsets.symmetric(horizontal: 22, vertical: 14),
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(14)),
      ),
    ),
    outlinedButtonTheme: OutlinedButtonThemeData(
      style: OutlinedButton.styleFrom(
        foregroundColor: palette.ink,
        side: BorderSide(color: palette.line),
        minimumSize: const Size(48, 48),
        padding: const EdgeInsets.symmetric(horizontal: 18, vertical: 14),
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(14)),
      ),
    ),
    textButtonTheme: TextButtonThemeData(
      style: TextButton.styleFrom(
        foregroundColor: palette.tealDeep,
        minimumSize: const Size(48, 48),
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
        textStyle: themed.labelLarge,
      ),
    ),
    iconButtonTheme: IconButtonThemeData(
      style: IconButton.styleFrom(
        foregroundColor: palette.inkMuted,
        minimumSize: const Size(48, 48),
      ),
    ),
    tabBarTheme: TabBarThemeData(
      labelColor: palette.tealDeep,
      unselectedLabelColor: palette.inkMuted,
      indicatorColor: palette.teal,
      dividerColor: palette.line,
      labelStyle: themed.labelLarge,
      unselectedLabelStyle: themed.labelLarge,
    ),
    dividerTheme: DividerThemeData(color: palette.line, space: 24),
    snackBarTheme: SnackBarThemeData(
      backgroundColor: brightness == Brightness.dark
          ? palette.surfaceRaised
          : palette.ink,
      contentTextStyle: themed.bodyMedium?.copyWith(
        color: brightness == Brightness.dark ? palette.ink : Colors.white,
      ),
      behavior: SnackBarBehavior.floating,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
    ),
  );
}
