import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../state/app_state.dart';
import '../theme/roomies_theme.dart';
import '../widgets/app_shell.dart';
import '../widgets/roomies_ui.dart';

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
          constraints: const BoxConstraints(maxWidth: 640),
          child: ListView(
            padding: const EdgeInsets.fromLTRB(20, 8, 20, 32),
            children: [
              RoomiesHeading(s.settings, level: 2),
              Text(
                s.defaultHouseHint,
                style: Theme.of(context).textTheme.bodyMedium,
              ),
              const SizedBox(height: 16),
              RoomiesCard(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      s.language,
                      style: Theme.of(context).textTheme.titleLarge,
                    ),
                    const SizedBox(height: 8),
                    RadioListTile<String?>(
                      contentPadding: EdgeInsets.zero,
                      title: Text(s.languageSystem),
                      subtitle: app.localeOverride == null
                          ? Text(
                              app.localeCode == 'pt'
                                  ? s.languagePortuguese
                                  : s.languageEnglish,
                            )
                          : null,
                      value: null,
                      groupValue: app.localeOverride,
                      onChanged: (value) => app.setLocaleOverride(value),
                    ),
                    RadioListTile<String?>(
                      contentPadding: EdgeInsets.zero,
                      title: Text(s.languageEnglish),
                      value: 'en',
                      groupValue: app.localeOverride,
                      onChanged: (value) => app.setLocaleOverride(value),
                    ),
                    RadioListTile<String?>(
                      contentPadding: EdgeInsets.zero,
                      title: Text(s.languagePortuguese),
                      value: 'pt',
                      groupValue: app.localeOverride,
                      onChanged: (value) => app.setLocaleOverride(value),
                    ),
                  ],
                ),
              ),
              RoomiesCard(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      s.defaultHouse,
                      style: Theme.of(context).textTheme.titleLarge,
                    ),
                    const SizedBox(height: 4),
                    Text(
                      s.defaultHouseHint,
                      style: Theme.of(context).textTheme.bodyMedium,
                    ),
                    const SizedBox(height: 8),
                    RadioListTile<String?>(
                      contentPadding: EdgeInsets.zero,
                      title: Text(s.noDefaultHouse),
                      value: null,
                      groupValue: app.defaultHouseId,
                      onChanged: (value) => app.setDefaultHouseId(value),
                    ),
                    ...app.houses.map(
                      (house) => RadioListTile<String?>(
                        contentPadding: EdgeInsets.zero,
                        title: Text(house.name),
                        value: house.id,
                        groupValue: app.defaultHouseId,
                        onChanged: (value) => app.setDefaultHouseId(value),
                      ),
                    ),
                    if (app.houses.isNotEmpty) ...[
                      const SizedBox(height: 8),
                      Text(
                        s.switchHouse,
                        style: Theme.of(context).textTheme.titleMedium,
                      ),
                      const SizedBox(height: 6),
                      ...app.houses.map(
                        (house) => ListTile(
                          contentPadding: EdgeInsets.zero,
                          leading: Icon(
                            Icons.home_outlined,
                            color: RoomiesColors.tealDeep,
                          ),
                          title: Text(house.name),
                          trailing: house.id == app.defaultHouseId
                              ? Text(
                                  s.defaultBadge,
                                  style: Theme.of(context)
                                      .textTheme
                                      .bodySmall
                                      ?.copyWith(color: RoomiesColors.tealDeep),
                                )
                              : null,
                          onTap: () => context.go('/house/${house.id}'),
                        ),
                      ),
                    ],
                  ],
                ),
              ),
              RoomiesCard(
                child: ListTile(
                  contentPadding: EdgeInsets.zero,
                  leading: const Icon(Icons.logout_rounded),
                  title: Text(s.logout),
                  onTap: () async {
                    await app.logout();
                    if (context.mounted) context.go('/');
                  },
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
