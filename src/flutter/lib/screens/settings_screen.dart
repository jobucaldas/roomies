import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../l10n/strings.dart';
import '../models/models.dart';
import '../state/app_state.dart';
import '../theme/roomies_theme.dart';
import '../widgets/app_shell.dart';

class SettingsScreen extends StatelessWidget {
  const SettingsScreen({super.key});

  Future<void> _rename(BuildContext context, AppState app, House house) async {
    final s = app.strings;
    final name = await showDialog<String>(
      context: context,
      builder: (context) => _RenameHouseDialog(strings: s, initial: house.name),
    );
    if (name == null || name.isEmpty || name == house.name) return;
    if (!context.mounted) return;
    final messenger = ScaffoldMessenger.of(context);
    try {
      await app.api.updateHouse(house.id, name);
      await app.refreshHouses();
      messenger.showSnackBar(SnackBar(content: Text(s.houseRenamed)));
    } catch (error) {
      messenger.showSnackBar(SnackBar(content: Text(error.toString())));
    }
  }

  @override
  Widget build(BuildContext context) {
    final app = context.watch<AppState>();
    final s = app.strings;

    return AppShell(
      child: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 560),
          child: ListView(
            padding: const EdgeInsets.fromLTRB(20, 8, 20, 32),
            children: [
              Semantics(
                header: true,
                child: Padding(
                  padding: const EdgeInsets.only(bottom: 20),
                  child: Text(
                    s.settings,
                    style: Theme.of(context).textTheme.headlineMedium,
                  ),
                ),
              ),
              if (app.user != null) _AccountCard(user: app.user!, strings: s),
              _Section(
                title: s.language,
                child: _Segmented<String?>(
                  selected: app.localeOverride,
                  onChanged: app.setLocaleOverride,
                  options: [
                    (null, s.languageSystem),
                    ('en', s.languageEnglish),
                    ('pt', s.languagePortuguese),
                  ],
                ),
              ),
              _Section(
                title: s.appearance,
                child: _Segmented<String?>(
                  selected: app.themeOverride,
                  onChanged: app.setThemeOverride,
                  options: [
                    (null, s.themeSystem),
                    ('light', s.themeLight),
                    ('dark', s.themeDark),
                  ],
                ),
              ),
              _Section(
                title: s.brandTheme,
                child: Wrap(
                  spacing: 10,
                  runSpacing: 10,
                  children: [
                    for (final accent in BrandAccent.values)
                      _AccentChip(
                        label: accent == BrandAccent.mint
                            ? s.brandMint
                            : s.brandPlum,
                        color: RoomiesPalette.forBrand(
                          accent,
                          Theme.of(context).brightness,
                        ).teal,
                        selected: app.brandAccent == accent,
                        onTap: () => app.setBrandAccent(accent),
                      ),
                  ],
                ),
              ),
              _Section(
                title: s.houses,
                subtitle: s.defaultHouseHint,
                child: Column(
                  children: [
                    _ChoiceTile(
                      label: s.noDefaultHouse,
                      selected: app.defaultHouseId == null,
                      onTap: () => app.setDefaultHouseId(null),
                    ),
                    for (final house in app.houses)
                      _ChoiceTile(
                        label: house.name,
                        selected: app.defaultHouseId == house.id,
                        onTap: () => app.setDefaultHouseId(house.id),
                        trailing: IconButton(
                          tooltip: s.renameHouseNamed(house.name),
                          icon: const Icon(Icons.edit_outlined, size: 20),
                          onPressed: () => _rename(context, app, house),
                        ),
                      ),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _AccountCard extends StatelessWidget {
  const _AccountCard({required this.user, required this.strings});

  final User user;
  final RoomiesStrings strings;

  @override
  Widget build(BuildContext context) {
    final p = RoomiesPalette.of(context);
    final initial = user.name.trim().isEmpty
        ? '?'
        : user.name.trim().characters.first.toUpperCase();
    return Padding(
      padding: const EdgeInsets.only(bottom: 28),
      child: Container(
        padding: const EdgeInsets.all(16),
        decoration: BoxDecoration(
          color: p.surface,
          borderRadius: BorderRadius.circular(16),
          border: Border.all(color: p.line),
        ),
        child: Row(
          children: [
            CircleAvatar(
              radius: 22,
              backgroundColor: p.tealSoft,
              foregroundColor: p.onTealSoft,
              child: Text(
                initial,
                style: const TextStyle(fontWeight: FontWeight.w700),
              ),
            ),
            const SizedBox(width: 14),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    user.name,
                    overflow: TextOverflow.ellipsis,
                    style: Theme.of(context).textTheme.titleMedium,
                  ),
                  Text(
                    user.email,
                    overflow: TextOverflow.ellipsis,
                    style: Theme.of(context).textTheme.bodyMedium,
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

class _Section extends StatelessWidget {
  const _Section({
    required this.title,
    required this.child,
    this.subtitle,
  });

  final String title;
  final String? subtitle;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 28),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(title, style: Theme.of(context).textTheme.titleLarge),
          if (subtitle != null) ...[
            const SizedBox(height: 2),
            Text(
              subtitle!,
              style: Theme.of(context).textTheme.bodyMedium,
            ),
          ],
          const SizedBox(height: 12),
          child,
        ],
      ),
    );
  }
}

class _Segmented<T> extends StatelessWidget {
  const _Segmented({
    required this.selected,
    required this.onChanged,
    required this.options,
  });

  final T selected;
  final ValueChanged<T> onChanged;
  final List<(T, String)> options;

  @override
  Widget build(BuildContext context) {
    // Plain ToggleButtons-style row: SegmentedButton cannot hold a null value,
    // and "Device" is modelled as no override.
    final p = RoomiesPalette.of(context);
    return Container(
      padding: const EdgeInsets.all(4),
      decoration: BoxDecoration(
        color: p.mist,
        borderRadius: BorderRadius.circular(14),
      ),
      child: Row(
        children: [
          for (final (value, label) in options)
            Expanded(
              child: Semantics(
                button: true,
                selected: value == selected,
                label: label,
                excludeSemantics: true,
                child: Material(
                  color: value == selected ? p.surface : Colors.transparent,
                  borderRadius: BorderRadius.circular(10),
                  child: InkWell(
                    borderRadius: BorderRadius.circular(10),
                    onTap: () => onChanged(value),
                    child: Padding(
                      padding: const EdgeInsets.symmetric(
                        horizontal: 8,
                        vertical: 12,
                      ),
                      child: Text(
                        label,
                        textAlign: TextAlign.center,
                        overflow: TextOverflow.ellipsis,
                        style: Theme.of(context).textTheme.titleSmall?.copyWith(
                              color: value == selected ? p.ink : p.inkMuted,
                              fontWeight: value == selected
                                  ? FontWeight.w700
                                  : FontWeight.w500,
                            ),
                      ),
                    ),
                  ),
                ),
              ),
            ),
        ],
      ),
    );
  }
}

class _ChoiceTile extends StatelessWidget {
  const _ChoiceTile({
    required this.label,
    required this.selected,
    required this.onTap,
    this.trailing,
  });

  final String label;
  final bool selected;
  final VoidCallback onTap;
  final Widget? trailing;

  @override
  Widget build(BuildContext context) {
    final p = RoomiesPalette.of(context);
    return Material(
      color: selected ? p.tealSoft : Colors.transparent,
      borderRadius: BorderRadius.circular(12),
      child: InkWell(
        borderRadius: BorderRadius.circular(12),
        onTap: onTap,
        child: Padding(
          padding: EdgeInsets.fromLTRB(12, 4, trailing == null ? 12 : 0, 4),
          child: ConstrainedBox(
            constraints: const BoxConstraints(minHeight: 48),
            child: Row(
              children: [
                Icon(
                  selected
                      ? Icons.radio_button_checked
                      : Icons.radio_button_off,
                  size: 22,
                  color: selected ? p.onTealSoft : p.inkMuted,
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: Text(
                    label,
                    style: Theme.of(context).textTheme.titleMedium?.copyWith(
                          color: selected ? p.onTealSoft : p.ink,
                          fontWeight:
                              selected ? FontWeight.w700 : FontWeight.w500,
                        ),
                  ),
                ),
                if (trailing != null) trailing!,
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _AccentChip extends StatelessWidget {
  const _AccentChip({
    required this.label,
    required this.color,
    required this.selected,
    required this.onTap,
  });

  final String label;
  final Color color;
  final bool selected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final p = RoomiesPalette.of(context);
    return Semantics(
      button: true,
      selected: selected,
      label: label,
      excludeSemantics: true,
      child: Material(
        color: selected ? p.surface : Colors.transparent,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(12),
          side: BorderSide(
            color: selected ? p.tealDeep : p.line,
            width: selected ? 1.6 : 1,
          ),
        ),
        child: InkWell(
          borderRadius: BorderRadius.circular(12),
          onTap: onTap,
          child: Padding(
            padding: const EdgeInsets.fromLTRB(12, 10, 16, 10),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Container(
                  width: 22,
                  height: 22,
                  decoration: BoxDecoration(
                    color: color,
                    shape: BoxShape.circle,
                  ),
                ),
                const SizedBox(width: 10),
                Text(
                  label,
                  style: Theme.of(context).textTheme.titleSmall?.copyWith(
                        fontWeight:
                            selected ? FontWeight.w700 : FontWeight.w500,
                      ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _RenameHouseDialog extends StatefulWidget {
  const _RenameHouseDialog({required this.strings, required this.initial});

  final RoomiesStrings strings;
  final String initial;

  @override
  State<_RenameHouseDialog> createState() => _RenameHouseDialogState();
}

class _RenameHouseDialogState extends State<_RenameHouseDialog> {
  late final _controller = TextEditingController(text: widget.initial);

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  void _submit() => Navigator.of(context).pop(_controller.text.trim());

  @override
  Widget build(BuildContext context) {
    final s = widget.strings;
    return AlertDialog(
      title: Text(s.renameHouse),
      content: TextField(
        controller: _controller,
        autofocus: true,
        decoration: InputDecoration(labelText: s.houseName),
        textInputAction: TextInputAction.done,
        onSubmitted: (_) => _submit(),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(context).pop(),
          child: Text(s.cancel),
        ),
        FilledButton(onPressed: _submit, child: Text(s.save)),
      ],
    );
  }
}
