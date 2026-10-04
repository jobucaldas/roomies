import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/datetime_format.dart';
import '../../core/roles.dart';
import '../../core/scheduled_events.dart';
import '../../l10n/strings.dart';
import '../../models/models.dart';
import '../../push/push_service.dart';
import '../../state/app_state.dart';
import '../../widgets/roomies_date_field.dart';
import '../../widgets/roomies_ui.dart';

const _fieldGap = 14.0;
const _sectionGap = 20.0;

/// Spaces [children] evenly so form fields and buttons never touch.
List<Widget> _spaced(List<Widget> children, {double gap = _fieldGap}) => [
      for (var i = 0; i < children.length; i++) ...[
        if (i > 0) SizedBox(height: gap),
        children[i],
      ],
    ];

/// Two fields side by side on wide layouts, stacked on narrow ones.
class _FieldRow extends StatelessWidget {
  const _FieldRow(this.children);

  final List<Widget> children;

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        if (constraints.maxWidth < 520) {
          return Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: _spaced(children),
          );
        }
        return Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            for (var i = 0; i < children.length; i++) ...[
              if (i > 0) const SizedBox(width: _fieldGap),
              Expanded(child: children[i]),
            ],
          ],
        );
      },
    );
  }
}

class _StatusText extends StatelessWidget {
  const _StatusText(this.text);

  final String text;

  @override
  Widget build(BuildContext context) {
    if (text.isEmpty) return const SizedBox.shrink();
    return Semantics(
      liveRegion: true,
      child: Text(
        text,
        style: Theme.of(context).textTheme.bodySmall?.copyWith(
              color: Theme.of(context).colorScheme.onSurfaceVariant,
            ),
      ),
    );
  }
}

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
    final s = context.watch<AppState>().strings;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        RoomiesHeading(s.notificationsSchedule, level: 2),
        const SizedBox(height: 4),
        if (_loading)
          Semantics(
            liveRegion: true,
            child: Text(s.loadingNotificationSettings),
          )
        else if (_loadError != null) ...[
          Semantics(
            liveRegion: true,
            child: Text(s.unableToLoadNotificationSettings(_loadError!)),
          ),
          const SizedBox(height: 12),
          Align(
            alignment: Alignment.centerLeft,
            child: FilledButton(
              onPressed: _load,
              child: Text(s.retryNotificationSettings),
            ),
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
    final s = context.read<AppState>().strings;
    if (_value.timezone.trim().isEmpty) {
      setState(() {
        _status = s.timezoneValidationHint;
      });
      return;
    }
    try {
      final saved = await context.read<AppState>().api.putNotificationPreferences(
            widget.prefs.houseId,
            _value,
          );
      if (!mounted) return;
      setState(() => _status = s.notificationPreferencesSaved);
      widget.onSaved(saved);
    } catch (error) {
      if (mounted) {
        setState(() => _status = s.couldNotSavePreferences('$error'));
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final s = context.watch<AppState>().strings;
    return Card(
      margin: const EdgeInsets.only(top: _sectionGap),
      child: Padding(
        padding: const EdgeInsets.all(20),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            RoomiesHeading(s.yourNotificationPreferences, level: 3),
            const SizedBox(height: 4),
            CheckboxListTile(
              contentPadding: EdgeInsets.zero,
              controlAffinity: ListTileControlAffinity.leading,
              title: Text(s.sharedExpenseAlerts),
              value: _value.expenseCreatedEnabled,
              onChanged: (checked) => setState(
                () => _value =
                    _value.copyWith(expenseCreatedEnabled: checked ?? true),
              ),
            ),
            CheckboxListTile(
              contentPadding: EdgeInsets.zero,
              controlAffinity: ListTileControlAffinity.leading,
              title: Text(s.scheduledReminderAlerts),
              value: _value.reminderEnabled,
              onChanged: (checked) => setState(
                () => _value = _value.copyWith(reminderEnabled: checked ?? true),
              ),
            ),
            const SizedBox(height: _sectionGap),
            ..._spaced([
              _FieldRow([
                DropdownButtonFormField<String>(
                  value: _value.cadence,
                  isExpanded: true,
                  decoration: InputDecoration(labelText: s.delivery),
                  items: [
                    DropdownMenuItem(
                      value: 'immediate',
                      child: Text(s.deliveryImmediate),
                    ),
                    DropdownMenuItem(
                      value: 'daily_digest',
                      child: Text(s.deliveryDailyDigest),
                    ),
                  ],
                  onChanged: (cadence) {
                    if (cadence == null) return;
                    setState(() => _value = _value.copyWith(cadence: cadence));
                  },
                ),
                TextFormField(
                  initialValue: _value.timezone,
                  decoration: InputDecoration(
                    labelText: s.ianaTimezone,
                    hintText: 'Europe/Lisbon',
                  ),
                  onChanged: (timezone) => setState(
                    () => _value = _value.copyWith(timezone: timezone),
                  ),
                ),
              ]),
              _FieldRow([
                RoomiesTimeField(
                  label: s.quietStartLocal,
                  minutes: _value.quietStartMinutes,
                  clearable: true,
                  onChanged: (minutes) => setState(
                    () => _value = minutes == null
                        ? _value.copyWith(clearQuietStart: true)
                        : _value.copyWith(quietStartMinutes: minutes),
                  ),
                ),
                RoomiesTimeField(
                  label: s.quietEndLocal,
                  minutes: _value.quietEndMinutes,
                  clearable: true,
                  onChanged: (minutes) => setState(
                    () => _value = minutes == null
                        ? _value.copyWith(clearQuietEnd: true)
                        : _value.copyWith(quietEndMinutes: minutes),
                  ),
                ),
                RoomiesTimeField(
                  label: s.dailyDigestTimeLocal,
                  minutes: _value.digestMinutes,
                  onChanged: (minutes) => setState(
                    () => _value = _value.copyWith(
                      digestMinutes: minutes ?? _value.digestMinutes,
                    ),
                  ),
                ),
              ]),
            ]),
            const SizedBox(height: _sectionGap),
            Align(
              alignment: Alignment.centerLeft,
              child: FilledButton(
                onPressed: _save,
                child: Text(s.savePreferences),
              ),
            ),
            if (_status.isNotEmpty) ...[
              const SizedBox(height: 12),
              _StatusText(_status),
            ],
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
    final s = context.read<AppState>().strings;
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
      final message = s.browserPushEnabled;
      setState(() {
        _status = message;
        _busy = false;
      });
      widget.onStatusChanged(message);
    } catch (error) {
      final message = s.subscriptionRefreshFailed('$error');
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
    final s = context.read<AppState>().strings;
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
    final message = s.browserPushDisabled;
    setState(() {
      _status = message;
      _busy = false;
    });
    widget.onStatusChanged(message);
  }

  @override
  Widget build(BuildContext context) {
    final app = context.watch<AppState>();
    final s = app.strings;
    return Card(
      margin: const EdgeInsets.only(top: _sectionGap),
      child: Padding(
        padding: const EdgeInsets.all(20),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            RoomiesHeading(s.browserPush, level: 3),
            const SizedBox(height: 4),
            Text(
              s.browserPushOnlyWeb,
              style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                    color: Theme.of(context).colorScheme.onSurfaceVariant,
                  ),
            ),
            const SizedBox(height: _fieldGap),
            if (_enabled)
              OutlinedButton(
                onPressed: _busy ? null : _disable,
                child: Text(s.disableBrowserPush),
              )
            else
              FilledButton(
                onPressed: _busy ? null : _enable,
                child: Text(s.enableBrowserPush),
              ),
            if (_status.isNotEmpty) ...[
              const SizedBox(height: 12),
              _StatusText(_status),
            ],
            const SizedBox(height: _sectionGap),
            const Divider(height: 1),
            const SizedBox(height: _sectionGap),
            RoomiesHeading(s.yourDevices, level: 4),
            const SizedBox(height: 4),
            for (final subscription in widget.subscriptions)
              Padding(
                padding: const EdgeInsets.symmetric(vertical: 6),
                child: Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Padding(
                      padding: const EdgeInsets.only(top: 2, right: 10),
                      child: Icon(
                        subscription.platform == 'web_push'
                            ? Icons.language
                            : Icons.smartphone,
                        size: 18,
                      ),
                    ),
                    Expanded(
                      child: Text(
                        s.deviceLine(
                          subscription.platform,
                          subscription.deviceLabel,
                          s.lastSeen(
                            formatDisplayDateTime(
                              subscription.lastSeenAt,
                              localeCode: app.localeCode,
                            ),
                          ),
                        ),
                      ),
                    ),
                  ],
                ),
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
    final s = context.read<AppState>().strings;
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
      _status = s.editingScheduledEvent;
    });
  }

  void _cancelEdit() {
    final s = context.read<AppState>().strings;
    setState(() {
      _editingId = null;
      _status = s.eventEditingCancelled;
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
    final s = context.read<AppState>().strings;
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
        _status = s.scheduledEventUpdated;
      } else {
        items.add(event);
        _status = s.scheduledEventCreated;
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
          _status = s.couldNotSaveEvent('$error');
          _saving = false;
        });
      }
    }
  }

  Future<void> _delete(String eventId) async {
    final s = context.read<AppState>().strings;
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
        _status = s.scheduledEventDeleted;
      });
    } catch (error) {
      if (mounted) {
        setState(() => _status = s.couldNotDeleteEvent('$error'));
      }
    }
  }

  String _eventWhenLine(ScheduledHouseEvent event, RoomiesStrings s, String locale) {
    final when = formatDisplayDateTime(event.dtstartLocal, localeCode: locale);
    return '$when · ${event.timezone} · ${event.rrule}';
  }

  @override
  Widget build(BuildContext context) {
    final app = context.watch<AppState>();
    final s = app.strings;
    return Card(
      margin: const EdgeInsets.only(top: _sectionGap),
      child: Padding(
        padding: const EdgeInsets.all(20),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            RoomiesHeading(s.scheduledEvents, level: 3),
            const SizedBox(height: 4),
            for (final event in _items)
              Padding(
                padding: const EdgeInsets.only(top: 10),
                child: RoomiesArticleCard(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        event.title,
                        style: Theme.of(context).textTheme.titleMedium,
                      ),
                      const SizedBox(height: 4),
                      Text(_eventWhenLine(event, s, app.localeCode)),
                      if (event.exdates.isNotEmpty) ...[
                        const SizedBox(height: 4),
                        Text(
                          s.exceptionsList(
                            event.exdates
                                .map(
                                  (d) => formatDisplayDateTime(
                                    d,
                                    localeCode: app.localeCode,
                                  ),
                                )
                                .join(', '),
                          ),
                        ),
                      ],
                      if (event.creatorId == widget.userId || widget.admin) ...[
                        const SizedBox(height: 8),
                        Wrap(
                          spacing: 8,
                          children: [
                            TextButton(
                              onPressed: () => _beginEdit(event),
                              child: Text(s.edit),
                            ),
                            TextButton(
                              onPressed: () => _delete(event.id),
                              child: Text(s.delete),
                            ),
                          ],
                        ),
                      ],
                    ],
                  ),
                ),
              ),
            if (widget.canCreate) ...[
              const SizedBox(height: _sectionGap),
              const Divider(height: 1),
              const SizedBox(height: _sectionGap),
              RoomiesHeading(
                _editingId != null ? s.editScheduledEvent : s.addScheduledEvent,
                level: 4,
              ),
              const SizedBox(height: 4),
              ..._spaced([
                TextFormField(
                  controller: _title,
                  decoration: InputDecoration(labelText: s.title),
                ),
                _FieldRow([
                  RoomiesDateField(
                    controller: _start,
                    kind: RoomiesDateFieldKind.dateTime,
                    label: s.localStart,
                  ),
                  TextFormField(
                    controller: _zone,
                    decoration: InputDecoration(labelText: s.ianaTimezone),
                  ),
                ]),
                _FieldRow([
                  DropdownButtonFormField<String>(
                    value: _frequency,
                    isExpanded: true,
                    decoration: InputDecoration(labelText: s.frequency),
                    items: [
                      DropdownMenuItem(
                        value: 'DAILY',
                        child: Text(s.frequencyDaily),
                      ),
                      DropdownMenuItem(
                        value: 'WEEKLY',
                        child: Text(s.frequencyWeekly),
                      ),
                      DropdownMenuItem(
                        value: 'MONTHLY',
                        child: Text(s.frequencyMonthly),
                      ),
                    ],
                    onChanged: (value) {
                      if (value == null) return;
                      setState(() => _frequency = value);
                    },
                  ),
                  TextFormField(
                    controller: _interval,
                    decoration: InputDecoration(labelText: s.intervalRange),
                    keyboardType: TextInputType.number,
                  ),
                  TextFormField(
                    controller: _count,
                    decoration: InputDecoration(labelText: s.countRange),
                    keyboardType: TextInputType.number,
                  ),
                ]),
                _FieldRow([
                  RoomiesDateField(
                    controller: _until,
                    kind: RoomiesDateFieldKind.dateTime,
                    label: s.untilOptional,
                    compact: true,
                    clearable: true,
                    timeFrom: _start,
                  ),
                  RoomiesDateField(
                    controller: _exdates,
                    kind: RoomiesDateFieldKind.dateTime,
                    label: s.exdateLocalTimes,
                    multiple: true,
                    clearable: true,
                    timeFrom: _start,
                  ),
                ]),
              ]),
              const SizedBox(height: _sectionGap),
              Wrap(
                spacing: 8,
                runSpacing: 8,
                children: [
                  FilledButton(
                    onPressed: _saving ? null : _save,
                    child: Text(
                      _saving
                          ? s.savingEllipsis
                          : _editingId != null
                              ? s.saveScheduledEvent
                              : s.createScheduledEvent,
                    ),
                  ),
                  if (_editingId != null)
                    TextButton(
                      onPressed: _saving ? null : _cancelEdit,
                      child: Text(s.cancelEdit),
                    ),
                ],
              ),
            ] else ...[
              const SizedBox(height: 12),
              Text(s.monitorsViewOnlySchedule),
            ],
            if (_status.isNotEmpty) ...[
              const SizedBox(height: 12),
              _StatusText(_status),
            ],
          ],
        ),
      ),
    );
  }
}
