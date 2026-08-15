import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:google_fonts/google_fonts.dart';
import 'package:sentry_flutter/sentry_flutter.dart';
import 'package:supabase_flutter/supabase_flutter.dart';

import 'app.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();

  await Supabase.initialize(
    url: const String.fromEnvironment('SUPABASE_URL'),
    anonKey: const String.fromEnvironment('SUPABASE_ANON_KEY'),
  );

  GoogleFonts.config.allowRuntimeFetching = false;

  // Crash reporting is opt-in: without SENTRY_DSN in .env.json the app runs
  // exactly as before, nothing is sent anywhere.
  const sentryDsn = String.fromEnvironment('SENTRY_DSN');
  if (sentryDsn.isEmpty) {
    runApp(const ProviderScope(child: App()));
    return;
  }
  await SentryFlutter.init(
    (options) => options.dsn = sentryDsn,
    appRunner: () => runApp(const ProviderScope(child: App())),
  );
}
