// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'profile_sync_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

String _$profileSyncHash() => r'74c3f7c6cc817f845c0906b94b4bd95a52b8df77';

/// Mirrors the authenticated Supabase user into the API's `users` table via
/// the `upsertUser` mutation. Match flows reference `users(id)`, and a fresh
/// signup — or a session restored after a `supabase db reset` — has no row
/// yet. Failures are logged and retried on the next auth event or relaunch.
///
/// Copied from [ProfileSync].
@ProviderFor(ProfileSync)
final profileSyncProvider = NotifierProvider<ProfileSync, void>.internal(
  ProfileSync.new,
  name: r'profileSyncProvider',
  debugGetCreateSourceHash:
      const bool.fromEnvironment('dart.vm.product') ? null : _$profileSyncHash,
  dependencies: null,
  allTransitiveDependencies: null,
);

typedef _$ProfileSync = Notifier<void>;
// ignore_for_file: type=lint
// ignore_for_file: subtype_of_sealed_class, invalid_use_of_internal_member, invalid_use_of_visible_for_testing_member, deprecated_member_use_from_same_package
