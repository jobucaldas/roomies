import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../core/roles.dart';
import '../../models/models.dart';
import '../../state/app_state.dart';
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

class _HouseScreenState extends State<HouseScreen> {
  House? _house;
  List<HouseMember> _members = [];
  BalanceResponse? _balances;
  var _loading = true;
  String? _error;
  var _tab = 'Expenses';

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
    _load();
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

  void _selectTab(String tab) {
    setState(() => _tab = tab);
    if (tab == 'Balances') {
      _loadBalances();
    }
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

    return RoomiesPage(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (_error != null) RoomiesError(_error!),
          if (_loading)
            const Text('Loading house…')
          else if (_house != null) ...[
            RoomiesHeading(_house!.name),
            if (admin) _HouseEditor(house: _house!, onSaved: (h) => setState(() => _house = h)),
          ],
          RoomiesPrimaryButton(
            label: 'Back',
            onPressed: () => context.go('/dashboard'),
          ),
          RoomiesTabStrip(
            tabs: _tabs,
            selected: _tab,
            onSelected: _selectTab,
          ),
          RoomiesTabPanel(
            name: _tab,
            child: _buildPanel(userId, role, admin),
          ),
        ],
      ),
    );
  }

  Widget _buildPanel(String userId, HouseRole? role, bool admin) {
    switch (_tab) {
      case 'Expenses':
        return ExpensesSection(
          houseId: widget.houseId,
          role: role,
          userId: userId,
        );
      case 'Notes':
        return NotesSection(
          houseId: widget.houseId,
          role: role,
          userId: userId,
        );
      case 'Groceries':
        return GroceriesSection(
          houseId: widget.houseId,
          role: role,
          members: _members,
        );
      case 'Chores':
        return ChoresSection(houseId: widget.houseId, role: role);
      case 'Calendar':
        return CalendarSection(houseId: widget.houseId, role: role);
      case 'Chat':
        return ChatSection(
          houseId: widget.houseId,
          role: role,
          userId: userId,
        );
      case 'Balances':
        return BalancesSection(balances: _balances);
      case 'Notifications / Schedule':
        return NotificationsSection(houseId: widget.houseId, role: role);
      case 'Members':
        return MembersSection(
          houseId: widget.houseId,
          admin: admin,
          members: _members,
          onRefresh: _load,
        );
      default:
        return const SizedBox.shrink();
    }
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
    return RoomiesCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const RoomiesHeading('House settings', level: 2),
          RoomiesLabeledField(
            label: 'Name',
            child: TextField(controller: _name),
          ),
          RoomiesPrimaryButton(label: 'Save house', onPressed: _save),
          if (_status.isNotEmpty) Text(_status),
        ],
      ),
    );
  }
}
