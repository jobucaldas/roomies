import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../core/house_tabs.dart';
import '../../core/roles.dart';
import '../../models/models.dart';
import '../../state/app_state.dart';
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

class _HouseScreenState extends State<HouseScreen> {
  House? _house;
  List<HouseMember> _members = [];
  BalanceResponse? _balances;
  var _loading = true;
  String? _error;

  /// Active section key. The shell nav drives it through `?tab=`; there is no
  /// second in-page tab strip.
  String get _activeTab => HouseTabs.keyAt(HouseTabs.indexOf(widget.initialTab));

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void didUpdateWidget(covariant HouseScreen oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.houseId != widget.houseId) {
      _balances = null;
      _load();
    } else if (oldWidget.initialTab != widget.initialTab &&
        _activeTab == HouseTabs.balances) {
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
      await context.read<AppState>().refreshHouses();
      final house = await api.getHouse(widget.houseId);
      final members = await api.getMembers(widget.houseId);
      if (mounted) {
        setState(() {
          _house = house;
          _members = members;
          _loading = false;
        });
        if (_activeTab == HouseTabs.balances) _loadBalances();
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
    final houseId = widget.houseId;
    final balances = await context.read<AppState>().api.getBalances(houseId);
    if (mounted && houseId == widget.houseId) {
      setState(() => _balances = balances);
    }
  }

  Widget _section(String tab, HouseRole? role, String userId, bool admin) {
    switch (tab) {
      case HouseTabs.notes:
        return NotesSection(houseId: widget.houseId, role: role, userId: userId);
      case HouseTabs.groceries:
        return GroceriesSection(
          houseId: widget.houseId,
          role: role,
          members: _members,
        );
      case HouseTabs.chores:
        return ChoresSection(houseId: widget.houseId, role: role);
      case HouseTabs.calendar:
        return CalendarSection(houseId: widget.houseId, role: role);
      case HouseTabs.balances:
        return BalancesSection(balances: _balances);
      case HouseTabs.notifications:
        return NotificationsSection(
          houseId: widget.houseId,
          role: role,
          userId: userId,
        );
      case HouseTabs.members:
        return MembersSection(
          houseId: widget.houseId,
          admin: admin,
          members: _members,
          onRefresh: _load,
        );
      case HouseTabs.expenses:
      default:
        return ExpensesSection(
          houseId: widget.houseId,
          role: role,
          userId: userId,
        );
    }
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
    final activeTab = _activeTab;

    return AppShell(
      currentHouseId: widget.houseId,
      activeTab: activeTab,
      onHouseSelected: (id) {
        context.go('/house/$id?tab=$activeTab');
      },
      child: Align(
        alignment: Alignment.topCenter,
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 860),
          child: _loading
              ? Center(
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      const CircularProgressIndicator(strokeWidth: 2.5),
                      const SizedBox(height: 16),
                      Text(s.loading),
                    ],
                  ),
                )
              : SingleChildScrollView(
                  // Keyed by section so switching sections starts at the top
                  // and section state does not leak between houses.
                  key: ValueKey('${widget.houseId}/$activeTab'),
                  padding: const EdgeInsets.fromLTRB(20, 8, 20, 32),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: [
                      if (_error != null) RoomiesError(_error!),
                      if (_house != null)
                        KeyedSubtree(
                          key: ValueKey('${widget.houseId}/$activeTab/body'),
                          child: _section(activeTab, role, userId, admin),
                        ),
                    ],
                  ),
                ),
        ),
      ),
    );
  }
}
