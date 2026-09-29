import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../models/models.dart';
import '../state/app_state.dart';
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
    return RoomiesPage(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const RoomiesHeading('Dashboard', level: 2),
          const Text('Welcome to Roomies!'),
          RoomiesPrimaryButton(label: 'Logout', onPressed: _logout),
          const RoomiesHeading('Create New House', level: 2),
          RoomiesCard(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
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
            const Text('Loading...')
          else if (_houses.isEmpty)
            const Text('No houses yet. Create one above!')
          else
            ..._houses.map(
              (house) => RoomiesCard(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    RoomiesHeading(house.name, level: 3),
                    TextButton(
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
