import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';

const _scaffoldBg = Color(0xFF0D1117);
const _surface = Color(0xFF161B22);
const _primary = Color(0xFF58A6FF);
const _error = Color(0xFFF85149);
const _onSurface = Color(0xFFC9D1D9);
const _onSurfaceVariant = Color(0xFF8B949E);
const _cardBorder = Color(0xFF30363D);

class AppTheme {
  AppTheme._();

  static ThemeData get dark => ThemeData(
    useMaterial3: true,
    brightness: Brightness.dark,
    scaffoldBackgroundColor: _scaffoldBg,
    colorScheme: const ColorScheme.dark(
      surface: _surface,
      primary: _primary,
      error: _error,
      onSurface: _onSurface,
      onSurfaceVariant: _onSurfaceVariant,
    ),
    textTheme: GoogleFonts.interTextTheme(ThemeData.dark().textTheme).copyWith(
      displayLarge: GoogleFonts.interTight(
        fontWeight: FontWeight.bold,
        color: _onSurface,
      ),
      displayMedium: GoogleFonts.interTight(
        fontWeight: FontWeight.bold,
        color: _onSurface,
      ),
      displaySmall: GoogleFonts.interTight(
        fontWeight: FontWeight.bold,
        color: _onSurface,
      ),
      headlineLarge: GoogleFonts.interTight(
        fontWeight: FontWeight.bold,
        color: _onSurface,
      ),
      headlineMedium: GoogleFonts.interTight(
        fontWeight: FontWeight.w600,
        color: _onSurface,
      ),
      headlineSmall: GoogleFonts.interTight(
        fontWeight: FontWeight.w600,
        color: _onSurface,
      ),
    ),
    cardTheme: CardThemeData(
      elevation: 0,
      color: _surface,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(8),
        side: const BorderSide(color: _cardBorder),
      ),
    ),
    drawerTheme: const DrawerThemeData(backgroundColor: _surface),
    dividerColor: _cardBorder,
  );
}
