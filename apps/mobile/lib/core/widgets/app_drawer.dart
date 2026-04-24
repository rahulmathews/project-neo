import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:supabase_flutter/supabase_flutter.dart';

class ScaffoldWithDrawer extends StatelessWidget {
  const ScaffoldWithDrawer({
    required this.location,
    required this.child,
    super.key,
  });

  final String location;
  final Widget child;

  String get _title => switch (location) {
    '/home' => 'Home',
    '/rides' => 'Available Rides',
    '/matches' => 'My Matches',
    '/profile' => 'Profile',
    '/settings' => 'Settings',
    _ => 'Neo',
  };

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: Text(_title)),
      drawer: const AppDrawer(),
      body: child,
    );
  }
}

class AppDrawer extends StatelessWidget {
  const AppDrawer({super.key});

  @override
  Widget build(BuildContext context) {
    final email = Supabase.instance.client.auth.currentUser?.email ?? '';

    return Drawer(
      child: Column(
        children: [
          DrawerHeader(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              mainAxisAlignment: MainAxisAlignment.end,
              children: [
                Text(
                  'Neo',
                  style: Theme.of(
                    context,
                  ).textTheme.titleLarge?.copyWith(fontWeight: FontWeight.bold),
                ),
                const SizedBox(height: 4),
                Text(
                  email,
                  style: Theme.of(context).textTheme.bodySmall?.copyWith(
                    color: Theme.of(context).colorScheme.onSurfaceVariant,
                  ),
                ),
              ],
            ),
          ),
          Expanded(
            child: ListView(
              padding: EdgeInsets.zero,
              children: [
                ExpansionTile(
                  initiallyExpanded: true,
                  title: const Text('Rides'),
                  children: [
                    _DrawerItem(label: 'Home', route: '/home'),
                    _DrawerItem(label: 'Available Rides', route: '/rides'),
                    _DrawerItem(label: 'My Matches', route: '/matches'),
                  ],
                ),
                ExpansionTile(
                  initiallyExpanded: true,
                  title: const Text('Account'),
                  children: [
                    _DrawerItem(label: 'Profile', route: '/profile'),
                    _DrawerItem(label: 'Settings', route: '/settings'),
                  ],
                ),
              ],
            ),
          ),
          const Divider(height: 1),
          ListTile(
            leading: const Icon(Icons.logout),
            title: const Text('Sign Out'),
            onTap: () async {
              Navigator.of(context).pop();
              await Supabase.instance.client.auth.signOut();
            },
          ),
        ],
      ),
    );
  }
}

class _DrawerItem extends StatelessWidget {
  const _DrawerItem({required this.label, required this.route});

  final String label;
  final String route;

  @override
  Widget build(BuildContext context) {
    return ListTile(
      contentPadding: const EdgeInsets.only(left: 32),
      title: Text(label),
      onTap: () {
        context.go(route);
        Navigator.of(context).pop();
      },
    );
  }
}
