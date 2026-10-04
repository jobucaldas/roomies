import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../api/api_error.dart';
import '../../core/roles.dart';
import '../../models/models.dart';
import '../../state/app_state.dart';
import '../../widgets/roomies_ui.dart';

bool _writable(HouseRole? role) => role != null && role != HouseRole.monitor;

String _defaultRrule() => 'FREQ=WEEKLY;INTERVAL=1;COUNT=1';

class GroceriesSection extends StatefulWidget {
  const GroceriesSection({
    super.key,
    required this.houseId,
    required this.role,
    required this.members,
  });

  final String houseId;
  final HouseRole? role;
  final List<HouseMember> members;

  @override
  State<GroceriesSection> createState() => _GroceriesSectionState();
}

class _GroceriesSectionState extends State<GroceriesSection> {
  List<GroceryItem> _items = [];
  var _loading = true;
  String? _loadError;
  String _status = '';
  final _name = TextEditingController();

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    _name.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _loadError = null;
    });
    try {
      final items =
          await context.read<AppState>().api.getGroceries(widget.houseId);
      if (mounted) {
        setState(() {
          _items = items;
          _loading = false;
        });
      }
    } catch (error) {
      if (mounted) {
        setState(() {
          _loadError = error.toString();
          _loading = false;
        });
      }
    }
  }

  Future<void> _add() async {
    final s = context.read<AppState>().strings;
    if (_name.text.trim().isEmpty) {
      setState(() => _status = s.name);
      return;
    }
    await context.read<AppState>().api.createGrocery(widget.houseId, {
      'name': _name.text.trim(),
      'quantity': '',
      'unit': '',
      'note': '',
      'assignee_id': null,
      'position': 0,
    });
    _name.clear();
    setState(() => _status = s.grocerySaved);
    await _load();
  }

  Future<void> _toggle(GroceryItem item) async {
    await context.read<AppState>().api.toggleGrocery(
          widget.houseId,
          item.id,
          !item.checked,
          item.version,
        );
    await _load();
  }

  Future<void> _delete(GroceryItem item) async {
    final s = context.read<AppState>().strings;
    await context
        .read<AppState>()
        .api
        .deleteGrocery(widget.houseId, item.id, item.version);
    setState(() => _status = s.groceryDeleted);
    await _load();
  }

  @override
  Widget build(BuildContext context) {
    final s = context.watch<AppState>().strings;
    final canWrite = _writable(widget.role);
    return RoomiesTabPanel(
      name: s.tabGroceries,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          RoomiesHeading(s.tabGroceries, level: 2),
          if (_loading) Text(s.loading),
          if (_loadError != null) ...[
            RoomiesError(_loadError!),
            RoomiesPrimaryButton(label: s.retry, onPressed: _load),
          ],
          if (!canWrite) Text(s.viewOnlyRole),
          if (canWrite && !_loading && _loadError == null) ...[
            RoomiesLabeledField(
              label: s.name,
              child: TextField(controller: _name),
            ),
            RoomiesPrimaryButton(label: s.addGrocery, onPressed: _add),
          ],
          if (_status.isNotEmpty) Text(_status),
          if (!_loading && _loadError == null && _items.isEmpty)
            Text(s.noGroceriesYet),
          ..._items.map((item) {
            return RoomiesArticleCard(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  RoomiesHeading(item.name, level: 3),
                  Text(item.checked ? s.checked : s.needed),
                  if (canWrite)
                    RoomiesItemActions([
                      RoomiesItemAction(
                        label: item.checked ? s.uncheck : s.check,
                        onPressed: () => _toggle(item),
                      ),
                      RoomiesItemAction(
                        label: s.delete,
                        destructive: true,
                        onPressed: () => _delete(item),
                      ),
                    ]),
                ],
              ),
            );
          }),
        ],
      ),
    );
  }
}

class ChoresSection extends StatefulWidget {
  const ChoresSection({super.key, required this.houseId, required this.role});

  final String houseId;
  final HouseRole? role;

  @override
  State<ChoresSection> createState() => _ChoresSectionState();
}

class _ChoresSectionState extends State<ChoresSection> {
  List<Chore> _chores = [];
  var _loading = true;
  String _status = '';
  final _title = TextEditingController();
  final _dueLocal = TextEditingController();

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    _title.dispose();
    _dueLocal.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    setState(() => _loading = true);
    final chores =
        await context.read<AppState>().api.getChores(widget.houseId);
    if (mounted) {
      setState(() {
        _chores = chores;
        _loading = false;
      });
    }
  }

  Future<void> _create() async {
    final s = context.read<AppState>().strings;
    final due =
        _dueLocal.text.length == 16 ? '${_dueLocal.text}:00' : _dueLocal.text;
    try {
      await context.read<AppState>().api.createChore(widget.houseId, {
        'title': _title.text.trim(),
        'description': '',
        'assignee_id': null,
        'timezone': 'UTC',
        'due_local': due,
        'rrule': _defaultRrule(),
        'exdates': <String>[],
        'enabled': true,
      });
      setState(() => _status = s.choreSaved);
      await _load();
    } on ApiError catch (error) {
      setState(() => _status = error.message);
    }
  }

  Future<void> _toggle(Chore chore) async {
    final s = context.read<AppState>().strings;
    await context.read<AppState>().api.updateChore(widget.houseId, chore.id, {
      'title': chore.title,
      'description': chore.description,
      'assignee_id': chore.assigneeId,
      'timezone': chore.timezone,
      'due_local': chore.dueLocal,
      'rrule': chore.rrule,
      'exdates': chore.exdates,
      'enabled': !chore.enabled,
      'version': chore.version,
    });
    setState(() => _status = s.choreEnabledSaved);
    await _load();
  }

  Future<void> _delete(Chore chore) async {
    final s = context.read<AppState>().strings;
    await context
        .read<AppState>()
        .api
        .deleteChore(widget.houseId, chore.id, chore.version);
    setState(() => _status = s.choreDeleted);
    await _load();
  }

  @override
  Widget build(BuildContext context) {
    final s = context.watch<AppState>().strings;
    final canWrite = _writable(widget.role);
    return RoomiesTabPanel(
      name: s.tabChores,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          RoomiesHeading(s.tabChores, level: 2),
          if (_loading) Text(s.loading),
          if (!canWrite) Text(s.viewOnlyRole),
          if (canWrite && !_loading) ...[
            RoomiesLabeledField(
              label: s.title,
              child: TextField(controller: _title),
            ),
            RoomiesLabeledField(
              label: s.dueLocal,
              child: TextField(controller: _dueLocal),
            ),
            RoomiesPrimaryButton(label: s.createChore, onPressed: _create),
          ],
          if (_status.isNotEmpty) Text(_status),
          ..._chores.map((chore) => RoomiesArticleCard(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    RoomiesHeading(chore.title, level: 3),
                    if (canWrite)
                      RoomiesItemActions([
                        RoomiesItemAction(
                          label: chore.enabled ? s.disable : s.enable,
                          onPressed: () => _toggle(chore),
                        ),
                        RoomiesItemAction(
                          label: s.delete,
                          destructive: true,
                          onPressed: () => _delete(chore),
                        ),
                      ]),
                  ],
                ),
              )),
        ],
      ),
    );
  }
}

class CalendarSection extends StatefulWidget {
  const CalendarSection({super.key, required this.houseId, required this.role});

  final String houseId;
  final HouseRole? role;

  @override
  State<CalendarSection> createState() => _CalendarSectionState();
}

class _CalendarSectionState extends State<CalendarSection> {
  List<CalendarEvent> _events = [];
  var _loading = true;
  String _status = '';
  final _title = TextEditingController();
  final _start = TextEditingController();
  final _end = TextEditingController();

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    _title.dispose();
    _start.dispose();
    _end.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    setState(() => _loading = true);
    final events =
        await context.read<AppState>().api.getCalendar(widget.houseId);
    if (mounted) {
      setState(() {
        _events = events;
        _loading = false;
      });
    }
  }

  String _local(String value) => value.length == 16 ? '$value:00' : value;

  Future<void> _create() async {
    final s = context.read<AppState>().strings;
    await context.read<AppState>().api.createCalendarEvent(widget.houseId, {
      'title': _title.text.trim(),
      'description': '',
      'timezone': 'UTC',
      'start_local': _local(_start.text),
      'end_local': _local(_end.text),
      'all_day': false,
      'rrule': _defaultRrule(),
      'exdates': <String>[],
    });
    setState(() => _status = s.calendarEventSaved);
    await _load();
  }

  Future<void> _delete(CalendarEvent event) async {
    final s = context.read<AppState>().strings;
    await context
        .read<AppState>()
        .api
        .deleteCalendarEvent(widget.houseId, event.id, event.version);
    setState(() => _status = s.calendarEventDeleted);
    await _load();
  }

  @override
  Widget build(BuildContext context) {
    final s = context.watch<AppState>().strings;
    final canWrite = _writable(widget.role);
    return RoomiesTabPanel(
      name: s.tabCalendar,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          RoomiesHeading(s.tabCalendar, level: 2),
          if (_loading) Text(s.loading),
          if (!canWrite) Text(s.viewOnlyRole),
          if (canWrite && !_loading) ...[
            RoomiesLabeledField(
              label: s.title,
              child: TextField(controller: _title),
            ),
            RoomiesLabeledField(
              label: s.startLocal,
              child: TextField(controller: _start),
            ),
            RoomiesLabeledField(
              label: s.endLocal,
              child: TextField(controller: _end),
            ),
            RoomiesPrimaryButton(
              label: s.createCalendarEvent,
              onPressed: _create,
            ),
          ],
          if (_status.isNotEmpty) Text(_status),
          ..._events.map((event) => RoomiesArticleCard(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    RoomiesHeading(event.title, level: 3),
                    if (canWrite)
                      RoomiesItemActions([
                        RoomiesItemAction(
                          label: s.delete,
                          destructive: true,
                          onPressed: () => _delete(event),
                        ),
                      ]),
                  ],
                ),
              )),
        ],
      ),
    );
  }
}
