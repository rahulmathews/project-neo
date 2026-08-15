import 'dart:developer' as developer;

import 'package:graphql_flutter/graphql_flutter.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:supabase_flutter/supabase_flutter.dart';

import '../../../core/graphql/client.dart';
import 'auth_provider.dart';

part 'profile_sync_provider.g.dart';

const _upsertUserMutation = r'''
mutation UpsertUser($input: UpsertUserInput!) {
  upsertUser(input: $input) {
    id
  }
}
''';

/// Mirrors the authenticated Supabase user into the API's `users` table via
/// the `upsertUser` mutation. Match flows reference `users(id)`, and a fresh
/// signup — or a session restored after a `supabase db reset` — has no row
/// yet. Failures are logged and retried on the next auth event or relaunch.
@Riverpod(keepAlive: true)
class ProfileSync extends _$ProfileSync {
  String? _syncedUserId;
  bool _syncing = false;

  @override
  void build() {
    ref.listen(authStateProvider, (_, next) {
      final session =
          next.valueOrNull?.session ??
          Supabase.instance.client.auth.currentSession;
      final user = session?.user;
      if (user != null && user.id != _syncedUserId && !_syncing) {
        _syncing = true;
        _upsert(user).whenComplete(() => _syncing = false);
      }
    });
  }

  Future<void> _upsert(User user) async {
    final metadataName = (user.userMetadata?['name'] as String?)?.trim();
    final name =
        (metadataName == null || metadataName.isEmpty)
            ? (user.email?.split('@').first ?? 'Rider')
            : metadataName;

    final client = ref.read(graphQLClientProvider);
    final result = await client.mutate(
      MutationOptions(
        document: gql(_upsertUserMutation),
        variables: {
          'input': {'name': name},
        },
      ),
    );

    if (result.hasException) {
      developer.log('profile sync failed: ${result.exception}');
      return;
    }
    _syncedUserId = user.id;
  }
}
