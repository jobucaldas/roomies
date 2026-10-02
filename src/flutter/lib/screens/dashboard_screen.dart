import 'dart:async';
import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../core/datetime_format.dart';
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
  Timer? _noteRotate;
  var _noteIndex = 0;

  /// Bumps on every summary fetch. A response applies only if it is still the
  /// latest, so a slow house cannot overwrite the one the user switched to.
  int _summaryGeneration = 0;

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final create = GoRouterState.of(context).uri.queryParameters['create'];
    if (create == '1' && !_showCreate) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (!mounted) return;
        setState(() => _showCreate = true);
      });
    }
  }

  @override
  void dispose() {
    _noteRotate?.cancel();
    _houseNameFocus.dispose();
    _houseName.dispose();
    super.dispose();
  }

  void _syncNoteRotation() {
    _noteRotate?.cancel();
    if (_notes.length < 2) return;
    _noteRotate = Timer.periodic(const Duration(seconds: 5), (_) {
      if (!mounted || _notes.isEmpty) return;
      setState(() => _noteIndex = (_noteIndex + 1) % _notes.length);
    });
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
        _showCreate = empty ||
            GoRouterState.of(context).uri.queryParameters['create'] == '1';
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
      _noteIndex = 0;
    });
    try {
      final api = context.read<AppState>().api;
      final results = await Future.wait([
        api.getExpenses(houseId),
        api.getNotes(houseId),
        api.getCalendar(houseId),
        api.getScheduledEvents(houseId),
        api.getChores(houseId),
      ]);
      if (!mounted || generation != _summaryGeneration) return;
      final expenses = results[0] as List<Expense>;
      final notes = results[1] as List<Note>;
      final calendar = results[2] as List<CalendarEvent>;
      final scheduled = results[3] as List<ScheduledHouseEvent>;
      final chores = results[4] as List<Chore>;
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
        ...chores.where((c) => c.enabled).map(
              (c) => _DashEvent(
                title: c.title,
                when: c.dueLocal,
                kind: 'chore',
              ),
            ),
      ]..sort((a, b) => a.when.compareTo(b.when));
      setState(() {
        _monthExpenses = monthExpenses;
        _notes = notes.take(8).toList();
        _events = events.take(6).toList();
        _summaryLoading = false;
      });
      _syncNoteRotation();
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
      currentHouseId: focusHouse?.id,
      onHouseSelected: (id) => _loadSummary(id),
      child: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 760),
          child: ListView(
            padding: const EdgeInsets.fromLTRB(20, 8, 20, 32),
            children: [
              TweenAnimationBuilder<double>(
                tween: Tween(begin: 0, end: 1),
                duration: const Duration(milliseconds: 420),
                curve: Curves.easeOutCubic,
                builder: (context, value, child) {
                  return Transform.translate(
                    offset: Offset(0, (1 - value) * 12),
                    child: child,
                  );
                },
                child: Semantics(
                  header: true,
                  child: Text(
                    user == null ? s.welcomeGuest : s.welcomeBack(user.name),
                    style: Theme.of(context).textTheme.headlineMedium,
                  ),
                ),
              ),
              if (focusHouse == null) ...[
                const SizedBox(height: 6),
                Text(
                  s.noHousesYet,
                  style: Theme.of(context).textTheme.bodyLarge?.copyWith(
                        color: RoomiesPalette.of(context).inkMuted,
                      ),
                ),
              ],
              const SizedBox(height: 16),
              if (_error != null) RoomiesError(_error!),
              if (_loading)
                const Padding(
                  padding: EdgeInsets.symmetric(vertical: 48),
                  child: Center(
                    child: CircularProgressIndicator(strokeWidth: 2.5),
                  ),
                )
              else ...[
                if (houses.isEmpty || _showCreate)
                  _CreateHouseBlock(
                    controller: _houseName,
                    focusNode: _houseNameFocus,
                    creating: _creating,
                    title: s.createNewHouse,
                    hint: s.houseName,
                    buttonLabel: s.createHouse,
                    onSubmit: _createHouse,
                    onCancel: houses.isEmpty
                        ? null
                        : () => setState(() => _showCreate = false),
                    cancelLabel: s.cancel,
                  ),
                if (focusHouse != null && !_showCreate) ...[
                  if (_summaryError != null) ...[
                    RoomiesError(_summaryError!),
                    TextButton(
                      onPressed: () => _loadSummary(focusHouse!.id),
                      child: Text(s.retry),
                    ),
                    const SizedBox(height: 12),
                  ],
                  if (_summaryLoading)
                    const Padding(
                      padding: EdgeInsets.symmetric(vertical: 40),
                      child: Center(
                        child: CircularProgressIndicator(strokeWidth: 2.5),
                      ),
                    )
                  else ...[
                    _MoneyGraphCard(
                      title: s.moneyGraph,
                      totalLabel: s.monthSpendTotal(
                        app.money(monthTotal),
                      ),
                      emptyLabel: s.noMonthExpenses,
                      expenses: _monthExpenses,
                      money: app.money,
                    ),
                    const SizedBox(height: 12),
                    _UpcomingCard(
                      title: s.upcoming,
                      emptyLabel: s.noUpcoming,
                      events: _events,
                      localeCode: app.localeCode,
                    ),
                    const SizedBox(height: 12),
                    _NotesShowcaseCard(
                      title: s.notesShowcase,
                      emptyLabel: s.noRecentNotes,
                      notes: _notes,
                      index: _noteIndex,
                      onOpen: () =>
                          context.go('/house/${focusHouse!.id}?tab=notes'),
                      openLabel: s.openNotes,
                      onIndexChanged: (i) => setState(() => _noteIndex = i),
                    ),
                  ],
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

class _MoneyGraphCard extends StatelessWidget {
  const _MoneyGraphCard({
    required this.title,
    required this.totalLabel,
    required this.emptyLabel,
    required this.expenses,
    required this.money,
  });

  final String title;
  final String totalLabel;
  final String emptyLabel;
  final List<Expense> expenses;
  final String Function(double amount) money;

  @override
  Widget build(BuildContext context) {
    final p = RoomiesPalette.of(context);
    final now = DateTime.now();
    final daysInMonth = DateTime(now.year, now.month + 1, 0).day;
    final byDay = List<double>.filled(daysInMonth, 0);
    for (final expense in expenses) {
      final day = int.tryParse(expense.date.split('-').last) ?? 1;
      if (day >= 1 && day <= daysInMonth) byDay[day - 1] += expense.amount;
    }
    final maxVal = byDay.fold<double>(0, math.max);

    return RoomiesCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(title, style: Theme.of(context).textTheme.titleMedium),
          const SizedBox(height: 8),
          if (expenses.isEmpty)
            Text(emptyLabel)
          else ...[
            Text(
              totalLabel,
              style: Theme.of(context).textTheme.headlineSmall?.copyWith(
                    color: p.tealDeep,
                  ),
            ),
            const SizedBox(height: 16),
            SizedBox(
              height: 96,
              child: CustomPaint(
                painter: _SpendBarsPainter(
                  values: byDay,
                  maxValue: maxVal <= 0 ? 1 : maxVal,
                  barColor: p.teal,
                  trackColor: p.mist,
                ),
                child: const SizedBox.expand(),
              ),
            ),
            const SizedBox(height: 10),
            for (final expense in expenses.take(3))
              Padding(
                padding: const EdgeInsets.symmetric(vertical: 4),
                child: Row(
                  children: [
                    Expanded(
                      child: Text(
                        expense.description,
                        overflow: TextOverflow.ellipsis,
                        style: Theme.of(context).textTheme.titleSmall,
                      ),
                    ),
                    Text(
                      money(expense.amount),
                      style: Theme.of(context).textTheme.bodySmall?.copyWith(
                            color: p.inkMuted,
                          ),
                    ),
                  ],
                ),
              ),
          ],
        ],
      ),
    );
  }
}

class _SpendBarsPainter extends CustomPainter {
  _SpendBarsPainter({
    required this.values,
    required this.maxValue,
    required this.barColor,
    required this.trackColor,
  });

  final List<double> values;
  final double maxValue;
  final Color barColor;
  final Color trackColor;

  /// One slim bar per day of the month; days without spend show a short
  /// track so the month's shape stays readable.
  @override
  void paint(Canvas canvas, Size size) {
    if (values.isEmpty) return;
    final count = values.length;
    final slot = size.width / count;
    final barWidth = math.min(slot * 0.6, 14.0);
    final radius = Radius.circular(barWidth / 2);
    final paint = Paint()..style = PaintingStyle.fill;
    for (var i = 0; i < count; i++) {
      final x = i * slot + (slot - barWidth) / 2;
      final value = values[i];
      final h = value <= 0
          ? barWidth
          : math.max(barWidth, (value / maxValue) * size.height);
      paint.color = value <= 0 ? trackColor : barColor;
      canvas.drawRRect(
        RRect.fromRectAndRadius(
          Rect.fromLTWH(x, size.height - h, barWidth, h),
          radius,
        ),
        paint,
      );
    }
  }

  @override
  bool shouldRepaint(covariant _SpendBarsPainter oldDelegate) {
    return oldDelegate.values != values ||
        oldDelegate.maxValue != maxValue ||
        oldDelegate.barColor != barColor;
  }
}

class _UpcomingCard extends StatelessWidget {
  const _UpcomingCard({
    required this.title,
    required this.emptyLabel,
    required this.events,
    required this.localeCode,
  });

  final String title;
  final String emptyLabel;
  final List<_DashEvent> events;
  final String localeCode;

  IconData _iconFor(String kind) {
    switch (kind) {
      case 'chore':
        return Icons.checklist_rounded;
      case 'schedule':
        return Icons.alarm_rounded;
      default:
        return Icons.event_rounded;
    }
  }

  @override
  Widget build(BuildContext context) {
    final p = RoomiesPalette.of(context);
    return RoomiesCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(title, style: Theme.of(context).textTheme.titleMedium),
          const SizedBox(height: 10),
          if (events.isEmpty)
            Text(emptyLabel)
          else
            for (final event in events)
              Padding(
                padding: const EdgeInsets.symmetric(vertical: 8),
                child: Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Container(
                      width: 40,
                      height: 40,
                      decoration: BoxDecoration(
                        color: p.tealSoft,
                        borderRadius: BorderRadius.circular(12),
                      ),
                      child: Icon(
                        _iconFor(event.kind),
                        color: p.onTealSoft,
                        size: 20,
                      ),
                    ),
                    const SizedBox(width: 12),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            event.title,
                            style: Theme.of(context).textTheme.titleSmall,
                          ),
                          Text(
                            formatDisplayDateTime(
                              event.when,
                              localeCode: localeCode,
                            ),
                            style:
                                Theme.of(context).textTheme.bodySmall?.copyWith(
                                      color: p.inkMuted,
                                    ),
                          ),
                        ],
                      ),
                    ),
                  ],
                ),
              ),
        ],
      ),
    );
  }
}

class _NotesShowcaseCard extends StatelessWidget {
  const _NotesShowcaseCard({
    required this.title,
    required this.emptyLabel,
    required this.notes,
    required this.index,
    required this.onOpen,
    required this.openLabel,
    required this.onIndexChanged,
  });

  final String title;
  final String emptyLabel;
  final List<Note> notes;
  final int index;
  final VoidCallback onOpen;
  final String openLabel;
  final ValueChanged<int> onIndexChanged;

  @override
  Widget build(BuildContext context) {
    final p = RoomiesPalette.of(context);
    final safeIndex = notes.isEmpty ? 0 : index % notes.length;
    final note = notes.isEmpty ? null : notes[safeIndex];

    return RoomiesCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Row(
            children: [
              Expanded(
                child: Text(title, style: Theme.of(context).textTheme.titleMedium),
              ),
              TextButton(onPressed: onOpen, child: Text(openLabel)),
            ],
          ),
          const SizedBox(height: 8),
          if (note == null)
            Text(emptyLabel)
          else
            GestureDetector(
              onHorizontalDragEnd: (details) {
                if (notes.length < 2) return;
                final vx = details.primaryVelocity ?? 0;
                if (vx < -40) {
                  onIndexChanged((safeIndex + 1) % notes.length);
                } else if (vx > 40) {
                  onIndexChanged(
                    (safeIndex - 1 + notes.length) % notes.length,
                  );
                }
              },
              child: AnimatedSwitcher(
                duration: const Duration(milliseconds: 360),
                switchInCurve: Curves.easeOutCubic,
                switchOutCurve: Curves.easeInCubic,
                transitionBuilder: (child, anim) {
                  return FadeTransition(
                    opacity: anim,
                    child: SlideTransition(
                      position: Tween<Offset>(
                        begin: const Offset(0.04, 0),
                        end: Offset.zero,
                      ).animate(anim),
                      child: child,
                    ),
                  );
                },
                child: Container(
                  key: ValueKey(note.id),
                  width: double.infinity,
                  padding: const EdgeInsets.all(16),
                  decoration: BoxDecoration(
                    color: p.canvas,
                    borderRadius: BorderRadius.circular(14),
                    border: Border.all(color: p.line),
                  ),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        note.title,
                        style: Theme.of(context).textTheme.titleLarge,
                      ),
                      const SizedBox(height: 8),
                      Text(
                        note.content,
                        maxLines: 4,
                        overflow: TextOverflow.ellipsis,
                        style: Theme.of(context).textTheme.bodyLarge,
                      ),
                    ],
                  ),
                ),
              ),
            ),
          if (notes.length > 1) ...[
            const SizedBox(height: 12),
            Row(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                for (var i = 0; i < notes.length; i++)
                  Padding(
                    padding: const EdgeInsets.symmetric(horizontal: 3),
                    child: AnimatedContainer(
                      duration: const Duration(milliseconds: 220),
                      width: i == safeIndex ? 16 : 6,
                      height: 6,
                      decoration: BoxDecoration(
                        color: i == safeIndex ? p.teal : p.line,
                        borderRadius: BorderRadius.circular(999),
                      ),
                    ),
                  ),
              ],
            ),
          ],
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
    this.onCancel,
    this.cancelLabel,
  });

  final TextEditingController controller;
  final FocusNode focusNode;
  final bool creating;
  final String title;
  final String hint;
  final String buttonLabel;
  final VoidCallback onSubmit;
  final VoidCallback? onCancel;
  final String? cancelLabel;

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
          Row(
            mainAxisAlignment: MainAxisAlignment.end,
            children: [
              if (onCancel != null)
                TextButton(
                  onPressed: onCancel,
                  child: Text(cancelLabel ?? 'Cancel'),
                ),
              const SizedBox(width: 8),
              FilledButton(
                onPressed: creating ? null : onSubmit,
                child: Text(buttonLabel),
              ),
            ],
          ),
        ],
      ),
    );
  }
}
