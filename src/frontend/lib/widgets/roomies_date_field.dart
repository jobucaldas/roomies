import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../core/datetime_format.dart';
import '../state/app_state.dart';

enum RoomiesDateFieldKind { date, time, dateTime }

String _two(int v) => v.toString().padLeft(2, '0');

String _apiDate(DateTime d) => '${d.year.toString().padLeft(4, '0')}-${_two(d.month)}-${_two(d.day)}';

String _apiTime(TimeOfDay t) => '${_two(t.hour)}:${_two(t.minute)}';

/// Read-only field that opens the platform date/time picker and writes the
/// API string form (`yyyy-MM-dd`, `HH:mm`, `yyyy-MM-ddTHH:mm`) into
/// [controller], so callers keep treating it as a plain text value.
///
/// With [compact] the date-time is written as `yyyyMMddTHHmmss` (RRULE UNTIL).
/// With [multiple] each pick is appended to a comma separated list instead of
/// replacing the value (exception dates).
class RoomiesDateField extends StatefulWidget {
  const RoomiesDateField({
    super.key,
    required this.controller,
    this.kind = RoomiesDateFieldKind.date,
    this.label,
    this.compact = false,
    this.multiple = false,
    this.clearable = false,
    this.timeFrom,
    this.onChanged,
  });

  final TextEditingController controller;
  final RoomiesDateFieldKind kind;
  final String? label;
  final bool compact;
  final bool multiple;
  final bool clearable;
  /// Date-only pick that reuses the time of day of this controller's value
  /// (recurrence exceptions and UNTIL must line up with the start time).
  final TextEditingController? timeFrom;
  final VoidCallback? onChanged;

  @override
  State<RoomiesDateField> createState() => _RoomiesDateFieldState();
}

class _RoomiesDateFieldState extends State<RoomiesDateField> {
  late final TextEditingController _display = TextEditingController();

  @override
  void initState() {
    super.initState();
    widget.controller.addListener(_sync);
  }

  @override
  void didUpdateWidget(covariant RoomiesDateField old) {
    super.didUpdateWidget(old);
    if (old.controller != widget.controller) {
      old.controller.removeListener(_sync);
      widget.controller.addListener(_sync);
    }
  }

  @override
  void dispose() {
    widget.controller.removeListener(_sync);
    _display.dispose();
    super.dispose();
  }

  String _show(String raw, String locale) {
    final v = raw.trim();
    if (v.isEmpty) return '';
    switch (widget.kind) {
      case RoomiesDateFieldKind.time:
        return v;
      case RoomiesDateFieldKind.date:
        return formatDisplayDate(v, localeCode: locale);
      case RoomiesDateFieldKind.dateTime:
        return v
            .split(',')
            .map((p) => formatDisplayDateTime(p.trim(), localeCode: locale))
            .join(', ');
    }
  }

  void _sync() {
    final locale = context.read<AppState>().localeCode;
    final text = _show(widget.controller.text, locale);
    if (_display.text != text) _display.text = text;
  }

  DateTime? _current() {
    final raw = widget.controller.text.trim();
    if (widget.multiple || raw.isEmpty) return null;
    return tryParseApiDateTime(raw);
  }

  Future<void> _pick() async {
    final now = DateTime.now();
    final current = _current();
    final initial = current ?? now;
    DateTime? date;
    if (widget.kind != RoomiesDateFieldKind.time) {
      date = await showDatePicker(
        context: context,
        initialDate: initial,
        firstDate: DateTime(2000),
        lastDate: DateTime(2100),
      );
      if (date == null || !mounted) return;
    }
    TimeOfDay? time;
    final anchor = widget.timeFrom == null
        ? null
        : tryParseApiDateTime(widget.timeFrom!.text);
    if (anchor != null) {
      time = TimeOfDay.fromDateTime(anchor);
    } else if (widget.kind != RoomiesDateFieldKind.date) {
      final parts = widget.controller.text.split(':');
      final seed = widget.kind == RoomiesDateFieldKind.time && parts.length == 2
          ? TimeOfDay(
              hour: int.tryParse(parts[0]) ?? now.hour,
              minute: int.tryParse(parts[1]) ?? now.minute,
            )
          : TimeOfDay.fromDateTime(initial);
      time = await showTimePicker(context: context, initialTime: seed);
      if (time == null || !mounted) return;
    }
    final value = switch (widget.kind) {
      RoomiesDateFieldKind.date => _apiDate(date!),
      RoomiesDateFieldKind.time => _apiTime(time!),
      RoomiesDateFieldKind.dateTime => widget.compact
          ? '${date!.year.toString().padLeft(4, '0')}${_two(date.month)}${_two(date.day)}'
              'T${_two(time!.hour)}${_two(time.minute)}00'
          : '${_apiDate(date!)}T${_apiTime(time!)}:00',
    };
    // Scheduled-event requests expect `yyyy-MM-ddTHH:mm`; keep minutes only.
    final stored = widget.kind == RoomiesDateFieldKind.dateTime &&
            !widget.compact &&
            !widget.multiple
        ? value.substring(0, 16)
        : value;
    if (widget.multiple) {
      final existing = widget.controller.text
          .split(',')
          .map((p) => p.trim())
          .where((p) => p.isNotEmpty)
          .toList();
      if (!existing.contains(stored)) existing.add(stored);
      widget.controller.text = existing.join(', ');
    } else {
      widget.controller.text = stored;
    }
    widget.onChanged?.call();
  }

  void _clear() {
    widget.controller.clear();
    widget.onChanged?.call();
  }

  @override
  Widget build(BuildContext context) {
    final locale = context.watch<AppState>().localeCode;
    final text = _show(widget.controller.text, locale);
    if (_display.text != text) _display.text = text;
    final icon = widget.kind == RoomiesDateFieldKind.time
        ? Icons.schedule
        : Icons.calendar_today_outlined;
    return TextField(
      controller: _display,
      readOnly: true,
      canRequestFocus: true,
      showCursor: false,
      onTap: _pick,
      decoration: InputDecoration(
        labelText: widget.label,
        suffixIcon: widget.clearable && widget.controller.text.isNotEmpty
            ? IconButton(
                icon: const Icon(Icons.clear),
                onPressed: _clear,
              )
            : IconButton(icon: Icon(icon), onPressed: _pick),
      ),
    );
  }
}

/// Time-of-day field backed by minutes since midnight; opens the native time
/// picker. [onChanged] receives null when a clearable field is emptied.
class RoomiesTimeField extends StatelessWidget {
  const RoomiesTimeField({
    super.key,
    required this.label,
    required this.minutes,
    required this.onChanged,
    this.clearable = false,
  });

  final String label;
  final int? minutes;
  final ValueChanged<int?> onChanged;
  final bool clearable;

  Future<void> _pick(BuildContext context) async {
    final seed = minutes == null
        ? TimeOfDay.now()
        : TimeOfDay(hour: minutes! ~/ 60, minute: minutes! % 60);
    final picked = await showTimePicker(context: context, initialTime: seed);
    if (picked != null) onChanged(picked.hour * 60 + picked.minute);
  }

  @override
  Widget build(BuildContext context) {
    final text = minutes == null
        ? ''
        : '${_two(minutes! ~/ 60)}:${_two(minutes! % 60)}';
    return TextField(
      key: ValueKey('time-$label-$text'),
      controller: TextEditingController(text: text),
      readOnly: true,
      showCursor: false,
      onTap: () => _pick(context),
      decoration: InputDecoration(
        labelText: label,
        suffixIcon: clearable && minutes != null
            ? IconButton(
                icon: const Icon(Icons.clear),
                onPressed: () => onChanged(null),
              )
            : IconButton(
                icon: const Icon(Icons.schedule),
                onPressed: () => _pick(context),
              ),
      ),
    );
  }
}
