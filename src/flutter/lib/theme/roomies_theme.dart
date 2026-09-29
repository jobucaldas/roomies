import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';

/// Roomies visual system — calm teal household, not a generic dashboard template.
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

ThemeData buildRoomiesTheme() {
  final colorScheme = ColorScheme(
    brightness: Brightness.light,
    primary: RoomiesColors.teal,
    onPrimary: Colors.white,
    primaryContainer: RoomiesColors.tealSoft,
    onPrimaryContainer: RoomiesColors.tealDeep,
    secondary: RoomiesColors.ink,
    onSecondary: Colors.white,
    secondaryContainer: RoomiesColors.mist,
    onSecondaryContainer: RoomiesColors.ink,
    tertiary: RoomiesColors.tealDeep,
    onTertiary: Colors.white,
    error: RoomiesColors.danger,
    onError: Colors.white,
    errorContainer: RoomiesColors.dangerSoft,
    onErrorContainer: RoomiesColors.danger,
    surface: RoomiesColors.surface,
    onSurface: RoomiesColors.ink,
    onSurfaceVariant: RoomiesColors.inkMuted,
    outline: RoomiesColors.line,
    outlineVariant: RoomiesColors.mist,
    shadow: Colors.black,
    scrim: Colors.black,
    inverseSurface: RoomiesColors.ink,
    onInverseSurface: RoomiesColors.canvas,
    inversePrimary: RoomiesColors.tealSoft,
    surfaceTint: RoomiesColors.teal,
  );

  final display = GoogleFonts.frauncesTextTheme();
  final body = GoogleFonts.sourceSans3TextTheme();

  final textTheme = body.copyWith(
    displayLarge: display.displayLarge?.copyWith(
      color: RoomiesColors.ink,
      fontWeight: FontWeight.w600,
      letterSpacing: -0.5,
    ),
    displayMedium: display.displayMedium?.copyWith(
      color: RoomiesColors.ink,
      fontWeight: FontWeight.w600,
    ),
    headlineLarge: display.headlineLarge?.copyWith(
      color: RoomiesColors.tealDeep,
      fontWeight: FontWeight.w600,
      fontSize: 40,
      height: 1.05,
      letterSpacing: -0.8,
    ),
    headlineMedium: display.headlineMedium?.copyWith(
      color: RoomiesColors.ink,
      fontWeight: FontWeight.w600,
      fontSize: 28,
      height: 1.15,
    ),
    headlineSmall: display.headlineSmall?.copyWith(
      color: RoomiesColors.ink,
      fontWeight: FontWeight.w600,
      fontSize: 22,
    ),
    titleLarge: body.titleLarge?.copyWith(
      color: RoomiesColors.ink,
      fontWeight: FontWeight.w600,
      fontSize: 20,
    ),
    titleMedium: body.titleMedium?.copyWith(
      color: RoomiesColors.ink,
      fontWeight: FontWeight.w600,
    ),
    bodyLarge: body.bodyLarge?.copyWith(
      color: RoomiesColors.ink,
      fontSize: 16,
      height: 1.45,
    ),
    bodyMedium: body.bodyMedium?.copyWith(
      color: RoomiesColors.inkMuted,
      fontSize: 15,
      height: 1.45,
    ),
    labelLarge: body.labelLarge?.copyWith(
      fontWeight: FontWeight.w600,
      letterSpacing: 0.2,
    ),
  );

  return ThemeData(
    useMaterial3: true,
    colorScheme: colorScheme,
    scaffoldBackgroundColor: RoomiesColors.canvas,
    textTheme: textTheme,
    appBarTheme: AppBarTheme(
      backgroundColor: Colors.transparent,
      elevation: 0,
      foregroundColor: RoomiesColors.ink,
      titleTextStyle: textTheme.titleLarge,
    ),
    cardTheme: CardThemeData(
      color: RoomiesColors.surface,
      elevation: 0,
      margin: const EdgeInsets.symmetric(vertical: 10),
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(18),
        side: const BorderSide(color: RoomiesColors.line),
      ),
    ),
    inputDecorationTheme: InputDecorationTheme(
      filled: true,
      fillColor: Colors.white,
      hintStyle: textTheme.bodyMedium,
      contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
      border: OutlineInputBorder(
        borderRadius: BorderRadius.circular(14),
        borderSide: const BorderSide(color: RoomiesColors.line),
      ),
      enabledBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(14),
        borderSide: const BorderSide(color: RoomiesColors.line),
      ),
      focusedBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(14),
        borderSide: const BorderSide(color: RoomiesColors.teal, width: 1.6),
      ),
    ),
    filledButtonTheme: FilledButtonThemeData(
      style: FilledButton.styleFrom(
        backgroundColor: RoomiesColors.teal,
        foregroundColor: Colors.white,
        disabledBackgroundColor: RoomiesColors.mist,
        disabledForegroundColor: RoomiesColors.inkMuted,
        padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 14),
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(14)),
        textStyle: textTheme.labelLarge,
      ),
    ),
    elevatedButtonTheme: ElevatedButtonThemeData(
      style: ElevatedButton.styleFrom(
        backgroundColor: RoomiesColors.teal,
        foregroundColor: Colors.white,
        disabledBackgroundColor: RoomiesColors.mist,
        elevation: 0,
        padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 14),
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(14)),
      ),
    ),
    textButtonTheme: TextButtonThemeData(
      style: TextButton.styleFrom(
        foregroundColor: RoomiesColors.tealDeep,
        textStyle: textTheme.labelLarge,
      ),
    ),
    tabBarTheme: TabBarThemeData(
      labelColor: RoomiesColors.tealDeep,
      unselectedLabelColor: RoomiesColors.inkMuted,
      indicatorColor: RoomiesColors.teal,
      dividerColor: RoomiesColors.line,
      labelStyle: textTheme.labelLarge,
      unselectedLabelStyle: textTheme.labelLarge,
    ),
    dividerTheme: const DividerThemeData(color: RoomiesColors.line),
    snackBarTheme: SnackBarThemeData(
      backgroundColor: RoomiesColors.ink,
      contentTextStyle: textTheme.bodyMedium?.copyWith(color: Colors.white),
      behavior: SnackBarBehavior.floating,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
    ),
  );
}
