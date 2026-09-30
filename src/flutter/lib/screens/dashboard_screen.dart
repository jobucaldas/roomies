import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../models/models.dart';
import '../state/app_state.dart';
import '../theme/roomies_theme.dart';
import '../widgets/app_shell.dart';
import '../widgets/roomies_ui.dart';

class DashboardScreen extends StatefulWidget {
  const DashboardScreen({super.key});

  @override
  State<DashboardScreen> createState() => _DashboardScreenState();
}

class _DashboardScreenState extends State<DashboardScreen> {
  final _houseName = TextEditingController();
  var _loading = true;
  var _creating = false;
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
      await context.read<AppState>().refreshHouses();
      if (mounted) setState(() => _loading = false);
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
    setState(() => _creating = true);
    try {
      final house =
          await context.read<AppState>().createHouseAndSetDefault(name);
      _houseName.clear();
      if (!mounted) return;
      // Stay on Dashboard after create (default house is already set).
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(
            context.read<AppState>().strings.setAsDefault,
          ),
          action: SnackBarAction(
            label: context.read<AppState>().strings.viewHouse,
            onPressed: () => context.go('/house/${house.id}'),
          ),
        ),
      );
      setState(() => _creating = false);
    } catch (error) {
      if (mounted) {
        setState(() {
          _error = error.toString();
          _creating = false;
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final app = context.watch<AppState>();
    final s = app.strings;
    final user = app.user;
    final houses = app.houses;

    return AppShell(
      title: s.dashboard,
      showBrand: false,
      child: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 720),
          child: ListView(
            padding: const EdgeInsets.fromLTRB(20, 8, 20, 32),
            children: [
              RoomiesHeading(s.dashboard, level: 2),
              Text(
                user == null ? s.welcomeGuest : s.welcomeBack(user.name),
                style: Theme.of(context).textTheme.bodyLarge,
              ),
              const SizedBox(height: 16),
              RoomiesHeading(s.createNewHouse, level: 2),
              RoomiesCard(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      s.createHouseHint,
                      style: Theme.of(context).textTheme.bodyMedium,
                    ),
                    const SizedBox(height: 12),
                    TextField(
                      controller: _houseName,
                      decoration: InputDecoration(hintText: s.houseName),
                    ),
                    RoomiesPrimaryButton(
                      label: s.createHouse,
                      onPressed: _creating ? null : _createHouse,
                      enabled: !_creating,
                    ),
                  ],
                ),
              ),
              RoomiesHeading(s.yourHouses, level: 2),
              if (_error != null) RoomiesError(_error!),
              if (_loading)
                Padding(
                  padding: const EdgeInsets.symmetric(vertical: 24),
                  child: Center(
                    child: Column(
                      children: [
                        const CircularProgressIndicator(),
                        const SizedBox(height: 12),
                        Text(s.loading),
                      ],
                    ),
                  ),
                )
              else if (houses.isEmpty)
                RoomiesCard(
                  child: Text(
                    s.noHousesYet,
                    style: Theme.of(context).textTheme.bodyLarge,
                  ),
                )
              else
                ...houses.map(
                  (house) => _HouseRow(
                    house: house,
                    isDefault: house.id == app.defaultHouseId,
                    defaultLabel: s.defaultBadge,
                    viewLabel: s.viewHouse,
                    setDefaultLabel: s.setAsDefault,
                    onOpen: () => context.go('/house/${house.id}'),
                    onSetDefault: () => app.setDefaultHouseId(house.id),
                  ),
                ),
            ],
          ),
        ),
      ),
    );
  }
}

class _HouseRow extends StatelessWidget {
  const _HouseRow({
    required this.house,
    required this.isDefault,
    required this.defaultLabel,
    required this.viewLabel,
    required this.setDefaultLabel,
    required this.onOpen,
    required this.onSetDefault,
  });

  final House house;
  final bool isDefault;
  final String defaultLabel;
  final String viewLabel;
  final String setDefaultLabel;
  final VoidCallback onOpen;
  final VoidCallback onSetDefault;

  @override
  Widget build(BuildContext context) {
    return RoomiesCard(
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
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                RoomiesHeading(house.name, level: 3),
                if (isDefault)
                  Text(
                    defaultLabel,
                    style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                          color: RoomiesColors.tealDeep,
                          fontWeight: FontWeight.w600,
                        ),
                  ),
              ],
            ),
          ),
          if (!isDefault)
            TextButton(
              onPressed: onSetDefault,
              child: Text(setDefaultLabel),
            ),
          FilledButton(
            onPressed: onOpen,
            child: Text(viewLabel),
          ),
        ],
      ),
    );
  }
}
