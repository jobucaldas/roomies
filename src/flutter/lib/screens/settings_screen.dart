import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../models/models.dart';
import '../state/app_state.dart';
import '../theme/roomies_theme.dart';
import '../widgets/app_shell.dart';

class SettingsScreen extends StatefulWidget {
  const SettingsScreen({super.key});

  @override
  State<SettingsScreen> createState() => _SettingsScreenState();
}

class _SettingsScreenState extends State<SettingsScreen> {
  final _rename = TextEditingController();
  String? _renameHouseId;
  var _renaming = false;
  var _renameSeeded = false;
  String? _renameStatus;

  @override
  void dispose() {
    _rename.dispose();
    super.dispose();
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    if (_renameSeeded) return;
    final app = context.read<AppState>();
    final id = app.defaultHouseId ??
        (app.houses.isNotEmpty ? app.houses.first.id : null);
    _renameHouseId = id;
    if (id != null) {
      for (final house in app.houses) {
        if (house.id == id) {
          _rename.text = house.name;
          break;
        }
      }
    }
    _renameSeeded = true;
  }

  void _selectHouseForRename(House house) {
    setState(() {
      _renameHouseId = house.id;
      _rename.text = house.name;
      _renameStatus = null;
    });
  }

  Future<void> _saveRename(AppState app) async {
    final id = _renameHouseId;
    final name = _rename.text.trim();
    if (id == null || name.isEmpty) {
      setState(() => _renameStatus = app.strings.selectHouseToRename);
      return;
    }
    setState(() => _renaming = true);
    try {
      await app.api.updateHouse(id, name);
      await app.refreshHouses();
      if (!mounted) return;
      setState(() {
        _renaming = false;
        _renameStatus = app.strings.houseRenamed;
      });
    } catch (error) {
      if (mounted) {
        setState(() {
          _renaming = false;
          _renameStatus = error.toString();
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final app = context.watch<AppState>();
    final s = app.strings;

    return AppShell(
      title: s.settings,
      showBrand: false,
      child: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 560),
          child: ListView(
            padding: const EdgeInsets.fromLTRB(20, 8, 20, 32),
            children: [
              _Section(
                title: s.language,
                child: Column(
                  children: [
                    _ChoiceTile(
                      label: s.languageSystem,
                      selected: app.localeOverride == null,
                      trailing: app.localeOverride == null
                          ? Text(
                              app.localeCode == 'pt'
                                  ? s.languagePortuguese
                                  : s.languageEnglish,
                              style: Theme.of(context)
                                  .textTheme
                                  .bodySmall
                                  ?.copyWith(color: RoomiesPalette.of(context).inkMuted),
                            )
                          : null,
                      onTap: () => app.setLocaleOverride(null),
                    ),
                    _ChoiceTile(
                      label: s.languageEnglish,
                      selected: app.localeOverride == 'en',
                      onTap: () => app.setLocaleOverride('en'),
                    ),
                    _ChoiceTile(
                      label: s.languagePortuguese,
                      selected: app.localeOverride == 'pt',
                      onTap: () => app.setLocaleOverride('pt'),
                    ),
                  ],
                ),
              ),
              _Section(
                title: s.appearance,
                child: Column(
                  children: [
                    _ChoiceTile(
                      label: s.themeSystem,
                      selected: app.themeOverride == null,
                      onTap: () => app.setThemeOverride(null),
                    ),
                    _ChoiceTile(
                      label: s.themeLight,
                      selected: app.themeOverride == 'light',
                      onTap: () => app.setThemeOverride('light'),
                    ),
                    _ChoiceTile(
                      label: s.themeDark,
                      selected: app.themeOverride == 'dark',
                      onTap: () => app.setThemeOverride('dark'),
                    ),
                  ],
                ),
              ),
              _Section(
                title: s.defaultHouse,
                subtitle: s.defaultHouseHint,
                child: Column(
                  children: [
                    _ChoiceTile(
                      label: s.noDefaultHouse,
                      selected: app.defaultHouseId == null,
                      onTap: () => app.setDefaultHouseId(null),
                    ),
                    ...app.houses.map(
                      (house) => _ChoiceTile(
                        label: house.name,
                        selected: app.defaultHouseId == house.id,
                        onTap: () => app.setDefaultHouseId(house.id),
                        trailing: TextButton(
                          onPressed: () => context.go('/house/${house.id}'),
                          child: Text(s.switchHouse),
                        ),
                      ),
                    ),
                  ],
                ),
              ),
              if (app.houses.isNotEmpty)
                _Section(
                  title: s.renameHouse,
                  subtitle: s.renameHouseHint,
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: [
                      ...app.houses.map(
                        (house) => _ChoiceTile(
                          label: house.name,
                          selected: _renameHouseId == house.id,
                          onTap: () => _selectHouseForRename(house),
                        ),
                      ),
                      const SizedBox(height: 12),
                      TextField(
                        controller: _rename,
                        decoration: InputDecoration(labelText: s.houseName),
                      ),
                      const SizedBox(height: 16),
                      Align(
                        alignment: Alignment.centerRight,
                        child: FilledButton(
                          onPressed: _renaming ? null : () => _saveRename(app),
                          child: Text(s.save),
                        ),
                      ),
                      if (_renameStatus != null) ...[
                        const SizedBox(height: 8),
                        Text(_renameStatus!),
                      ],
                    ],
                  ),
                ),
              const SizedBox(height: 12),
              ListTile(
                contentPadding: EdgeInsets.zero,
                minVerticalPadding: 16,
                leading:
                    Icon(Icons.logout_rounded, color: RoomiesPalette.of(context).danger),
                title: Text(
                  s.logout,
                  style: TextStyle(color: RoomiesPalette.of(context).danger),
                ),
                onTap: () async {
                  await app.logout();
                  if (context.mounted) context.go('/');
                },
              ),
            ],
          ),
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
      padding: const EdgeInsets.only(bottom: 24),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(title, style: Theme.of(context).textTheme.titleLarge),
          if (subtitle != null) ...[
            const SizedBox(height: 4),
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
    return Material(
      color: selected ? RoomiesPalette.of(context).tealSoft : Colors.transparent,
      borderRadius: BorderRadius.circular(12),
      child: InkWell(
        borderRadius: BorderRadius.circular(12),
        onTap: onTap,
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 14),
          child: Row(
            children: [
              Icon(
                selected
                    ? Icons.radio_button_checked
                    : Icons.radio_button_off,
                size: 22,
                color: selected
                    ? RoomiesPalette.of(context).tealDeep
                    : RoomiesPalette.of(context).inkMuted,
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Text(
                  label,
                  style: Theme.of(context).textTheme.titleMedium?.copyWith(
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
    );
  }
}
