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
  var _showCreate = false;
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
      if (!mounted) return;
      final empty = context.read<AppState>().houses.isEmpty;
      setState(() {
        _loading = false;
        // Empty state: create form is the job. Otherwise keep it tucked away.
        _showCreate = empty;
      });
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
      final s = context.read<AppState>().strings;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(s.houseSetAsDefault(house.name)),
          action: SnackBarAction(
            label: s.viewHouse,
            onPressed: () => context.go('/house/${house.id}'),
          ),
        ),
      );
      setState(() {
        _creating = false;
        _showCreate = false;
      });
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
              Text(
                user == null ? s.welcomeGuest : s.welcomeBack(user.name),
                style: Theme.of(context).textTheme.headlineMedium,
              ),
              const SizedBox(height: 20),
              if (_error != null) RoomiesError(_error!),
              if (_loading)
                const Padding(
                  padding: EdgeInsets.symmetric(vertical: 48),
                  child: Center(
                    child: CircularProgressIndicator(strokeWidth: 2.5),
                  ),
                )
              else ...[
                // Houses first when they exist — that is the primary job.
                if (houses.isNotEmpty) ...[
                  Row(
                    children: [
                      Expanded(
                        child: Text(
                          s.yourHouses,
                          style: Theme.of(context).textTheme.titleLarge,
                        ),
                      ),
                      TextButton.icon(
                        onPressed: () =>
                            setState(() => _showCreate = !_showCreate),
                        icon: Icon(
                          _showCreate ? Icons.close : Icons.add,
                          size: 18,
                        ),
                        label: Text(s.createNewHouse),
                      ),
                    ],
                  ),
                  const SizedBox(height: 8),
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
                ] else
                  Padding(
                    padding: const EdgeInsets.only(bottom: 12),
                    child: Text(
                      s.noHousesYet,
                      style: Theme.of(context).textTheme.bodyLarge,
                    ),
                  ),
                if (_showCreate) ...[
                  const SizedBox(height: 8),
                  _CreateHouseBlock(
                    controller: _houseName,
                    creating: _creating,
                    title: s.createNewHouse,
                    hint: s.houseName,
                    buttonLabel: s.createHouse,
                    onSubmit: _createHouse,
                  ),
                ],
              ],
            ],
          ),
        ),
      ),
    );
  }
}

class _CreateHouseBlock extends StatelessWidget {
  const _CreateHouseBlock({
    required this.controller,
    required this.creating,
    required this.title,
    required this.hint,
    required this.buttonLabel,
    required this.onSubmit,
  });

  final TextEditingController controller;
  final bool creating;
  final String title;
  final String hint;
  final String buttonLabel;
  final VoidCallback onSubmit;

  @override
  Widget build(BuildContext context) {
    return RoomiesCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(title, style: Theme.of(context).textTheme.titleMedium),
          const SizedBox(height: 10),
          TextField(
            controller: controller,
            decoration: InputDecoration(hintText: hint),
            textInputAction: TextInputAction.done,
            onSubmitted: (_) => onSubmit(),
          ),
          const SizedBox(height: 8),
          Align(
            alignment: Alignment.centerRight,
            child: FilledButton(
              onPressed: creating ? null : onSubmit,
              child: Text(buttonLabel),
            ),
          ),
        ],
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
    return Material(
      color: RoomiesColors.surface,
      borderRadius: BorderRadius.circular(16),
      child: InkWell(
        borderRadius: BorderRadius.circular(16),
        onTap: onOpen,
        child: Container(
          margin: const EdgeInsets.symmetric(vertical: 6),
          padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 14),
          decoration: BoxDecoration(
            borderRadius: BorderRadius.circular(16),
            border: Border.all(color: RoomiesColors.line),
          ),
          child: Row(
            children: [
              Container(
                width: 44,
                height: 44,
                decoration: BoxDecoration(
                  color: RoomiesColors.tealSoft,
                  borderRadius: BorderRadius.circular(12),
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
              const SizedBox(width: 14),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      house.name,
                      style: Theme.of(context).textTheme.titleMedium,
                    ),
                    if (isDefault)
                      Text(
                        defaultLabel,
                        style: Theme.of(context).textTheme.bodySmall?.copyWith(
                              color: RoomiesColors.tealDeep,
                              fontWeight: FontWeight.w600,
                            ),
                      ),
                  ],
                ),
              ),
              if (!isDefault)
                IconButton(
                  tooltip: setDefaultLabel,
                  onPressed: onSetDefault,
                  icon: const Icon(Icons.star_outline_rounded),
                  color: RoomiesColors.inkMuted,
                ),
              Icon(
                Icons.chevron_right_rounded,
                color: RoomiesColors.inkMuted,
              ),
            ],
          ),
        ),
      ),
    );
  }
}
