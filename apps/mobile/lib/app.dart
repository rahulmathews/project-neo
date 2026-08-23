import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'core/router/app_router.dart';
import 'core/theme/app_theme.dart';
import 'features/auth/providers/profile_sync_provider.dart';

class App extends ConsumerWidget {
  const App({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    // Keep the profile sync alive for the whole app lifetime so every
    // sign-in (and restored session) is mirrored into the users table.
    ref.watch(profileSyncProvider);
    final router = ref.watch(appRouterProvider);

    return MaterialApp.router(
      title: 'Neo',
      theme: AppTheme.dark,
      routerConfig: router,
      debugShowCheckedModeBanner: false,
    );
  }
}
