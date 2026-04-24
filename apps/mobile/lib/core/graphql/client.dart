import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:graphql_flutter/graphql_flutter.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:supabase_flutter/supabase_flutter.dart';

import '../../features/auth/providers/auth_provider.dart';

part 'client.g.dart';

@riverpod
GraphQLClient graphQLClient(Ref ref) {
  // Recreate client whenever session changes (login, logout, token rotation)
  ref.listen(authStateProvider, (_, __) => ref.invalidateSelf());

  final session = Supabase.instance.client.auth.currentSession;
  final token = session?.accessToken;
  final bearerToken = token != null ? 'Bearer $token' : null;

  // AuthLink injects JWT into HTTP request headers (queries + mutations)
  final authLink = AuthLink(getToken: () async => bearerToken);

  final httpLink = HttpLink(const String.fromEnvironment('GRAPHQL_API_URL'));

  // WebSocketLink does NOT forward AuthLink headers.
  // JWT is passed via connectionParams (initialPayload) instead.
  final wsLink = WebSocketLink(
    const String.fromEnvironment('GRAPHQL_WS_URL'),
    config: SocketClientConfig(
      initialPayload:
          () async => bearerToken != null ? {'Authorization': bearerToken} : {},
    ),
  );

  final link = authLink.concat(
    Link.split((request) => request.isSubscription, wsLink, httpLink),
  );

  return GraphQLClient(link: link, cache: GraphQLCache());
}
