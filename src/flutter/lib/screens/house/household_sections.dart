import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

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
    if (_name.text.trim().isEmpty) {
      setState(() => _status = 'A grocery name is required.');
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
    setState(() => _status = 'Grocery saved.');
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
    await context
        .read<AppState>()
        .api
        .deleteGrocery(widget.houseId, item.id, item.version);
    setState(() => _status = 'Grocery deleted.');
    await _load();
  }

  @override
  Widget build(BuildContext context) {
    final canWrite = _writable(widget.role);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const RoomiesHeading('Groceries', level: 2),
        if (_loading) const Text('Loading groceries…'),
        if (_loadError != null) ...[
          RoomiesError('Unable to load groceries: $_loadError'),
          RoomiesPrimaryButton(label: 'Retry groceries', onPressed: _load),
        ],
        if (!canWrite) const Text('Your monitor role is view-only.'),
        if (canWrite && !_loading && _loadError == null) ...[
          RoomiesLabeledField(
            label: 'Name',
            child: TextField(controller: _name),
          ),
          RoomiesPrimaryButton(label: 'Add grocery', onPressed: _add),
        ],
        if (_status.isNotEmpty) Text(_status),
        if (!_loading && _loadError == null && _items.isEmpty)
          const Text('No groceries yet.'),
        ..._items.map((item) {
          return RoomiesArticleCard(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                RoomiesHeading(item.name, level: 3),
                Text(item.checked ? 'Checked' : 'Needed'),
                if (canWrite) ...[
                  RoomiesPrimaryButton(
                    label: item.checked ? 'Uncheck' : 'Check',
                    onPressed: () => _toggle(item),
                  ),
                  RoomiesPrimaryButton(
                    label: 'Delete',
                    onPressed: () => _delete(item),
                  ),
                ],
              ],
            ),
          );
        }),
      ],
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
    final due = _dueLocal.text.length == 16 ? '${_dueLocal.text}:00' : _dueLocal.text;
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
    setState(() => _status = 'Chore saved.');
    await _load();
  }

  Future<void> _toggle(Chore chore) async {
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
    setState(() => _status = 'Chore enabled state saved.');
    await _load();
  }

  Future<void> _delete(Chore chore) async {
    await context
        .read<AppState>()
        .api
        .deleteChore(widget.houseId, chore.id, chore.version);
    setState(() => _status = 'Chore deleted.');
    await _load();
  }

  @override
  Widget build(BuildContext context) {
    final canWrite = _writable(widget.role);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const RoomiesHeading('Chores', level: 2),
        if (_loading) const Text('Loading chores…'),
        if (!canWrite) const Text('Your monitor role is view-only.'),
        if (canWrite && !_loading) ...[
          RoomiesLabeledField(
            label: 'Title',
            child: TextField(controller: _title),
          ),
          RoomiesLabeledField(
            label: 'Due local',
            child: TextField(controller: _dueLocal),
          ),
          RoomiesPrimaryButton(label: 'Create chore', onPressed: _create),
        ],
        if (_status.isNotEmpty) Text(_status),
        ..._chores.map((chore) => RoomiesArticleCard(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  RoomiesHeading(chore.title, level: 3),
                  if (canWrite) ...[
                    RoomiesPrimaryButton(
                      label: chore.enabled ? 'Disable' : 'Enable',
                      onPressed: () => _toggle(chore),
                    ),
                    RoomiesPrimaryButton(
                      label: 'Delete',
                      onPressed: () => _delete(chore),
                    ),
                  ],
                ],
              ),
            )),
      ],
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

  String _local(String value) =>
      value.length == 16 ? '$value:00' : value;

  Future<void> _create() async {
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
    setState(() => _status = 'Calendar event saved.');
    await _load();
  }

  Future<void> _delete(CalendarEvent event) async {
    await context
        .read<AppState>()
        .api
        .deleteCalendarEvent(widget.houseId, event.id, event.version);
    setState(() => _status = 'Calendar event deleted.');
    await _load();
  }

  @override
  Widget build(BuildContext context) {
    final canWrite = _writable(widget.role);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const RoomiesHeading('Calendar', level: 2),
        if (_loading) const Text('Loading calendar…'),
        if (!canWrite) const Text('Your monitor role is view-only.'),
        if (canWrite && !_loading) ...[
          RoomiesLabeledField(
            label: 'Title',
            child: TextField(controller: _title),
          ),
          RoomiesLabeledField(
            label: 'Start local',
            child: TextField(controller: _start),
          ),
          RoomiesLabeledField(
            label: 'End local',
            child: TextField(controller: _end),
          ),
          RoomiesPrimaryButton(
            label: 'Create calendar event',
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
                    RoomiesPrimaryButton(
                      label: 'Delete',
                      onPressed: () => _delete(event),
                    ),
                ],
              ),
            )),
      ],
    );
  }
}

class ChatSection extends StatefulWidget {
  const ChatSection({
    super.key,
    required this.houseId,
    required this.role,
    required this.userId,
  });

  final String houseId;
  final HouseRole? role;
  final String userId;

  @override
  State<ChatSection> createState() => _ChatSectionState();
}

class _ChatSectionState extends State<ChatSection> {
  List<ChatMessage> _messages = [];
  var _loading = true;
  String _status = '';
  ChatMessage? _editing;
  final _body = TextEditingController();

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    _body.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    setState(() => _loading = true);
    final page = await context.read<AppState>().api.getChat(widget.houseId);
    if (mounted) {
      setState(() {
        _messages = page.messages;
        _loading = false;
      });
    }
  }

  Future<void> _send() async {
    if (_editing != null) {
      await context.read<AppState>().api.updateChat(
            widget.houseId,
            _editing!.id,
            _body.text.trim(),
          );
      setState(() {
        _status = 'Message edited.';
        _editing = null;
      });
    } else {
      await context
          .read<AppState>()
          .api
          .createChat(widget.houseId, _body.text.trim());
      setState(() => _status = 'Message sent.');
    }
    _body.clear();
    await _load();
  }

  Future<void> _delete(ChatMessage message) async {
    await context.read<AppState>().api.deleteChat(widget.houseId, message.id);
    setState(() => _status = 'Message deleted.');
    await _load();
  }

  @override
  Widget build(BuildContext context) {
    final canWrite = _writable(widget.role);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const RoomiesHeading('Chat', level: 2),
        if (_loading) const Text('Loading chat…'),
        RoomiesPrimaryButton(
          label: 'Refresh chat',
          onPressed: () async {
            await _load();
            setState(() => _status = 'Chat refreshed.');
          },
        ),
        if (!canWrite) const Text('Your monitor role is view-only.'),
        if (canWrite) ...[
          RoomiesLabeledField(
            label: 'Message',
            child: TextField(controller: _body, maxLines: 3),
          ),
          RoomiesPrimaryButton(
            label: _editing != null ? 'Save message' : 'Send message',
            onPressed: _send,
          ),
        ],
        if (_status.isNotEmpty) Text(_status),
        ..._messages.where((m) => m.body != null).map((message) {
          final own = message.authorId == widget.userId;
          return RoomiesArticleCard(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(message.body ?? ''),
                if (own && canWrite) ...[
                  RoomiesPrimaryButton(
                    label: 'Edit message',
                    onPressed: () {
                      setState(() {
                        _editing = message;
                        _body.text = message.body ?? '';
                      });
                    },
                  ),
                  RoomiesPrimaryButton(
                    label: 'Delete message',
                    onPressed: () => _delete(message),
                  ),
                ],
              ],
            ),
          );
        }),
      ],
    );
  }
}
