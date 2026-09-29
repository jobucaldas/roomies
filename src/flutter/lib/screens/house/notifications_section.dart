import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/roles.dart';
import '../../models/models.dart';
import '../../state/app_state.dart';
import '../../widgets/roomies_ui.dart';

class NotificationsSection extends StatefulWidget {
  const NotificationsSection({
    super.key,
    required this.houseId,
    required this.role,
  });

  final String houseId;
  final HouseRole? role;

  @override
  State<NotificationsSection> createState() => _NotificationsSectionState();
}

class _NotificationsSectionState extends State<NotificationsSection> {
  var _loading = true;
  String? _loadError;
  NotificationPreferences? _prefs;
  String _status = '';

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
      final prefs = await context
          .read<AppState>()
          .api
          .getNotificationPreferences(widget.houseId);
      if (mounted) {
        setState(() {
          _prefs = prefs;
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

  Future<void> _save() async {
    if (_prefs == null) return;
    try {
      await context
          .read<AppState>()
          .api
          .putNotificationPreferences(widget.houseId, _prefs!);
      setState(() => _status = 'Notification preferences saved.');
    } catch (error) {
      setState(() => _status = 'Could not save preferences: $error');
    }
  }

  @override
  Widget build(BuildContext context) {
    final monitor = widget.role == HouseRole.monitor;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const RoomiesHeading('Notifications & schedule', level: 2),
        if (_loading)
          const Text('Loading notification settings…')
        else if (_loadError != null) ...[
          Text('Unable to load notification settings: $_loadError'),
          RoomiesPrimaryButton(
            label: 'Retry loading notification settings',
            onPressed: _load,
          ),
        ] else if (_prefs != null) ...[
          RoomiesCard(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const RoomiesHeading('Your notification preferences', level: 3),
                CheckboxListTile(
                  title: const Text(' Shared expense alerts'),
                  value: _prefs!.expenseCreatedEnabled,
                  onChanged: (v) => setState(
                    () => _prefs = NotificationPreferences(
                      houseId: _prefs!.houseId,
                      userId: _prefs!.userId,
                      expenseCreatedEnabled: v ?? true,
                      reminderEnabled: _prefs!.reminderEnabled,
                      cadence: _prefs!.cadence,
                      timezone: _prefs!.timezone,
                      quietStartMinutes: _prefs!.quietStartMinutes,
                      quietEndMinutes: _prefs!.quietEndMinutes,
                      digestMinutes: _prefs!.digestMinutes,
                    ),
                  ),
                ),
                RoomiesPrimaryButton(
                  label: 'Save preferences',
                  onPressed: _save,
                ),
              ],
            ),
          ),
          if (monitor)
            const Text(
              'Monitors can view scheduled events but cannot create, edit, or delete them.',
            ),
        ],
        if (_status.isNotEmpty) Text(_status),
      ],
    );
  }
}
