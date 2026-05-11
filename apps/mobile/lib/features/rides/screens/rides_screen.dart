import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:graphql_flutter/graphql_flutter.dart';

import '../../../core/graphql/client.dart';
import '../../../core/widgets/error_view.dart';
import '../../../core/widgets/loading_indicator.dart';

const _groupsQuery = r'''
query DriverRideGroups {
  groups {
    id
    name
  }
}
''';

const _ridesQuery = r'''
query DriverRideFeed($groupId: UUID!) {
  rides(
    groupId: $groupId
    type: NEED_RIDE
    status: AVAILABLE
    limit: 50
    offset: 0
  ) {
    id
    type
    fromLocationText
    toLocationText
    departureTime
    isImmediate
    cost
    currency
    distance
    seatsAvailable
    status
    createdAt
    group {
      id
      name
    }
    fromLocationContext {
      locationName
      location {
        latitude
        longitude
      }
    }
    toLocationContext {
      locationName
      location {
        latitude
        longitude
      }
    }
  }
}
''';

class RidesScreen extends ConsumerStatefulWidget {
  const RidesScreen({super.key});

  @override
  ConsumerState<RidesScreen> createState() => _RidesScreenState();
}

class _RidesScreenState extends ConsumerState<RidesScreen> {
  GraphQLClient? _client;
  Future<_RideFeedData>? _future;
  String? _selectedGroupId;

  @override
  Widget build(BuildContext context) {
    final client = ref.watch(graphQLClientProvider);
    if (!identical(_client, client)) {
      _client = client;
      _future = _load(client);
    }

    return FutureBuilder<_RideFeedData>(
      future: _future,
      builder: (context, snapshot) {
        if (snapshot.connectionState == ConnectionState.waiting &&
            !snapshot.hasData) {
          return const LoadingIndicator();
        }
        if (snapshot.hasError) {
          return ErrorView(message: 'Unable to load rides\n${snapshot.error}');
        }

        final data = snapshot.data;
        if (data == null) {
          return const LoadingIndicator();
        }
        if (data.groups.isEmpty) {
          return const _EmptyState(
            title: 'No groups yet',
            message:
                'Start the workers service and let it sync WhatsApp groups before rides can appear here.',
          );
        }

        return Column(
          children: [
            _FeedHeader(
              data: data,
              onGroupChanged: (groupId) => _selectGroup(client, groupId),
            ),
            Expanded(
              child: RefreshIndicator(
                onRefresh: () => _refresh(client),
                child:
                    data.rides.isEmpty
                        ? ListView(
                          physics: const AlwaysScrollableScrollPhysics(),
                          children: const [
                            SizedBox(height: 96),
                            _EmptyState(
                              title: 'No ride requests',
                              message:
                                  'New parsed NEED_RIDE messages for this group will show up here.',
                            ),
                          ],
                        )
                        : ListView.separated(
                          physics: const AlwaysScrollableScrollPhysics(),
                          padding: const EdgeInsets.fromLTRB(16, 0, 16, 24),
                          itemCount: data.rides.length,
                          separatorBuilder:
                              (context, index) => const SizedBox(height: 12),
                          itemBuilder:
                              (context, index) =>
                                  _RideCard(ride: data.rides[index]),
                        ),
              ),
            ),
          ],
        );
      },
    );
  }

  Future<void> _refresh(GraphQLClient client) async {
    final future = _load(client);
    setState(() => _future = future);
    try {
      await future;
    } catch (_) {
      // FutureBuilder renders the error state.
    }
  }

  void _selectGroup(GraphQLClient client, String? groupId) {
    if (groupId == null || groupId == _selectedGroupId) {
      return;
    }
    setState(() {
      _selectedGroupId = groupId;
      _future = _load(client);
    });
  }

  Future<_RideFeedData> _load(GraphQLClient client) async {
    final groupsResult = await client.query(
      QueryOptions(
        document: gql(_groupsQuery),
        fetchPolicy: FetchPolicy.networkOnly,
      ),
    );
    _throwIfGraphQLError(groupsResult);

    final groups =
        ((groupsResult.data?['groups'] as List<dynamic>?) ?? const [])
            .whereType<Map<String, dynamic>>()
            .map(_RideGroup.fromJson)
            .toList();

    if (groups.isEmpty) {
      _selectedGroupId = null;
      return _RideFeedData(
        groups: groups,
        selectedGroupId: null,
        rides: const [],
      );
    }

    final selectedGroupId =
        groups.any((group) => group.id == _selectedGroupId)
            ? _selectedGroupId!
            : groups.first.id;
    _selectedGroupId = selectedGroupId;

    final ridesResult = await client.query(
      QueryOptions(
        document: gql(_ridesQuery),
        variables: {'groupId': selectedGroupId},
        fetchPolicy: FetchPolicy.networkOnly,
      ),
    );
    _throwIfGraphQLError(ridesResult);

    final rides =
        ((ridesResult.data?['rides'] as List<dynamic>?) ?? const [])
            .whereType<Map<String, dynamic>>()
            .map(_Ride.fromJson)
            .toList();

    return _RideFeedData(
      groups: groups,
      selectedGroupId: selectedGroupId,
      rides: rides,
    );
  }
}

void _throwIfGraphQLError(QueryResult<Object?> result) {
  if (result.hasException) {
    throw Exception(result.exception.toString());
  }
}

class _FeedHeader extends StatelessWidget {
  const _FeedHeader({required this.data, required this.onGroupChanged});

  final _RideFeedData data;
  final ValueChanged<String?> onGroupChanged;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 16, 16, 12),
      child: Card(
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                'Driver ride feed',
                style: theme.textTheme.titleLarge?.copyWith(
                  fontWeight: FontWeight.w700,
                ),
              ),
              const SizedBox(height: 6),
              Text(
                'Showing parsed rider requests. Matching and accepting rides will come later.',
                style: theme.textTheme.bodySmall?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant,
                ),
              ),
              const SizedBox(height: 16),
              DropdownButtonFormField<String>(
                initialValue: data.selectedGroupId,
                isExpanded: true,
                decoration: const InputDecoration(
                  labelText: 'Group',
                  border: OutlineInputBorder(),
                ),
                items:
                    data.groups
                        .map(
                          (group) => DropdownMenuItem(
                            value: group.id,
                            child: Text(group.name),
                          ),
                        )
                        .toList(),
                onChanged: onGroupChanged,
              ),
              const SizedBox(height: 12),
              Text(
                '${data.rides.length} available rider request${data.rides.length == 1 ? '' : 's'}',
                style: theme.textTheme.labelLarge?.copyWith(
                  color: theme.colorScheme.primary,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _RideCard extends StatelessWidget {
  const _RideCard({required this.ride});

  final _Ride ride;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Expanded(
                  child: Text(
                    ride.timeLabel,
                    style: theme.textTheme.titleMedium?.copyWith(
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                ),
                _StatusChip(label: ride.status),
              ],
            ),
            const SizedBox(height: 16),
            _Endpoint(
              icon: Icons.trip_origin,
              label: 'Pickup',
              value: ride.fromLabel,
              detail: ride.fromDetail,
              isMapped: ride.fromIsMapped,
            ),
            const SizedBox(height: 12),
            _Endpoint(
              icon: Icons.place_outlined,
              label: 'Drop',
              value: ride.toLabel,
              detail: ride.toDetail,
              isMapped: ride.toIsMapped,
            ),
            const SizedBox(height: 16),
            Wrap(
              spacing: 8,
              runSpacing: 8,
              children: [
                if (ride.costLabel != null)
                  _MetaChip(
                    icon: Icons.payments_outlined,
                    label: ride.costLabel!,
                  ),
                if (ride.distanceLabel != null)
                  _MetaChip(
                    icon: Icons.route_outlined,
                    label: ride.distanceLabel!,
                  ),
                if (ride.seatsLabel != null)
                  _MetaChip(
                    icon: Icons.event_seat_outlined,
                    label: ride.seatsLabel!,
                  ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

class _Endpoint extends StatelessWidget {
  const _Endpoint({
    required this.icon,
    required this.label,
    required this.value,
    required this.detail,
    required this.isMapped,
  });

  final IconData icon;
  final String label;
  final String value;
  final String detail;
  final bool isMapped;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Icon(icon, size: 20, color: theme.colorScheme.primary),
        const SizedBox(width: 12),
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                label,
                style: theme.textTheme.labelSmall?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant,
                ),
              ),
              const SizedBox(height: 2),
              Text(value, style: theme.textTheme.bodyLarge),
              const SizedBox(height: 2),
              Text(
                detail,
                style: theme.textTheme.bodySmall?.copyWith(
                  color:
                      isMapped
                          ? theme.colorScheme.onSurfaceVariant
                          : theme.colorScheme.error,
                ),
              ),
            ],
          ),
        ),
      ],
    );
  }
}

class _StatusChip extends StatelessWidget {
  const _StatusChip({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    return Chip(
      visualDensity: VisualDensity.compact,
      label: Text(label.replaceAll('_', ' ')),
    );
  }
}

class _MetaChip extends StatelessWidget {
  const _MetaChip({required this.icon, required this.label});

  final IconData icon;
  final String label;

  @override
  Widget build(BuildContext context) {
    return Chip(
      avatar: Icon(icon, size: 16),
      visualDensity: VisualDensity.compact,
      label: Text(label),
    );
  }
}

class _EmptyState extends StatelessWidget {
  const _EmptyState({required this.title, required this.message});

  final String title;
  final String message;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.all(24),
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Icon(
            Icons.local_taxi_outlined,
            size: 42,
            color: theme.colorScheme.onSurfaceVariant,
          ),
          const SizedBox(height: 12),
          Text(
            title,
            style: theme.textTheme.titleMedium?.copyWith(
              fontWeight: FontWeight.w700,
            ),
            textAlign: TextAlign.center,
          ),
          const SizedBox(height: 6),
          Text(
            message,
            style: theme.textTheme.bodyMedium?.copyWith(
              color: theme.colorScheme.onSurfaceVariant,
            ),
            textAlign: TextAlign.center,
          ),
        ],
      ),
    );
  }
}

class _RideFeedData {
  const _RideFeedData({
    required this.groups,
    required this.selectedGroupId,
    required this.rides,
  });

  final List<_RideGroup> groups;
  final String? selectedGroupId;
  final List<_Ride> rides;
}

class _RideGroup {
  const _RideGroup({required this.id, required this.name});

  factory _RideGroup.fromJson(Map<String, dynamic> json) {
    return _RideGroup(
      id: json['id'] as String,
      name: json['name'] as String? ?? 'Unnamed group',
    );
  }

  final String id;
  final String name;
}

class _Ride {
  const _Ride({
    required this.id,
    required this.status,
    required this.fromLocationText,
    required this.toLocationText,
    required this.fromContext,
    required this.toContext,
    required this.departureTime,
    required this.isImmediate,
    required this.cost,
    required this.currency,
    required this.distance,
    required this.seatsAvailable,
  });

  factory _Ride.fromJson(Map<String, dynamic> json) {
    return _Ride(
      id: json['id'] as String,
      status: json['status'] as String? ?? 'AVAILABLE',
      fromLocationText: json['fromLocationText'] as String?,
      toLocationText: json['toLocationText'] as String?,
      fromContext: _LocationContext.maybeFromJson(json['fromLocationContext']),
      toContext: _LocationContext.maybeFromJson(json['toLocationContext']),
      departureTime: json['departureTime'] as String?,
      isImmediate: json['isImmediate'] as bool? ?? false,
      cost: _toDouble(json['cost']),
      currency: json['currency'] as String? ?? 'USD',
      distance: _toDouble(json['distance']),
      seatsAvailable: _toInt(json['seatsAvailable']),
    );
  }

  final String id;
  final String status;
  final String? fromLocationText;
  final String? toLocationText;
  final _LocationContext? fromContext;
  final _LocationContext? toContext;
  final String? departureTime;
  final bool isImmediate;
  final double? cost;
  final String currency;
  final double? distance;
  final int? seatsAvailable;

  bool get fromIsMapped => fromContext?.location != null;
  bool get toIsMapped => toContext?.location != null;

  String get fromLabel =>
      fromContext?.locationName ?? fromLocationText ?? 'Unknown pickup';
  String get toLabel =>
      toContext?.locationName ?? toLocationText ?? 'Unknown drop';

  String get fromDetail => _locationDetail(fromContext, fromLocationText);
  String get toDetail => _locationDetail(toContext, toLocationText);

  String get timeLabel {
    if (isImmediate) {
      return 'Needs ride now';
    }
    final parsed = DateTime.tryParse(departureTime ?? '')?.toLocal();
    if (parsed == null) {
      return 'Time not listed';
    }
    return _formatDateTime(parsed);
  }

  String? get costLabel {
    if (cost == null) {
      return null;
    }
    return '$currency ${_formatNumber(cost!)}';
  }

  String? get distanceLabel {
    if (distance == null) {
      return null;
    }
    return '${_formatNumber(distance!)} km';
  }

  String? get seatsLabel {
    if (seatsAvailable == null) {
      return null;
    }
    return '$seatsAvailable seat${seatsAvailable == 1 ? '' : 's'}';
  }
}

class _LocationContext {
  const _LocationContext({required this.locationName, required this.location});

  static _LocationContext? maybeFromJson(Object? value) {
    if (value is! Map<String, dynamic>) {
      return null;
    }
    return _LocationContext(
      locationName: value['locationName'] as String? ?? 'Mapped location',
      location: _GeoLocation.maybeFromJson(value['location']),
    );
  }

  final String locationName;
  final _GeoLocation? location;
}

class _GeoLocation {
  const _GeoLocation({required this.latitude, required this.longitude});

  static _GeoLocation? maybeFromJson(Object? value) {
    if (value is! Map<String, dynamic>) {
      return null;
    }
    final latitude = _toDouble(value['latitude']);
    final longitude = _toDouble(value['longitude']);
    if (latitude == null || longitude == null) {
      return null;
    }
    return _GeoLocation(latitude: latitude, longitude: longitude);
  }

  final double latitude;
  final double longitude;

  String get label =>
      '${latitude.toStringAsFixed(5)}, ${longitude.toStringAsFixed(5)}';
}

String _locationDetail(_LocationContext? context, String? rawText) {
  final location = context?.location;
  if (location != null) {
    return location.label;
  }
  if (context != null) {
    return 'Alias mapped, coordinates missing';
  }
  final text = rawText?.trim();
  if (text == null || text.isEmpty) {
    return 'Location not found in message';
  }
  return 'Needs group location mapping';
}

String _formatDateTime(DateTime value) {
  final hour =
      value.hour == 0 ? 12 : (value.hour > 12 ? value.hour - 12 : value.hour);
  final minute = value.minute.toString().padLeft(2, '0');
  final period = value.hour >= 12 ? 'PM' : 'AM';
  return '${value.month}/${value.day} $hour:$minute $period';
}

String _formatNumber(double value) {
  if (value == value.roundToDouble()) {
    return value.toStringAsFixed(0);
  }
  return value.toStringAsFixed(2);
}

double? _toDouble(Object? value) {
  if (value is num) {
    return value.toDouble();
  }
  if (value is String) {
    return double.tryParse(value);
  }
  return null;
}

int? _toInt(Object? value) {
  if (value is int) {
    return value;
  }
  if (value is num) {
    return value.toInt();
  }
  if (value is String) {
    return int.tryParse(value);
  }
  return null;
}
