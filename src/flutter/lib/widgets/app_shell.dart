import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../l10n/strings.dart';
import '../models/models.dart';
import '../state/app_state.dart';
import '../theme/roomies_theme.dart';
import 'roomies_ui.dart';

/// Authenticated chrome: sidebar (wide) / drawer (narrow) for houses + settings.
///
/// Breakpoint state is sticky across soft-keyboard height changes: depending on
/// [MediaQuery.sizeOf] alone rebuilds the whole shell when only height changes,
/// which on Flutter web remounts semantics inputs and dismisses the keyboard.
class AppShell extends StatefulWidget {
  const AppShell({
    super.key,
    required this.child,
    this.currentHouseId,
    this.title,
    this.actions = const [],
    this.showBrand = true,
  });

  final Widget child;
  final String? currentHouseId;
  final String? title;
  final List<Widget> actions;
  final bool showBrand;

  static const double wideBreakpoint = 900;

  @override
  State<AppShell> createState() => _AppShellState();
}

class _AppShellState extends State<AppShell> {
  /// Last width-derived layout mode. Updated only when width crosses the
  /// breakpoint so keyboard-driven height / viewInsets churn does not swap
  /// Material+sidebar ↔ Scaffold+drawer (a full child remount).
  bool? _wide;

  @override
  Widget build(BuildContext context) {
    final app = context.watch<AppState>();
    final s = app.strings;

    // LayoutBuilder (not MediaQuery.sizeOf): soft-keyboard height changes must
    // not be treated as a breakpoint signal. Sticky [_wide] keeps sidebar vs
    // drawer stable across those rebuilds.
    return LayoutBuilder(
      builder: (context, constraints) {
        final wideNow = constraints.maxWidth >= AppShell.wideBreakpoint;
        if (_wide == null) {
          _wide = wideNow;
        } else if (_wide != wideNow) {
          final next = wideNow;
          WidgetsBinding.instance.addPostFrameCallback((_) {
            if (!mounted) return;
            if (_wide != next) setState(() => _wide = next);
          });
        }
        return _buildChrome(context, app, s, _wide ?? wideNow);
      },
    );
  }

  Widget _buildChrome(
    BuildContext context,
    AppState app,
    RoomiesStrings s,
    bool wide,
  ) {
    final nav = _ShellNav(
      strings: s,
      houses: app.houses,
      defaultHouseId: app.defaultHouseId,
      currentHouseId: widget.currentHouseId,
      currentPath: GoRouterState.of(context).uri.path,
      onDashboard: () => context.go('/dashboard'),
      onHouse: (id) => context.go('/house/$id'),
      onSettings: () => context.go('/settings'),
      onLogout: () async {
        await app.logout();
        if (context.mounted) context.go('/');
      },
    );

    if (wide) {
      return Material(
        color: RoomiesPalette.of(context).canvas,
        child: RoomiesAtmosphere(
          child: SafeArea(
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                SizedBox(width: 268, child: nav),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: [
                      _ShellTopBar(
                        title: widget.title,
                        showBrand: widget.showBrand,
                        strings: s,
                        actions: widget.actions,
                      ),
                      Expanded(child: widget.child),
                    ],
                  ),
                ),
              ],
            ),
          ),
        ),
      );
    }

    // resizeToAvoidBottomInset: false — on Flutter web / mobile Chrome the
    // default inset resize rebuilds the body under an open soft keyboard and
    // drops TextField focus (keyboard flashes then dismisses).
    return Scaffold(
      backgroundColor: Colors.transparent,
      resizeToAvoidBottomInset: false,
      drawer: Drawer(
        backgroundColor: RoomiesPalette.of(context).surface,
        child: SafeArea(child: nav),
      ),
      body: RoomiesAtmosphere(
        child: SafeArea(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Builder(
                builder: (context) => _ShellTopBar(
                  title: widget.title,
                  showBrand: widget.showBrand,
                  strings: s,
                  leading: IconButton(
                    tooltip: s.openMenu,
                    onPressed: () => Scaffold.of(context).openDrawer(),
                    icon: const Icon(Icons.menu_rounded),
                  ),
                  actions: widget.actions,
                ),
              ),
              Expanded(child: widget.child),
            ],
          ),
        ),
      ),
    );
  }
}

class _ShellTopBar extends StatelessWidget {
  const _ShellTopBar({
    required this.strings,
    required this.actions,
    this.title,
    this.showBrand = true,
    this.leading,
  });

  final RoomiesStrings strings;
  final List<Widget> actions;
  final String? title;
  final bool showBrand;
  final Widget? leading;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(12, 8, 12, 4),
      child: Row(
        children: [
          if (leading != null) leading!,
          if (showBrand)
            Text(
              strings.brand,
              style: Theme.of(context).textTheme.titleLarge?.copyWith(
                    color: RoomiesPalette.of(context).tealDeep,
                    fontFamily: 'Fraunces',
                  ),
            ),
          if (title != null) ...[
            if (showBrand) ...[
              const SizedBox(width: 10),
              Text(
                '·',
                style: TextStyle(color: RoomiesPalette.of(context).inkMuted),
              ),
              const SizedBox(width: 10),
            ],
            Expanded(
              // House / page title must be a heading for a11y
              // (Playwright getByRole('heading', { name: houseName })).
              child: Semantics(
                header: true,
                child: Text(
                  title!,
                  overflow: TextOverflow.ellipsis,
                  style: Theme.of(context).textTheme.titleMedium,
                ),
              ),
            ),
          ] else
            const Spacer(),
          ...actions,
        ],
      ),
    );
  }
}

class _ShellNav extends StatelessWidget {
  const _ShellNav({
    required this.strings,
    required this.houses,
    required this.defaultHouseId,
    required this.currentHouseId,
    required this.currentPath,
    required this.onDashboard,
    required this.onHouse,
    required this.onSettings,
    required this.onLogout,
  });

  final RoomiesStrings strings;
  final List<House> houses;
  final String? defaultHouseId;
  final String? currentHouseId;
  final String currentPath;
  final VoidCallback onDashboard;
  final ValueChanged<String> onHouse;
  final VoidCallback onSettings;
  final Future<void> Function() onLogout;

  @override
  Widget build(BuildContext context) {
    final onDash = currentPath.startsWith('/dashboard');
    final onSettingsPath = currentPath.startsWith('/settings');

    return Container(
      margin: const EdgeInsets.all(12),
      padding: const EdgeInsets.fromLTRB(14, 18, 14, 14),
      decoration: BoxDecoration(
        color: RoomiesPalette.of(context).surface.withValues(alpha: 0.94),
        borderRadius: BorderRadius.circular(22),
        border: Border.all(color: RoomiesPalette.of(context).line),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(
            strings.brand,
            style: Theme.of(context).textTheme.headlineSmall?.copyWith(
                  color: RoomiesPalette.of(context).tealDeep,
                ),
          ),
          const SizedBox(height: 4),
          Text(
            strings.navigation,
            style: Theme.of(context).textTheme.bodyMedium,
          ),
          const SizedBox(height: 18),
          _NavTile(
            selected: onDash,
            icon: Icons.grid_view_rounded,
            label: strings.dashboard,
            onTap: onDashboard,
          ),
          const SizedBox(height: 8),
          Text(
            strings.houses,
            style: Theme.of(context).textTheme.titleMedium,
          ),
          const SizedBox(height: 6),
          Expanded(
            child: houses.isEmpty
                ? Text(
                    strings.noHousesYet,
                    style: Theme.of(context).textTheme.bodyMedium,
                  )
                : ListView.separated(
                    itemCount: houses.length,
                    separatorBuilder: (_, __) => const SizedBox(height: 4),
                    itemBuilder: (context, index) {
                      final house = houses[index];
                      final selected = house.id == currentHouseId;
                      final isDefault = house.id == defaultHouseId;
                      return _NavTile(
                        selected: selected,
                        icon: Icons.home_outlined,
                        label: house.name,
                        badge: isDefault ? strings.defaultBadge : null,
                        onTap: () => onHouse(house.id),
                      );
                    },
                  ),
          ),
          const Divider(height: 24),
          _NavTile(
            selected: onSettingsPath,
            icon: Icons.settings_outlined,
            label: strings.settings,
            onTap: onSettings,
          ),
          _NavTile(
            selected: false,
            icon: Icons.logout_rounded,
            label: strings.logout,
            onTap: () {
              onLogout();
            },
          ),
        ],
      ),
    );
  }
}

class _NavTile extends StatelessWidget {
  const _NavTile({
    required this.selected,
    required this.icon,
    required this.label,
    required this.onTap,
    this.badge,
  });

  final bool selected;
  final IconData icon;
  final String label;
  final VoidCallback onTap;
  final String? badge;

  @override
  Widget build(BuildContext context) {
    return Semantics(
      button: true,
      label: label,
      excludeSemantics: true,
      child: Material(
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
                  icon,
                  size: 20,
                  color:
                      selected ? RoomiesPalette.of(context).tealDeep : RoomiesPalette.of(context).inkMuted,
                ),
                const SizedBox(width: 10),
                Expanded(
                  child: Text(
                    label,
                    overflow: TextOverflow.ellipsis,
                    style: Theme.of(context).textTheme.titleMedium?.copyWith(
                          color: selected
                              ? RoomiesPalette.of(context).tealDeep
                              : RoomiesPalette.of(context).ink,
                          fontWeight:
                              selected ? FontWeight.w700 : FontWeight.w500,
                        ),
                  ),
                ),
                if (badge != null)
                  Container(
                    padding:
                        const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                    decoration: BoxDecoration(
                      color: RoomiesPalette.of(context).mist,
                      borderRadius: BorderRadius.circular(999),
                    ),
                    child: Text(
                      badge!,
                      style: Theme.of(context).textTheme.bodySmall?.copyWith(
                            color: RoomiesPalette.of(context).tealDeep,
                            fontWeight: FontWeight.w600,
                          ),
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
