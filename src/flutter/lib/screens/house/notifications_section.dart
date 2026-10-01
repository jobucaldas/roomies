import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/roles.dart';
import '../../core/scheduled_events.dart';
import '../../models/models.dart';
import '../../push/push_service.dart';
import '../../state/app_state.dart';
import '../../widgets/roomies_ui.dart';

class NotificationsSection extends StatefulWidget {
  const NotificationsSection({
    super.key,
    required this.houseId,
    required this.role,
    required this.userId,
  });

  final String houseId;
  final HouseRole? role;
  final String userId;

  @override
  State<NotificationsSection> createState() => _NotificationsSectionState();
}

class _NotificationsSectionState extends State<NotificationsSection> {
  var _loading = true;
  String? _loadError;
  NotificationPreferences? _prefs;
  List<NotificationSubscription> _subscriptions = [];
  List<ScheduledHouseEvent> _events = [];
  String _pushStatus = browserPushInitialStatus(isWeb: kIsWeb);

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _loadError = null;
    });
    try {
      final api = context.read<AppState>().api;
      final results = await Future.wait([
        api.getNotificationPreferences(widget.houseId),
        api.getNotificationSubscriptions(widget.houseId),
        api.getScheduledEvents(widget.houseId),
      ]);
      if (!mounted) return;
      setState(() {
        _prefs = results[0] as NotificationPreferences;
        _subscriptions = results[1] as List<NotificationSubscription>;
        _events = results[2] as List<ScheduledHouseEvent>;
        _loading = false;
      });
    } catch (error) {
      if (mounted) {
        setState(() {
          _loadError = error.toString();
          _loading = false;
        });
      }
    }
  }

  bool get _canCreate =>
      widget.role != null && widget.role != HouseRole.monitor;

  bool get _admin => widget.role == HouseRole.admin;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        RoomiesHeading(context.watch<AppState>().strings.notificationsSchedule, level: 2),
        if (_loading)
          Semantics(
            liveRegion: true,
            child: Text(context.watch<AppState>().strings.loadingNotificationSettings),
          )
        else if (_loadError != null) ...[
          Semantics(
            liveRegion: true,
            child: Text('Unable to load notification settings: $_loadError'),
          ),
          FilledButton(
            onPressed: _load,
            child: Text(context.watch<AppState>().strings.retryNotificationSettings),
          ),
        ] else if (_prefs != null &&
            notificationControlsReady(loading: _loading, error: _loadError)) ...[
          _PreferencesCard(
            prefs: _prefs!,
            onSaved: (value) => setState(() => _prefs = value),
          ),
          _BrowserPushCard(
            houseId: widget.houseId,
            subscriptions: _subscriptions,
            initialStatus: _pushStatus,
            onSubscriptionsChanged: (value) =>
                setState(() => _subscriptions = value),
            onStatusChanged: (value) => setState(() => _pushStatus = value),
          ),
          _ScheduleCard(
            houseId: widget.houseId,
            userId: widget.userId,
            events: _events,
            canCreate: _canCreate,
            admin: _admin,
            onEventsChanged: (value) => setState(() => _events = value),
          ),
        ],
      ],
    );
  }
}

class _PreferencesCard extends StatefulWidget {
  const _PreferencesCard({
    required this.prefs,
    required this.onSaved,
  });

  final NotificationPreferences prefs;
  final ValueChanged<NotificationPreferences> onSaved;

  @override
  State<_PreferencesCard> createState() => _PreferencesCardState();
}

class _PreferencesCardState extends State<_PreferencesCard> {
  late NotificationPreferences _value;
  String _status = '';

  @override
  void initState() {
    super.initState();
    _value = widget.prefs;
  }

  @override
  void didUpdateWidget(covariant _PreferencesCard oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.prefs.houseId != widget.prefs.houseId ||
        oldWidget.prefs.userId != widget.prefs.userId) {
      _value = widget.prefs;
    }
  }

  Future<void> _save() async {
    if (_value.timezone.trim().isEmpty) {
      setState(() {
        _status =
            'Use an IANA timezone and times between 00:00 and 23:59.';
      });
      return;
    }
    try {
      final saved = await context.read<AppState>().api.putNotificationPreferences(
            widget.prefs.houseId,
            _value,
          );
      if (!mounted) return;
      setState(() => _status = 'Notification preferences saved.');
      widget.onSaved(saved);
    } catch (error) {
      if (mounted) {
        setState(() => _status = 'Could not save preferences: $error');
      }
    }
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
            RoomiesHeading(context.watch<AppState>().strings.yourNotificationPreferences, level: 3),
            CheckboxListTile(
              contentPadding: EdgeInsets.zero,
              title: const Text('Shared expense alerts'),
              value: _value.expenseCreatedEnabled,
              onChanged: (checked) => setState(
                () => _value =
                    _value.copyWith(expenseCreatedEnabled: checked ?? true),
              ),
            ),
            CheckboxListTile(
              contentPadding: EdgeInsets.zero,
              title: const Text('Scheduled reminder alerts'),
              value: _value.reminderEnabled,
              onChanged: (checked) => setState(
                () => _value = _value.copyWith(reminderEnabled: checked ?? true),
              ),
            ),
            DropdownButtonFormField<String>(
              value: _value.cadence,
              decoration: const InputDecoration(labelText: 'Delivery'),
              items: const [
                DropdownMenuItem(value: 'immediate', child: Text('Immediate')),
                DropdownMenuItem(
                  value: 'daily_digest',
                  child: Text('Daily digest'),
                ),
              ],
              onChanged: (cadence) {
                if (cadence == null) return;
                setState(() => _value = _value.copyWith(cadence: cadence));
              },
            ),
            TextFormField(
              initialValue: _value.timezone,
              decoration: const InputDecoration(
                labelText: 'IANA timezone',
                hintText: 'Europe/Lisbon',
              ),
              onChanged: (timezone) =>
                  setState(() => _value = _value.copyWith(timezone: timezone)),
            ),
            TextFormField(
              key: ValueKey('quiet-start-${_value.quietStartMinutes}'),
              initialValue: minutesTime(_value.quietStartMinutes),
              decoration: const InputDecoration(labelText: 'Quiet start (local)'),
              keyboardType: TextInputType.datetime,
              onChanged: (raw) {
                final minutes = timeMinutes(raw);
                setState(
                  () => _value = minutes == null
                      ? _value.copyWith(clearQuietStart: true)
                      : _value.copyWith(quietStartMinutes: minutes),
                );
              },
            ),
            TextFormField(
              key: ValueKey('quiet-end-${_value.quietEndMinutes}'),
              initialValue: minutesTime(_value.quietEndMinutes),
              decoration: const InputDecoration(labelText: 'Quiet end (local)'),
              keyboardType: TextInputType.datetime,
              onChanged: (raw) {
                final minutes = timeMinutes(raw);
                setState(
                  () => _value = minutes == null
                      ? _value.copyWith(clearQuietEnd: true)
                      : _value.copyWith(quietEndMinutes: minutes),
                );
              },
            ),
            TextFormField(
              key: ValueKey('digest-${_value.digestMinutes}'),
              initialValue: minutesTime(_value.digestMinutes),
              decoration:
                  const InputDecoration(labelText: 'Daily digest time (local)'),
              keyboardType: TextInputType.datetime,
              onChanged: (raw) {
                final minutes = timeMinutes(raw);
                setState(
                  () => _value = _value.copyWith(
                    digestMinutes: minutes ?? _value.digestMinutes,
                  ),
                );
              },
            ),
            FilledButton(
              onPressed: _save,
              child: Text(context.watch<AppState>().strings.savePreferences),
            ),
            if (_status.isNotEmpty)
              Semantics(liveRegion: true, child: Text(_status)),
          ],
        ),
      ),
    );
  }
}

class _BrowserPushCard extends StatefulWidget {
  const _BrowserPushCard({
    required this.houseId,
    required this.subscriptions,
    required this.initialStatus,
    required this.onSubscriptionsChanged,
    required this.onStatusChanged,
  });

  final String houseId;
  final List<NotificationSubscription> subscriptions;
  final String initialStatus;
  final ValueChanged<List<NotificationSubscription>> onSubscriptionsChanged;
  final ValueChanged<String> onStatusChanged;

  @override
  State<_BrowserPushCard> createState() => _BrowserPushCardState();
}

class _BrowserPushCardState extends State<_BrowserPushCard> {
  late String _status;
  var _busy = false;

  @override
  void initState() {
    super.initState();
    _status = widget.initialStatus;
  }

  bool get _enabled =>
      widget.subscriptions.any((subscription) => subscription.platform == 'web_push');

  Future<void> _enable() async {
    setState(() => _busy = true);
    final api = context.read<AppState>().api;
    final error = await enableBrowserPush(api, widget.houseId);
    if (!mounted) return;
    if (error != null) {
      setState(() {
        _status = error;
        _busy = false;
      });
      widget.onStatusChanged(error);
      return;
    }
    try {
      final refreshed =
          await api.getNotificationSubscriptions(widget.houseId);
      if (!mounted) return;
      widget.onSubscriptionsChanged(refreshed);
      const message = 'Browser push enabled.';
      setState(() {
        _status = message;
        _busy = false;
      });
      widget.onStatusChanged(message);
    } catch (error) {
      final message = 'Subscription saved but refresh failed: $error';
      setState(() {
        _status = message;
        _busy = false;
      });
      widget.onStatusChanged(message);
    }
  }

  Future<void> _disable() async {
    setState(() => _busy = true);
    final api = context.read<AppState>().api;
    final ids = widget.subscriptions
        .where((subscription) => subscription.platform == 'web_push')
        .map((subscription) => subscription.id)
        .toList();
    final error = await disableBrowserPush(api, widget.houseId, ids);
    if (!mounted) return;
    if (error != null) {
      setState(() {
        _status = error;
        _busy = false;
      });
      widget.onStatusChanged(error);
      return;
    }
    final remaining = widget.subscriptions
        .where((subscription) => subscription.platform != 'web_push')
        .toList();
    widget.onSubscriptionsChanged(remaining);
    const message = 'Browser push disabled.';
    setState(() {
      _status = message;
      _busy = false;
    });
    widget.onStatusChanged(message);
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
            const RoomiesHeading('Browser push', level: 3),
            const Text(
              'Browser push is only available in a supported web browser.',
            ),
            if (_enabled)
              FilledButton(
                onPressed: _busy ? null : _disable,
                child: const Text('Disable browser push'),
              )
            else
              FilledButton(
                onPressed: _busy ? null : _enable,
                child: const Text('Enable browser push'),
              ),
            Semantics(liveRegion: true, child: Text(_status)),
            const RoomiesHeading('Your devices', level: 4),
            for (final subscription in widget.subscriptions)
              Text(
                '${subscription.platform}: ${subscription.deviceLabel} '
                '(last seen ${subscription.lastSeenAt})',
              ),
          ],
        ),
      ),
    );
  }
}

class _ScheduleCard extends StatefulWidget {
  const _ScheduleCard({
    required this.houseId,
    required this.userId,
    required this.events,
    required this.canCreate,
    required this.admin,
    required this.onEventsChanged,
  });

  final String houseId;
  final String userId;
  final List<ScheduledHouseEvent> events;
  final bool canCreate;
  final bool admin;
  final ValueChanged<List<ScheduledHouseEvent>> onEventsChanged;

  @override
  State<_ScheduleCard> createState() => _ScheduleCardState();
}

class _ScheduleCardState extends State<_ScheduleCard> {
  late List<ScheduledHouseEvent> _items;
  String _status = '';
  var _saving = false;
  String? _editingId;
  final _title = TextEditingController();
  final _start = TextEditingController();
  final _zone = TextEditingController(text: 'UTC');
  var _frequency = 'DAILY';
  final _interval = TextEditingController(text: '1');
  final _count = TextEditingController(text: '1');
  final _until = TextEditingController();
  final _exdates = TextEditingController();

  @override
  void initState() {
    super.initState();
    _items = List.of(widget.events);
  }

  @override
  void didUpdateWidget(covariant _ScheduleCard oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.events != widget.events && _editingId == null) {
      _items = List.of(widget.events);
    }
  }

  @override
  void dispose() {
    _title.dispose();
    _start.dispose();
    _zone.dispose();
    _interval.dispose();
    _count.dispose();
    _until.dispose();
    _exdates.dispose();
    super.dispose();
  }

  void _beginEdit(ScheduledHouseEvent event) {
    _title.text = event.title;
    _start.text = event.dtstartLocal.length >= 16
        ? event.dtstartLocal.substring(0, 16)
        : event.dtstartLocal;
    _zone.text = event.timezone;
    final rule = recurrenceFields(event.rrule);
    _frequency = rule.$1.isEmpty ? 'DAILY' : rule.$1;
    _interval.text = rule.$2;
    _count.text = rule.$3;
    _until.text = rule.$4;
    _exdates.text = event.exdates.join(', ');
    setState(() {
      _editingId = event.id;
      _status = 'Editing scheduled event.';
    });
  }

  void _cancelEdit() {
    setState(() {
      _editingId = null;
      _status = 'Event editing cancelled.';
    });
  }

  ScheduledEventRequest? _buildRequest() {
    try {
      return buildScheduledEventRequest(
        title: _title.text,
        start: _start.text,
        zone: _zone.text,
        freq: _frequency,
        interval: _interval.text,
        count: _count.text,
        until: _until.text,
        exdates: _exdates.text,
      );
    } on FormatException catch (error) {
      setState(() => _status = error.message);
      return null;
    }
  }

  Future<void> _save() async {
    final request = _buildRequest();
    if (request == null) return;
    setState(() => _saving = true);
    final api = context.read<AppState>().api;
    try {
      final ScheduledHouseEvent event;
      if (_editingId != null) {
        event = await api.updateScheduledEvent(
          widget.houseId,
          _editingId!,
          request.toJson(),
        );
      } else {
        event = await api.createScheduledEvent(
          widget.houseId,
          request.toJson(),
        );
      }
      if (!mounted) return;
      final items = List<ScheduledHouseEvent>.of(_items);
      final index = items.indexWhere((value) => value.id == event.id);
      if (index >= 0) {
        items[index] = event;
        _status = 'Scheduled event updated.';
      } else {
        items.add(event);
        _status = 'Scheduled event created.';
      }
      widget.onEventsChanged(items);
      setState(() {
        _items = items;
        _editingId = null;
        _saving = false;
      });
    } catch (error) {
      if (mounted) {
        setState(() {
          _status = 'Could not save event: $error';
          _saving = false;
        });
      }
    }
  }

  Future<void> _delete(String eventId) async {
    try {
      await context
          .read<AppState>()
          .api
          .deleteScheduledEvent(widget.houseId, eventId);
      if (!mounted) return;
      final items =
          _items.where((event) => event.id != eventId).toList(growable: false);
      widget.onEventsChanged(items);
      setState(() {
        _items = items;
        _status = 'Scheduled event deleted.';
      });
    } catch (error) {
      if (mounted) {
        setState(() => _status = 'Could not delete event: $error');
      }
    }
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
            RoomiesHeading(context.watch<AppState>().strings.scheduledEvents, level: 3),
            for (final event in _items)
              RoomiesArticleCard(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(event.title, style: Theme.of(context).textTheme.titleMedium),
                    Text('${event.dtstartLocal} · ${event.timezone} · ${event.rrule}'),
                    if (event.exdates.isNotEmpty)
                      Text('Exceptions: ${event.exdates.join(', ')}'),
                    if (event.creatorId == widget.userId || widget.admin) ...[
                      TextButton(
                        onPressed: () => _beginEdit(event),
                        child: Text(context.watch<AppState>().strings.edit),
                      ),
                      TextButton(
                        onPressed: () => _delete(event.id),
                        child: Text(context.watch<AppState>().strings.delete),
                      ),
                    ],
                  ],
                ),
              ),
            if (widget.canCreate) ...[
              RoomiesHeading(
                _editingId != null ? context.read<AppState>().strings.editScheduledEvent : 'Add scheduled event',
                level: 4,
              ),
              TextFormField(
                controller: _title,
                decoration: const InputDecoration(labelText: 'Title'),
              ),
              TextFormField(
                controller: _start,
                decoration: const InputDecoration(labelText: 'Local start'),
              ),
              TextFormField(
                controller: _zone,
                decoration: const InputDecoration(labelText: 'IANA timezone'),
              ),
              DropdownButtonFormField<String>(
                value: _frequency,
                decoration: const InputDecoration(labelText: 'Frequency'),
                items: const [
                  DropdownMenuItem(value: 'DAILY', child: Text('Daily')),
                  DropdownMenuItem(value: 'WEEKLY', child: Text('Weekly')),
                  DropdownMenuItem(value: 'MONTHLY', child: Text('Monthly')),
                ],
                onChanged: (value) {
                  if (value == null) return;
                  setState(() => _frequency = value);
                },
              ),
              TextFormField(
                controller: _interval,
                decoration:
                    const InputDecoration(labelText: 'Interval (1–366)'),
                keyboardType: TextInputType.number,
              ),
              TextFormField(
                controller: _count,
                decoration: const InputDecoration(
                  labelText: 'Count (1–366; leave blank to use until)',
                ),
                keyboardType: TextInputType.number,
              ),
              TextFormField(
                controller: _until,
                decoration: const InputDecoration(
                  labelText: 'Until (optional, YYYYMMDDTHHMMSS)',
                ),
              ),
              TextFormField(
                controller: _exdates,
                decoration: const InputDecoration(
                  labelText: 'EXDATE local times (comma-separated)',
                ),
              ),
              FilledButton(
                onPressed: _saving ? null : _save,
                child: Text(
                  _saving
                      ? 'Saving…'
                      : _editingId != null
                          ? 'Save scheduled event'
                          : 'Create scheduled event',
                ),
              ),
              if (_editingId != null)
                TextButton(
                  onPressed: _saving ? null : _cancelEdit,
                  child: const Text('Cancel edit'),
                ),
            ] else
              const Text(
                'Monitors can view scheduled events but cannot create, edit, or delete them.',
              ),
            if (_status.isNotEmpty)
              Semantics(liveRegion: true, child: Text(_status)),
          ],
        ),
      ),
    );
  }
}
