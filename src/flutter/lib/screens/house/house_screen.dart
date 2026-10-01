import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../core/house_tabs.dart';
import '../../core/roles.dart';
import '../../models/models.dart';
import '../../state/app_state.dart';
import '../../theme/roomies_theme.dart';
import '../../widgets/app_shell.dart';
import '../../widgets/roomies_ui.dart';
import 'balances_section.dart';
import 'expenses_section.dart';
import 'household_sections.dart';
import 'members_section.dart';
import 'notes_section.dart';
import 'notifications_section.dart';

class HouseScreen extends StatefulWidget {
  const HouseScreen({
    super.key,
    required this.houseId,
    this.initialTab,
  });

  final String houseId;
  final String? initialTab;

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

  @override
  void initState() {
    super.initState();
    final initial = HouseTabs.indexOf(widget.initialTab);
    _tabController = TabController(
      length: HouseTabs.ordered.length,
      vsync: this,
      initialIndex: initial,
    );
    _tabController.addListener(_onTabChanged);
    _load();
  }

  @override
  void didUpdateWidget(covariant HouseScreen oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.houseId != widget.houseId) {
      _load();
    }
    if (oldWidget.initialTab != widget.initialTab) {
      final next = HouseTabs.indexOf(widget.initialTab);
      if (_tabController.index != next) {
        _tabController.animateTo(next);
      }
    }
  }

  @override
  void dispose() {
    _tabController.removeListener(_onTabChanged);
    _tabController.dispose();
    super.dispose();
  }

  void _onTabChanged() {
    if (_tabController.indexIsChanging) return;
    setState(() {});
    final tabs = context.read<AppState>().strings.houseTabs;
    if (_tabController.index < tabs.length &&
        tabs[_tabController.index] ==
            context.read<AppState>().strings.tabBalances) {
      _loadBalances();
    }
    final key = HouseTabs.keyAt(_tabController.index);
    final uri = GoRouterState.of(context).uri;
    final current = uri.queryParameters['tab'];
    if (current != key) {
      context.go('/house/${widget.houseId}?tab=$key');
    }
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final api = context.read<AppState>().api;
      await context.read<AppState>().refreshHouses();
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
    final s = app.strings;
    if (!app.api.isAuthenticated) {
      WidgetsBinding.instance.addPostFrameCallback((_) => context.go('/'));
      return RoomiesPage(child: Text(s.redirectingToLogin));
    }
    final userId = app.user?.id ?? '';
    final role = _currentRole(userId);
    final admin = _admin(role);
    final tabs = s.houseTabs;
    final activeTab = HouseTabs.keyAt(_tabController.index);

    return AppShell(
      currentHouseId: widget.houseId,
      activeTab: activeTab,
      title: _house?.name,
      showBrand: false,
      onHouseSelected: (id) {
        context.go('/house/$id?tab=$activeTab');
      },
      actions: [
        TextButton(
          onPressed: () => context.go('/dashboard'),
          child: Text(s.back),
        ),
      ],
      child: Padding(
        padding: const EdgeInsets.fromLTRB(20, 8, 20, 12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            if (_error != null) RoomiesError(_error!),
            if (_loading)
              Expanded(
                child: Center(
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      const CircularProgressIndicator(),
                      const SizedBox(height: 16),
                      Text(s.loading),
                    ],
                  ),
                ),
              )
            else if (_house != null) ...[
              Material(
                color:
                    RoomiesPalette.of(context).surface.withValues(alpha: 0.72),
                borderRadius: BorderRadius.circular(16),
                child: SingleChildScrollView(
                  scrollDirection: Axis.horizontal,
                  padding:
                      const EdgeInsets.symmetric(horizontal: 4, vertical: 4),
                  child: Row(
                    children: [
                      for (var i = 0; i < tabs.length; i++)
                        Padding(
                          padding: const EdgeInsets.symmetric(horizontal: 4),
                          child: TextButton(
                            onPressed: () => _tabController.animateTo(i),
                            style: TextButton.styleFrom(
                              minimumSize: const Size(48, 48),
                              padding: const EdgeInsets.symmetric(
                                horizontal: 14,
                                vertical: 12,
                              ),
                              foregroundColor: _tabController.index == i
                                  ? RoomiesPalette.of(context).tealDeep
                                  : RoomiesPalette.of(context).inkMuted,
                            ),
                            child: Text(tabs[i]),
                          ),
                        ),
                    ],
                  ),
                ),
              ),
              const SizedBox(height: 12),
              Expanded(
                child: TabBarView(
                  controller: _tabController,
                  physics: const NeverScrollableScrollPhysics(),
                  children: [
                    _panel(ExpensesSection(
                      houseId: widget.houseId,
                      role: role,
                      userId: userId,
                    )),
                    _panel(NotesSection(
                      houseId: widget.houseId,
                      role: role,
                      userId: userId,
                    )),
                    _panel(GroceriesSection(
                      houseId: widget.houseId,
                      role: role,
                      members: _members,
                    )),
                    _panel(ChoresSection(
                        houseId: widget.houseId, role: role)),
                    _panel(CalendarSection(
                        houseId: widget.houseId, role: role)),
                    _panel(BalancesSection(balances: _balances)),
                    _panel(NotificationsSection(
                      houseId: widget.houseId,
                      role: role,
                      userId: userId,
                    )),
                    _panel(MembersSection(
                      houseId: widget.houseId,
                      admin: admin,
                      members: _members,
                      onRefresh: _load,
                    )),
                  ],
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }
}

Widget _panel(Widget child) {
  return SingleChildScrollView(
    padding: const EdgeInsets.only(bottom: 24),
    child: child,
  );
}
