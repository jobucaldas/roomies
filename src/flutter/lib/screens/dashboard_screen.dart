import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../core/datetime_format.dart';
import '../core/money.dart';
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
  final _houseNameFocus = FocusNode(debugLabel: 'create-house-name');
  var _loading = true;
  var _summaryLoading = false;
  var _creating = false;
  var _showCreate = false;
  String? _error;
  String? _summaryError;
  String? _focusHouseId;
  List<Expense> _monthExpenses = [];
  List<Note> _notes = [];
  List<_DashEvent> _events = [];

  /// Bumps on every summary fetch. A response applies only if it is still the
  /// latest, so a slow house cannot overwrite the one the user switched to.
  int _summaryGeneration = 0;

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    _houseNameFocus.dispose();
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
      final app = context.read<AppState>();
      final empty = app.houses.isEmpty;
      final focus = app.defaultHouseId ??
          (app.houses.isNotEmpty ? app.houses.first.id : null);
      setState(() {
        _loading = false;
        _showCreate = empty;
        _focusHouseId = focus;
      });
      if (focus != null) {
        await _loadSummary(focus);
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

  Future<void> _loadSummary(String houseId) async {
    final generation = ++_summaryGeneration;
    setState(() {
      _summaryLoading = true;
      _summaryError = null;
      _focusHouseId = houseId;
    });
    try {
      final api = context.read<AppState>().api;
      final results = await Future.wait([
        api.getExpenses(houseId),
        api.getNotes(houseId),
        api.getCalendar(houseId),
        api.getScheduledEvents(houseId),
      ]);
      if (!mounted || generation != _summaryGeneration) return;
      final expenses = results[0] as List<Expense>;
      final notes = results[1] as List<Note>;
      final calendar = results[2] as List<CalendarEvent>;
      final scheduled = results[3] as List<ScheduledHouseEvent>;
      final now = DateTime.now();
      final monthPrefix =
          '${now.year.toString().padLeft(4, '0')}-${now.month.toString().padLeft(2, '0')}';
      final monthExpenses =
          expenses.where((e) => e.date.startsWith(monthPrefix)).toList();
      final events = <_DashEvent>[
        ...calendar.map(
          (e) => _DashEvent(
            title: e.title,
            when: e.startLocal,
            kind: 'calendar',
          ),
        ),
        ...scheduled.where((e) => e.enabled).map(
              (e) => _DashEvent(
                title: e.title,
                when: e.nextOccurrenceAt ?? e.dtstartLocal,
                kind: 'schedule',
              ),
            ),
      ]..sort((a, b) => a.when.compareTo(b.when));
      setState(() {
        _monthExpenses = monthExpenses;
        _notes = notes.take(5).toList();
        _events = events.take(5).toList();
        _summaryLoading = false;
      });
    } catch (error) {
      if (mounted && generation == _summaryGeneration) {
        setState(() {
          _summaryError = error.toString();
          _summaryLoading = false;
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
        _focusHouseId = house.id;
      });
      await _loadSummary(house.id);
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
    House? focusHouse;
    for (final house in houses) {
      if (house.id == _focusHouseId) {
        focusHouse = house;
        break;
      }
    }
    focusHouse ??= houses.isNotEmpty ? houses.first : null;
    final monthTotal =
        _monthExpenses.fold<double>(0, (sum, e) => sum + e.amount);

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
                if (houses.isEmpty)
                  Padding(
                    padding: const EdgeInsets.only(bottom: 12),
                    child: Text(
                      s.noHousesYet,
                      style: Theme.of(context).textTheme.bodyLarge,
                    ),
                  )
                else ...[
                  if (focusHouse != null) ...[
                    Builder(
                      builder: (context) {
                        final house = focusHouse!;
                        return Column(
                          crossAxisAlignment: CrossAxisAlignment.stretch,
                          children: [
                            _SectionHeader(
                              title: s.focusHouse,
                              trailing: FilledButton.tonal(
                                onPressed: () =>
                                    context.go('/house/${house.id}'),
                                style: FilledButton.styleFrom(
                                  minimumSize: const Size(48, 44),
                                  padding: const EdgeInsets.symmetric(
                                    horizontal: 14,
                                    vertical: 10,
                                  ),
                                ),
                                child: Text(s.openHouse),
                              ),
                            ),
                            Text(
                              house.name,
                              style: Theme.of(context).textTheme.titleLarge,
                            ),
                            const SizedBox(height: 16),
                            if (_summaryError != null) ...[
                              RoomiesError(_summaryError!),
                              TextButton(
                                onPressed: () => _loadSummary(house.id),
                                child: Text(s.retry),
                              ),
                              const SizedBox(height: 12),
                            ],
                            if (_summaryLoading)
                              const Padding(
                                padding: EdgeInsets.symmetric(vertical: 24),
                                child: Center(
                                  child: CircularProgressIndicator(
                                      strokeWidth: 2.5),
                                ),
                              )
                            else ...[
                              _SummaryCard(
                                title: s.recentEvents,
                                child: _events.isEmpty
                                    ? Text(s.noRecentEvents)
                                    : Column(
                                        children: [
                                          for (final event in _events)
                                            _SummaryRow(
                                              title: event.title,
                                              subtitle: formatDisplayDateTime(
                                                event.when,
                                                localeCode: app.localeCode,
                                              ),
                                            ),
                                        ],
                                      ),
                              ),
                              const SizedBox(height: 12),
                              _SummaryCard(
                                title: s.monthMoney,
                                child: _monthExpenses.isEmpty
                                    ? Text(s.noMonthExpenses)
                                    : Column(
                                        crossAxisAlignment:
                                            CrossAxisAlignment.start,
                                        children: [
                                          Text(
                                            s.monthSpendTotal(
                                              formatMoney(
                                                monthTotal,
                                                localeCode: app.localeCode,
                                              ),
                                            ),
                                            style: Theme.of(context)
                                                .textTheme
                                                .titleMedium
                                                ?.copyWith(
                                                  color:
                                                      RoomiesPalette.of(context).tealDeep,
                                                  fontWeight: FontWeight.w700,
                                                ),
                                          ),
                                          const SizedBox(height: 8),
                                          for (final expense
                                              in _monthExpenses.take(4))
                                            _SummaryRow(
                                              title: expense.description,
                                              subtitle:
                                                  '${formatDisplayDate(expense.date, localeCode: app.localeCode)} · ${formatMoney(expense.amount, localeCode: app.localeCode)}',
                                            ),
                                        ],
                                      ),
                              ),
                              const SizedBox(height: 12),
                              _SummaryCard(
                                title: s.recentNotes,
                                child: _notes.isEmpty
                                    ? Text(s.noRecentNotes)
                                    : Column(
                                        children: [
                                          for (final note in _notes)
                                            _SummaryRow(
                                              title: note.title,
                                              subtitle: note.content,
                                            ),
                                        ],
                                      ),
                              ),
                              const SizedBox(height: 24),
                            ],
                          ],
                        );
                      },
                    ),
                  ],
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
                  const SizedBox(height: 12),
                  ...houses.map(
                    (house) => _HouseRow(
                      house: house,
                      isDefault: house.id == app.defaultHouseId,
                      isFocus: house.id == (_focusHouseId ?? focusHouse?.id),
                      defaultLabel: s.defaultBadge,
                      viewLabel: s.viewHouse,
                      setDefaultLabel: s.setAsDefault,
                      onOpen: () => context.go('/house/${house.id}'),
                      onFocus: () => _loadSummary(house.id),
                      onSetDefault: () => app.setDefaultHouseId(house.id),
                    ),
                  ),
                ],
                if (_showCreate) ...[
                  const SizedBox(height: 16),
                  _CreateHouseBlock(
                    controller: _houseName,
                    focusNode: _houseNameFocus,
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

class _DashEvent {
  const _DashEvent({
    required this.title,
    required this.when,
    required this.kind,
  });

  final String title;
  final String when;
  final String kind;
}

class _SectionHeader extends StatelessWidget {
  const _SectionHeader({required this.title, this.trailing});

  final String title;
  final Widget? trailing;

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Expanded(
          child: Text(title, style: Theme.of(context).textTheme.titleMedium),
        ),
        if (trailing != null) trailing!,
      ],
    );
  }
}

class _SummaryCard extends StatelessWidget {
  const _SummaryCard({required this.title, required this.child});

  final String title;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    return RoomiesCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(title, style: Theme.of(context).textTheme.titleMedium),
          const SizedBox(height: 10),
          child,
        ],
      ),
    );
  }
}

class _SummaryRow extends StatelessWidget {
  const _SummaryRow({required this.title, required this.subtitle});

  final String title;
  final String subtitle;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 6),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(title, style: Theme.of(context).textTheme.titleSmall),
          Text(
            subtitle,
            maxLines: 2,
            overflow: TextOverflow.ellipsis,
            style: Theme.of(context).textTheme.bodySmall?.copyWith(
                  color: RoomiesPalette.of(context).inkMuted,
                ),
          ),
        ],
      ),
    );
  }
}

class _CreateHouseBlock extends StatelessWidget {
  const _CreateHouseBlock({
    required this.controller,
    required this.focusNode,
    required this.creating,
    required this.title,
    required this.hint,
    required this.buttonLabel,
    required this.onSubmit,
  });

  final TextEditingController controller;
  final FocusNode focusNode;
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
          const SizedBox(height: 12),
          // Stable key + FocusNode: AppState/GoRouter/MediaQuery rebuilds must
          // not remount this field and drop soft-keyboard focus on mobile web.
          TextField(
            key: const ValueKey('create-house-name'),
            controller: controller,
            focusNode: focusNode,
            decoration: InputDecoration(
              hintText: hint,
              labelText: hint,
            ),
            textInputAction: TextInputAction.done,
            onSubmitted: (_) => onSubmit(),
          ),
          const SizedBox(height: 16),
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
    required this.isFocus,
    required this.defaultLabel,
    required this.viewLabel,
    required this.setDefaultLabel,
    required this.onOpen,
    required this.onFocus,
    required this.onSetDefault,
  });

  final House house;
  final bool isDefault;
  final bool isFocus;
  final String defaultLabel;
  final String viewLabel;
  final String setDefaultLabel;
  final VoidCallback onOpen;
  final VoidCallback onFocus;
  final VoidCallback onSetDefault;

  @override
  Widget build(BuildContext context) {
    return Semantics(
      button: true,
      label: '$viewLabel ${house.name}',
      excludeSemantics: true,
      child: Material(
        color: isFocus ? RoomiesPalette.of(context).tealSoft : RoomiesPalette.of(context).surface,
        borderRadius: BorderRadius.circular(16),
        child: InkWell(
          borderRadius: BorderRadius.circular(16),
          onTap: onFocus,
          onDoubleTap: onOpen,
          child: Container(
            margin: const EdgeInsets.symmetric(vertical: 8),
            padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 16),
            decoration: BoxDecoration(
              borderRadius: BorderRadius.circular(16),
              border: Border.all(
                color: isFocus ? RoomiesPalette.of(context).teal : RoomiesPalette.of(context).line,
              ),
            ),
            child: Row(
              children: [
                Container(
                  width: 48,
                  height: 48,
                  decoration: BoxDecoration(
                    color: RoomiesPalette.of(context).tealSoft,
                    borderRadius: BorderRadius.circular(12),
                  ),
                  alignment: Alignment.center,
                  child: Text(
                    house.name.isEmpty
                        ? '?'
                        : house.name.substring(0, 1).toUpperCase(),
                    style: Theme.of(context).textTheme.titleLarge?.copyWith(
                          color: RoomiesPalette.of(context).tealDeep,
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
                          style:
                              Theme.of(context).textTheme.bodySmall?.copyWith(
                                    color: RoomiesPalette.of(context).tealDeep,
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
                    color: RoomiesPalette.of(context).inkMuted,
                  ),
                IconButton(
                  tooltip: viewLabel,
                  onPressed: onOpen,
                  icon: const Icon(Icons.chevron_right_rounded),
                  color: RoomiesPalette.of(context).inkMuted,
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
