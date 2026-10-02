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

  /// Brand swatch shown in Settings. Fills use AA-safe shades per mode.
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

/// Roomies visual system — brand accent over pinned neutral surfaces.
///
/// Surfaces stay neutral (stone in light, graphite in dark) so the accent is
/// the only color on screen; fills use a shade dark/light enough for AA
/// contrast with their label. Fonts are bundled under `assets/fonts/` (see
/// `pubspec.yaml`) so the boot splash and every screen render the same
/// typefaces. Prefer [RoomiesPalette.of] in widgets so light/dark tokens stay
/// in sync. Field names `teal*` are historical accent slots (mint or plum).
@immutable
class RoomiesPalette extends ThemeExtension<RoomiesPalette> {
  const RoomiesPalette({
    required this.ink,
    required this.inkMuted,
    required this.teal,
    required this.onTeal,
    required this.tealDeep,
    required this.tealSoft,
    required this.onTealSoft,
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

  /// Accent fill (primary buttons, selected chips, chart bars).
  final Color teal;

  /// Label color on [teal] fills.
  final Color onTeal;

  /// Accent used for text/icons on [canvas] or [surface].
  final Color tealDeep;

  /// Soft accent container (selected nav rows, badges).
  final Color tealSoft;

  /// Label color on [tealSoft].
  final Color onTealSoft;

  /// Neutral subtle fill (tracks, inactive chips).
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

  // Pinned neutrals — do not shift with accent.
  static const _lightInk = Color(0xFF1C1917);
  static const _lightInkMuted = Color(0xFF57534E);
  static const _lightCanvas = Color(0xFFF5F5F4);
  static const _lightSurface = Color(0xFFFFFFFF);
  static const _lightLine = Color(0xFFE2DFDC);
  static const _lightMist = Color(0xFFEDEBE9);

  static const _darkInk = Color(0xFFF5F5F5);
  static const _darkInkMuted = Color(0xFFB8B8B8);
  static const _darkCanvas = Color(0xFF121212);
  static const _darkSurface = Color(0xFF1C1C1C);
  static const _darkSurfaceRaised = Color(0xFF242424);
  static const _darkLine = Color(0xFF333333);
  static const _darkMist = Color(0xFF2A2A2A);

  static RoomiesPalette _lightFor(BrandAccent accent) {
    final (teal, deep, soft, onSoft) = switch (accent) {
      BrandAccent.mint => (
          const Color(0xFF0F7F5C),
          const Color(0xFF0D7353),
          const Color(0xFFDCF1E8),
          const Color(0xFF0B5940),
        ),
      BrandAccent.plum => (
          const Color(0xFF53134B),
          const Color(0xFF6E2264),
          const Color(0xFFF2E6F0),
          const Color(0xFF4A1143),
        ),
    };
    return RoomiesPalette(
      ink: _lightInk,
      inkMuted: _lightInkMuted,
      teal: teal,
      onTeal: Colors.white,
      tealDeep: deep,
      tealSoft: soft,
      onTealSoft: onSoft,
      mist: _lightMist,
      canvas: _lightCanvas,
      surface: _lightSurface,
      surfaceRaised: _lightSurface,
      line: _lightLine,
      danger: const Color(0xFFB42318),
      dangerSoft: const Color(0xFFFCE8E6),
      success: const Color(0xFF15803D),
      shadow: const Color(0x0F1C1917),
    );
  }

  static RoomiesPalette _darkFor(BrandAccent accent) {
    final (teal, onTeal, deep, soft, onSoft) = switch (accent) {
      BrandAccent.mint => (
          const Color(0xFF21C68F),
          const Color(0xFF062419),
          const Color(0xFF5FD6A8),
          const Color(0xFF17392D),
          const Color(0xFFB4EFD7),
        ),
      BrandAccent.plum => (
          const Color(0xFFD38AC7),
          const Color(0xFF2A0A26),
          const Color(0xFFE3A6D8),
          const Color(0xFF3D1B39),
          const Color(0xFFF5D7EF),
        ),
    };
    return RoomiesPalette(
      ink: _darkInk,
      inkMuted: _darkInkMuted,
      teal: teal,
      onTeal: onTeal,
      tealDeep: deep,
      tealSoft: soft,
      onTealSoft: onSoft,
      mist: _darkMist,
      canvas: _darkCanvas,
      surface: _darkSurface,
      surfaceRaised: _darkSurfaceRaised,
      line: _darkLine,
      danger: const Color(0xFFFB7185),
      dangerSoft: const Color(0xFF3A1C1F),
      success: const Color(0xFF4ADE80),
      shadow: const Color(0x00000000),
    );
  }

  @override
  RoomiesPalette copyWith({
    Color? ink,
    Color? inkMuted,
    Color? teal,
    Color? onTeal,
    Color? tealDeep,
    Color? tealSoft,
    Color? onTealSoft,
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
      onTeal: onTeal ?? this.onTeal,
      tealDeep: tealDeep ?? this.tealDeep,
      tealSoft: tealSoft ?? this.tealSoft,
      onTealSoft: onTealSoft ?? this.onTealSoft,
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
      onTeal: mix(onTeal, other.onTeal),
      tealDeep: mix(tealDeep, other.tealDeep),
      tealSoft: mix(tealSoft, other.tealSoft),
      onTealSoft: mix(onTealSoft, other.onTealSoft),
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

/// Bundled font families (see `pubspec.yaml` → `flutter.fonts`).
const roomiesDisplayFamily = 'Fraunces';
const roomiesBodyFamily = 'Source Sans 3';

ThemeData buildRoomiesTheme({
  Brightness brightness = Brightness.light,
  BrandAccent brand = BrandAccent.mint,
}) {
  final palette = RoomiesPalette.forBrand(brand, brightness);
  final dark = brightness == Brightness.dark;

  final colorScheme = ColorScheme(
    brightness: brightness,
    primary: palette.teal,
    onPrimary: palette.onTeal,
    primaryContainer: palette.tealSoft,
    onPrimaryContainer: palette.onTealSoft,
    secondary: palette.ink,
    onSecondary: palette.canvas,
    secondaryContainer: palette.mist,
    onSecondaryContainer: palette.ink,
    tertiary: palette.tealDeep,
    onTertiary: palette.onTeal,
    error: palette.danger,
    onError: dark ? const Color(0xFF2A0A0F) : Colors.white,
    errorContainer: palette.dangerSoft,
    onErrorContainer: palette.danger,
    surface: palette.surface,
    onSurface: palette.ink,
    onSurfaceVariant: palette.inkMuted,
    surfaceContainerLowest: palette.canvas,
    surfaceContainerLow: palette.surface,
    surfaceContainer: palette.surface,
    surfaceContainerHigh: palette.surfaceRaised,
    surfaceContainerHighest: palette.mist,
    outline: palette.line,
    outlineVariant: palette.mist,
    shadow: Colors.black,
    scrim: Colors.black,
    inverseSurface: palette.ink,
    onInverseSurface: palette.canvas,
    inversePrimary: palette.tealSoft,
    surfaceTint: Colors.transparent,
  );

  final base = ThemeData(useMaterial3: true, brightness: brightness).textTheme;
  final textTheme = base.apply(
    fontFamily: roomiesBodyFamily,
    displayColor: palette.ink,
    bodyColor: palette.ink,
  );

  TextStyle? display(TextStyle? style) =>
      style?.copyWith(fontFamily: roomiesDisplayFamily, color: palette.ink);

  final themed = textTheme.copyWith(
    displayLarge: display(textTheme.displayLarge)?.copyWith(
      fontWeight: FontWeight.w700,
      letterSpacing: -0.6,
    ),
    displayMedium: display(textTheme.displayMedium)?.copyWith(
      fontWeight: FontWeight.w600,
    ),
    headlineLarge: display(textTheme.headlineLarge)?.copyWith(
      fontWeight: FontWeight.w700,
      fontSize: 40,
      height: 1.05,
      letterSpacing: -0.8,
    ),
    headlineMedium: display(textTheme.headlineMedium)?.copyWith(
      fontWeight: FontWeight.w600,
      fontSize: 28,
      height: 1.15,
      letterSpacing: -0.3,
    ),
    headlineSmall: display(textTheme.headlineSmall)?.copyWith(
      fontWeight: FontWeight.w600,
      fontSize: 22,
      height: 1.2,
    ),
    titleLarge: textTheme.titleLarge?.copyWith(
      color: palette.ink,
      fontWeight: FontWeight.w700,
      fontSize: 19,
      letterSpacing: -0.1,
    ),
    titleMedium: textTheme.titleMedium?.copyWith(
      color: palette.ink,
      fontWeight: FontWeight.w600,
    ),
    titleSmall: textTheme.titleSmall?.copyWith(
      color: palette.ink,
      fontWeight: FontWeight.w600,
    ),
    bodyLarge: textTheme.bodyLarge?.copyWith(
      color: palette.ink,
      fontSize: 16,
      height: 1.45,
    ),
    bodyMedium: textTheme.bodyMedium?.copyWith(
      color: palette.inkMuted,
      fontSize: 15,
      height: 1.45,
    ),
    bodySmall: textTheme.bodySmall?.copyWith(
      color: palette.inkMuted,
      height: 1.4,
    ),
    labelLarge: textTheme.labelLarge?.copyWith(
      fontWeight: FontWeight.w700,
      letterSpacing: 0.1,
    ),
  );

  final fieldBorder = OutlineInputBorder(
    borderRadius: BorderRadius.circular(12),
    borderSide: BorderSide(color: palette.line),
  );

  return ThemeData(
    useMaterial3: true,
    brightness: brightness,
    colorScheme: colorScheme,
    scaffoldBackgroundColor: palette.canvas,
    canvasColor: palette.canvas,
    textTheme: themed,
    fontFamily: roomiesBodyFamily,
    extensions: [palette],
    appBarTheme: AppBarTheme(
      backgroundColor: Colors.transparent,
      surfaceTintColor: Colors.transparent,
      elevation: 0,
      foregroundColor: palette.ink,
      titleTextStyle: themed.titleLarge,
    ),
    cardTheme: CardThemeData(
      color: palette.surface,
      surfaceTintColor: Colors.transparent,
      elevation: 0,
      margin: const EdgeInsets.symmetric(vertical: 8),
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(18),
        side: BorderSide(color: palette.line),
      ),
    ),
    drawerTheme: DrawerThemeData(
      backgroundColor: palette.canvas,
      surfaceTintColor: Colors.transparent,
    ),
    dialogTheme: DialogThemeData(
      backgroundColor: palette.surface,
      surfaceTintColor: Colors.transparent,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(18)),
    ),
    popupMenuTheme: PopupMenuThemeData(
      color: palette.surfaceRaised,
      surfaceTintColor: Colors.transparent,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(14),
        side: BorderSide(color: palette.line),
      ),
      textStyle: themed.bodyLarge,
    ),
    listTileTheme: ListTileThemeData(
      textColor: palette.ink,
      iconColor: palette.inkMuted,
      subtitleTextStyle: themed.bodyMedium,
    ),
    inputDecorationTheme: InputDecorationTheme(
      filled: true,
      fillColor: palette.surfaceRaised,
      hintStyle: themed.bodyMedium,
      labelStyle: themed.bodyMedium,
      contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 16),
      border: fieldBorder,
      enabledBorder: fieldBorder,
      focusedBorder: fieldBorder.copyWith(
        borderSide: BorderSide(color: palette.teal, width: 1.6),
      ),
    ),
    filledButtonTheme: FilledButtonThemeData(
      style: FilledButton.styleFrom(
        backgroundColor: palette.teal,
        foregroundColor: palette.onTeal,
        disabledBackgroundColor: palette.mist,
        disabledForegroundColor: palette.inkMuted,
        minimumSize: const Size(48, 48),
        padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 14),
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
        textStyle: themed.labelLarge,
      ),
    ),
    elevatedButtonTheme: ElevatedButtonThemeData(
      style: ElevatedButton.styleFrom(
        backgroundColor: palette.teal,
        foregroundColor: palette.onTeal,
        disabledBackgroundColor: palette.mist,
        elevation: 0,
        minimumSize: const Size(48, 48),
        padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 14),
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
      ),
    ),
    outlinedButtonTheme: OutlinedButtonThemeData(
      style: OutlinedButton.styleFrom(
        foregroundColor: palette.ink,
        side: BorderSide(color: palette.line),
        minimumSize: const Size(48, 48),
        padding: const EdgeInsets.symmetric(horizontal: 18, vertical: 14),
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
        textStyle: themed.labelLarge,
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
    checkboxTheme: CheckboxThemeData(
      fillColor: WidgetStateProperty.resolveWith(
        (states) =>
            states.contains(WidgetState.selected) ? palette.teal : null,
      ),
      checkColor: WidgetStatePropertyAll(palette.onTeal),
    ),
    switchTheme: SwitchThemeData(
      thumbColor: WidgetStateProperty.resolveWith(
        (states) => states.contains(WidgetState.selected)
            ? palette.onTeal
            : palette.inkMuted,
      ),
      trackColor: WidgetStateProperty.resolveWith(
        (states) => states.contains(WidgetState.selected)
            ? palette.teal
            : palette.mist,
      ),
    ),
    segmentedButtonTheme: SegmentedButtonThemeData(
      style: ButtonStyle(
        foregroundColor: WidgetStateProperty.resolveWith(
          (states) => states.contains(WidgetState.selected)
              ? palette.onTealSoft
              : palette.ink,
        ),
        backgroundColor: WidgetStateProperty.resolveWith(
          (states) => states.contains(WidgetState.selected)
              ? palette.tealSoft
              : palette.surface,
        ),
        side: WidgetStatePropertyAll(BorderSide(color: palette.line)),
      ),
    ),
    chipTheme: ChipThemeData(
      backgroundColor: palette.surface,
      selectedColor: palette.tealSoft,
      labelStyle: themed.labelLarge?.copyWith(color: palette.ink),
      secondaryLabelStyle:
          themed.labelLarge?.copyWith(color: palette.onTealSoft),
      side: BorderSide(color: palette.line),
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
    ),
    progressIndicatorTheme: ProgressIndicatorThemeData(
      color: palette.tealDeep,
      linearTrackColor: palette.mist,
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
      backgroundColor: dark ? palette.surfaceRaised : palette.ink,
      contentTextStyle: themed.bodyMedium?.copyWith(
        color: dark ? palette.ink : palette.canvas,
      ),
      actionTextColor: dark ? palette.tealDeep : palette.tealSoft,
      behavior: SnackBarBehavior.floating,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
    ),
  );
}
