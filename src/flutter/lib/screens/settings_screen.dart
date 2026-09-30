import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../state/app_state.dart';
import '../theme/roomies_theme.dart';
import '../widgets/app_shell.dart';

class SettingsScreen extends StatelessWidget {
  const SettingsScreen({super.key});

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
                                  ?.copyWith(color: RoomiesColors.inkMuted),
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
              const SizedBox(height: 12),
              ListTile(
                contentPadding: EdgeInsets.zero,
                leading: Icon(Icons.logout_rounded, color: RoomiesColors.danger),
                title: Text(
                  s.logout,
                  style: TextStyle(color: RoomiesColors.danger),
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
      padding: const EdgeInsets.only(bottom: 20),
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
          const SizedBox(height: 10),
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
      color: selected ? RoomiesColors.tealSoft : Colors.transparent,
      borderRadius: BorderRadius.circular(12),
      child: InkWell(
        borderRadius: BorderRadius.circular(12),
        onTap: onTap,
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 12),
          child: Row(
            children: [
              Icon(
                selected
                    ? Icons.radio_button_checked
                    : Icons.radio_button_off,
                size: 20,
                color: selected
                    ? RoomiesColors.tealDeep
                    : RoomiesColors.inkMuted,
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
