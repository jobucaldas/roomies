import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../core/roles.dart';
import '../../models/models.dart';
import '../../state/app_state.dart';
import '../../theme/roomies_theme.dart';
import '../../widgets/house_tab_a11y.dart';
import '../../widgets/roomies_ui.dart';
import 'balances_section.dart';
import 'expenses_section.dart';
import 'household_sections.dart';
import 'members_section.dart';
import 'notes_section.dart';
import 'notifications_section.dart';

class HouseScreen extends StatefulWidget {
  const HouseScreen({super.key, required this.houseId});

  final String houseId;

  @override
  State<HouseScreen> createState() => _HouseScreenState();
}

class _HouseScreenState extends State<HouseScreen>
    with SingleTickerProviderStateMixin {
  House? _house;
  List<HouseMember> _members = [];
  BalanceResponse? _balances;
  var _loading = true;
  String? _error;
  late TabController _tabController;

  static const _tabs = [
    'Expenses',
    'Notes',
    'Groceries',
    'Chores',
    'Calendar',
    'Chat',
    'Balances',
    'Notifications / Schedule',
    'Members',
  ];

  @override
  void initState() {
    super.initState();
    _tabController = TabController(length: _tabs.length, vsync: this);
    _tabController.addListener(_onTabChanged);
    if (kIsWeb) {
      installHouseTabA11y(_tabs);
      updateHouseTabA11ySelection(_tabController.index);
    }
    _load();
  }

  @override
  void dispose() {
    _tabController.removeListener(_onTabChanged);
    _tabController.dispose();
    if (kIsWeb) uninstallHouseTabA11y();
    super.dispose();
  }

  void _onTabChanged() {
    if (_tabController.indexIsChanging) return;
    if (kIsWeb) updateHouseTabA11ySelection(_tabController.index);
    if (_tabs[_tabController.index] == 'Balances') {
      _loadBalances();
    }
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final api = context.read<AppState>().api;
      final house = await api.getHouse(widget.houseId);
      final members = await api.getMembers(widget.houseId);
      if (mounted) {
        setState(() {
          _house = house;
          _members = members;
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

  HouseRole? _currentRole(String userId) {
    HouseMember? member;
    for (final m in _members) {
      if (m.userId == userId) {
        member = m;
        break;
      }
    }
    return member == null ? null : parseRole(member.role);
  }

  bool _admin(HouseRole? role) => role != null && canManage(role);

  Future<void> _loadBalances() async {
    final balances =
        await context.read<AppState>().api.getBalances(widget.houseId);
    if (mounted) setState(() => _balances = balances);
  }

  @override
  Widget build(BuildContext context) {
    final app = context.watch<AppState>();
    if (!app.api.isAuthenticated) {
      WidgetsBinding.instance.addPostFrameCallback((_) => context.go('/'));
      return const RoomiesPage(child: Text('Redirecting to login…'));
    }
    final userId = app.user?.id ?? '';
    final role = _currentRole(userId);
    final admin = _admin(role);

    return Scaffold(
      backgroundColor: Colors.transparent,
      body: RoomiesAtmosphere(
        child: SafeArea(
          child: Center(
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 960),
              child: Padding(
                padding: const EdgeInsets.fromLTRB(20, 16, 20, 12),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    if (_error != null) RoomiesError(_error!),
                    if (_loading)
                      const Expanded(
                        child: Center(child: CircularProgressIndicator()),
                      )
                    else if (_house != null) ...[
                      Row(
                        children: [
                          Expanded(child: RoomiesHeading(_house!.name)),
                          TextButton(
                            onPressed: () => context.go('/dashboard'),
                            child: const Text('Back'),
                          ),
                        ],
                      ),
                      if (admin)
                        _HouseEditor(
                          house: _house!,
                          onSaved: (h) => setState(() => _house = h),
                        ),
                      const SizedBox(height: 8),
                      Material(
                        color: RoomiesColors.surface.withValues(alpha: 0.72),
                        borderRadius: BorderRadius.circular(16),
                        child: TabBar(
                          controller: _tabController,
                          isScrollable: true,
                          tabs: [for (final tab in _tabs) Tab(text: tab)],
                        ),
                      ),
                      const SizedBox(height: 8),
                      Expanded(
                        child: TabBarView(
                          controller: _tabController,
                          children: [
                            RoomiesTabPanel(
                              name: 'Expenses',
                              child: ExpensesSection(
                                houseId: widget.houseId,
                                role: role,
                                userId: userId,
                              ),
                            ),
                            RoomiesTabPanel(
                              name: 'Notes',
                              child: NotesSection(
                                houseId: widget.houseId,
                                role: role,
                                userId: userId,
                              ),
                            ),
                            RoomiesTabPanel(
                              name: 'Groceries',
                              child: GroceriesSection(
                                houseId: widget.houseId,
                                role: role,
                                members: _members,
                              ),
                            ),
                            RoomiesTabPanel(
                              name: 'Chores',
                              child: ChoresSection(
                                  houseId: widget.houseId, role: role),
                            ),
                            RoomiesTabPanel(
                              name: 'Calendar',
                              child: CalendarSection(
                                  houseId: widget.houseId, role: role),
                            ),
                            RoomiesTabPanel(
                              name: 'Chat',
                              child: ChatSection(
                                houseId: widget.houseId,
                                role: role,
                                userId: userId,
                              ),
                            ),
                            RoomiesTabPanel(
                              name: 'Balances',
                              child: BalancesSection(balances: _balances),
                            ),
                            RoomiesTabPanel(
                              name: 'Notifications / Schedule',
                              child: NotificationsSection(
                                houseId: widget.houseId,
                                role: role,
                                userId: userId,
                              ),
                            ),
                            RoomiesTabPanel(
                              name: 'Members',
                              child: MembersSection(
                                houseId: widget.houseId,
                                admin: admin,
                                members: _members,
                                onRefresh: _load,
                              ),
                            ),
                          ],
                        ),
                      ),
                    ],
                  ],
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}

class _HouseEditor extends StatefulWidget {
  const _HouseEditor({required this.house, required this.onSaved});

  final House house;
  final ValueChanged<House> onSaved;

  @override
  State<_HouseEditor> createState() => _HouseEditorState();
}

class _HouseEditorState extends State<_HouseEditor> {
  late final TextEditingController _name;
  String _status = '';

  @override
  void initState() {
    super.initState();
    _name = TextEditingController(text: widget.house.name);
  }

  @override
  void dispose() {
    _name.dispose();
    super.dispose();
  }

  Future<void> _save() async {
    if (_name.text.trim().isEmpty) {
      setState(() => _status = 'House name is required');
      return;
    }
    final updated = await context.read<AppState>().api.updateHouse(
          widget.house.id,
          _name.text.trim(),
        );
    setState(() => _status = 'House updated.');
    widget.onSaved(updated);
  }

  @override
  Widget build(BuildContext context) {
    return Card(
      margin: const EdgeInsets.symmetric(vertical: 10),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const RoomiesHeading('House settings', level: 2),
            TextFormField(
              controller: _name,
              decoration: const InputDecoration(labelText: 'Name'),
            ),
            FilledButton(onPressed: _save, child: const Text('Save house')),
            if (_status.isNotEmpty) Text(_status),
          ],
        ),
      ),
    );
  }
}
