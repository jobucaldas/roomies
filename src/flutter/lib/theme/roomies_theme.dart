import 'package:flutter/material.dart';

/// Roomies visual system — calm teal household, not a generic SaaS template.
///
/// Fonts load via `web/index.html` (Fraunces + Source Sans 3). Prefer
/// [RoomiesPalette.of] in widgets so light/dark tokens stay in sync.
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
    required this.atmosphere,
    required this.blobPrimary,
    required this.blobSecondary,
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
  final List<Color> atmosphere;
  final Color blobPrimary;
  final Color blobSecondary;
  final Color shadow;

  static RoomiesPalette of(BuildContext context) {
    return Theme.of(context).extension<RoomiesPalette>() ?? light;
  }

  static const light = RoomiesPalette(
    ink: Color(0xFF14212B),
    inkMuted: Color(0xFF5A6B76),
    teal: Color(0xFF0D7377),
    tealDeep: Color(0xFF095456),
    tealSoft: Color(0xFFD7ECEB),
    mist: Color(0xFFE7EEF0),
    canvas: Color(0xFFF3F6F7),
    surface: Color(0xFFFFFFF8),
    surfaceRaised: Color(0xFFFFFFFF),
    line: Color(0xFFD3DEE2),
    danger: Color(0xFFB42318),
    dangerSoft: Color(0xFFFCE8E6),
    success: Color(0xFF1B7A4A),
    atmosphere: [
      Color(0xFFCBE3E1),
      Color(0xFFE7EEF0),
      Color(0xFFF3F6F7),
      Color(0xFFDDE8EA),
    ],
    blobPrimary: Color(0x240D7377),
    blobSecondary: Color(0x1A095456),
    shadow: Color(0x1414212B),
  );

  /// Night household — deep ink teal, not purple/glow SaaS dark.
  static const dark = RoomiesPalette(
    ink: Color(0xFFE6EEF1),
    inkMuted: Color(0xFF9AADB8),
    teal: Color(0xFF3CB8B8),
    tealDeep: Color(0xFF7AD4D1),
    tealSoft: Color(0xFF1A3336),
    mist: Color(0xFF1C2A31),
    canvas: Color(0xFF0B1418),
    surface: Color(0xFF121C22),
    surfaceRaised: Color(0xFF18242B),
    line: Color(0xFF2A3A43),
    danger: Color(0xFFFF8A7A),
    dangerSoft: Color(0xFF3A1C1A),
    success: Color(0xFF5DCE8E),
    atmosphere: [
      Color(0xFF0B1418),
      Color(0xFF102026),
      Color(0xFF0E1A1E),
      Color(0xFF132428),
    ],
    blobPrimary: Color(0x333CB8B8),
    blobSecondary: Color(0x227AD4D1),
    shadow: Color(0x66000000),
  );

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
    List<Color>? atmosphere,
    Color? blobPrimary,
    Color? blobSecondary,
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
      atmosphere: atmosphere ?? this.atmosphere,
      blobPrimary: blobPrimary ?? this.blobPrimary,
      blobSecondary: blobSecondary ?? this.blobSecondary,
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
      atmosphere: [
        for (var i = 0; i < atmosphere.length; i++)
          mix(atmosphere[i], other.atmosphere[i.clamp(0, other.atmosphere.length - 1)]),
      ],
      blobPrimary: mix(blobPrimary, other.blobPrimary),
      blobSecondary: mix(blobSecondary, other.blobSecondary),
      shadow: mix(shadow, other.shadow),
    );
  }
}

/// Legacy static accessors — light tokens only. Prefer [RoomiesPalette.of].
abstract final class RoomiesColors {
  static const ink = Color(0xFF14212B);
  static const inkMuted = Color(0xFF5A6B76);
  static const teal = Color(0xFF0D7377);
  static const tealDeep = Color(0xFF095456);
  static const tealSoft = Color(0xFFD7ECEB);
  static const mist = Color(0xFFE7EEF0);
  static const canvas = Color(0xFFF3F6F7);
  static const surface = Color(0xFFFFFFF8);
  static const line = Color(0xFFD3DEE2);
  static const danger = Color(0xFFB42318);
  static const dangerSoft = Color(0xFFFCE8E6);
  static const success = Color(0xFF1B7A4A);
}

const _displayFamily = 'Fraunces';
const _bodyFamily = 'Source Sans 3';

ThemeData buildRoomiesTheme({Brightness brightness = Brightness.light}) {
  final palette =
      brightness == Brightness.dark ? RoomiesPalette.dark : RoomiesPalette.light;
  final onPrimary =
      brightness == Brightness.dark ? const Color(0xFF062022) : Colors.white;

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
