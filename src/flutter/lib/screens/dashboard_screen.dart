import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../models/models.dart';
import '../state/app_state.dart';
import '../theme/roomies_theme.dart';
import '../widgets/roomies_ui.dart';

class DashboardScreen extends StatefulWidget {
  const DashboardScreen({super.key});

  @override
  State<DashboardScreen> createState() => _DashboardScreenState();
}

class _DashboardScreenState extends State<DashboardScreen> {
  final _houseName = TextEditingController();
  List<House> _houses = [];
  var _loading = true;
  String? _error;

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    _houseName.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final houses = await context.read<AppState>().api.getHouses();
      if (mounted) {
        setState(() {
          _houses = houses;
          _loading = false;
        });
      }
    } catch (error) {
      if (mounted) {
        setState(() {
          _error = error.toString();
          _loading = false;
        });
      }
    }
  }

  Future<void> _createHouse() async {
    final name = _houseName.text.trim();
    if (name.isEmpty) return;
    await context.read<AppState>().api.createHouse(name);
    _houseName.clear();
    await _load();
  }

  Future<void> _logout() async {
    await context.read<AppState>().logout();
    if (mounted) context.go('/');
  }

  @override
  Widget build(BuildContext context) {
    final user = context.watch<AppState>().user;
    return RoomiesPage(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    const RoomiesBrandMark(compact: true),
                    const RoomiesHeading('Dashboard', level: 2),
                    Text(
                      user == null
                          ? 'Welcome to Roomies!'
                          : 'Welcome back, ${user.name}.',
                      style: Theme.of(context).textTheme.bodyLarge,
                    ),
                  ],
                ),
              ),
              TextButton(
                onPressed: _logout,
                child: const Text('Logout'),
              ),
            ],
          ),
          const SizedBox(height: 8),
          const RoomiesHeading('Create New House', level: 2),
          RoomiesCard(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  'Start a house for expenses, chores, and shared notes.',
                  style: Theme.of(context).textTheme.bodyMedium,
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: _houseName,
                  decoration: const InputDecoration(hintText: 'House name'),
                ),
                RoomiesPrimaryButton(
                  label: 'Create House',
                  onPressed: _createHouse,
                ),
              ],
            ),
          ),
          const RoomiesHeading('Your Houses', level: 2),
          if (_error != null) RoomiesError(_error!),
          if (_loading)
            const Padding(
              padding: EdgeInsets.symmetric(vertical: 24),
              child: Center(child: CircularProgressIndicator()),
            )
          else if (_houses.isEmpty)
            RoomiesCard(
              child: Text(
                'No houses yet. Create one above!',
                style: Theme.of(context).textTheme.bodyLarge,
              ),
            )
          else
            ..._houses.map(
              (house) => RoomiesCard(
                child: Row(
                  children: [
                    Container(
                      width: 48,
                      height: 48,
                      decoration: BoxDecoration(
                        color: RoomiesColors.tealSoft,
                        borderRadius: BorderRadius.circular(14),
                      ),
                      alignment: Alignment.center,
                      child: Text(
                        house.name.isEmpty
                            ? '?'
                            : house.name.substring(0, 1).toUpperCase(),
                        style: Theme.of(context).textTheme.titleLarge?.copyWith(
                              color: RoomiesColors.tealDeep,
                            ),
                      ),
                    ),
                    const SizedBox(width: 16),
                    Expanded(
                      child: RoomiesHeading(house.name, level: 3),
                    ),
                    FilledButton(
                      onPressed: () => context.go('/house/${house.id}'),
                      child: const Text('View House'),
                    ),
                  ],
                ),
              ),
            ),
        ],
      ),
    );
  }
}
