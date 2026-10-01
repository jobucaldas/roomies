import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../core/house_tabs.dart';
import '../l10n/strings.dart';
import '../models/models.dart';
import '../state/app_state.dart';
import '../theme/roomies_theme.dart';
import 'roomies_ui.dart';

/// Authenticated chrome: action sidebar (wide) / hamburger drawer (narrow),
/// with a compact house switcher in the top bar.
///
/// Breakpoint state is sticky across soft-keyboard height changes: depending on
/// [MediaQuery.sizeOf] alone rebuilds the whole shell when only height changes,
/// which on Flutter web remounts semantics inputs and dismisses the keyboard.
class AppShell extends StatefulWidget {
  const AppShell({
    super.key,
    required this.child,
    this.currentHouseId,
    this.onHouseSelected,
    this.title,
    this.actions = const [],
    this.showBrand = true,
    this.activeTab,
  });

  final Widget child;
  final String? currentHouseId;
  final ValueChanged<String>? onHouseSelected;
  final String? title;
  final List<Widget> actions;
  final bool showBrand;

  /// Optional house-tab key for highlighting the matching action tile.
  final String? activeTab;

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

  String? _resolveActiveHouseId(AppState app) {
    final current = widget.currentHouseId;
    if (current != null && current.isNotEmpty) {
      for (final house in app.houses) {
        if (house.id == current) return current;
      }
    }
    final fallback = app.defaultHouseId;
    if (fallback != null) {
      for (final house in app.houses) {
        if (house.id == fallback) return fallback;
      }
    }
    return app.houses.isNotEmpty ? app.houses.first.id : null;
  }

  void _selectHouse(BuildContext context, AppState app, String houseId) {
    final onSelected = widget.onHouseSelected;
    if (onSelected != null) {
      onSelected(houseId);
      return;
    }
    final path = GoRouterState.of(context).uri.path;
    if (path.startsWith('/house/')) {
      final tab = GoRouterState.of(context).uri.queryParameters['tab'];
      final q = tab == null || tab.isEmpty ? '' : '?tab=$tab';
      context.go('/house/$houseId$q');
    } else {
      context.go('/house/$houseId');
    }
  }

  void _openAction(
    BuildContext context,
    AppState app,
    RoomiesStrings s,
    String? houseId,
    String tab,
  ) {
    if (houseId == null) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(s.needHouseForAction)),
      );
      return;
    }
    context.go('/house/$houseId?tab=$tab');
  }

  Widget _buildChrome(
    BuildContext context,
    AppState app,
    RoomiesStrings s,
    bool wide,
  ) {
    final activeHouseId = _resolveActiveHouseId(app);
    final path = GoRouterState.of(context).uri.path;
    final nav = _ActionNav(
      strings: s,
      currentPath: path,
      activeTab: widget.activeTab ??
          GoRouterState.of(context).uri.queryParameters['tab'],
      onHome: () => context.go('/dashboard'),
      onAction: (tab) => _openAction(context, app, s, activeHouseId, tab),
      onSettings: () => context.go('/settings'),
      onLogout: () async {
        await app.logout();
        if (context.mounted) context.go('/');
      },
    );

    final houseSwitcher = _HouseSwitcher(
      strings: s,
      houses: app.houses,
      selectedHouseId: activeHouseId,
      defaultHouseId: app.defaultHouseId,
      onSelected: (id) => _selectHouse(context, app, id),
      onCreate: () => context.go('/dashboard?create=1'),
    );

    if (wide) {
      return Material(
        color: RoomiesPalette.of(context).canvas,
        child: RoomiesAtmosphere(
          child: SafeArea(
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                SizedBox(width: 248, child: nav),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: [
                      _ShellTopBar(
                        title: widget.title,
                        showBrand: widget.showBrand,
                        strings: s,
                        houseSwitcher: houseSwitcher,
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
                  houseSwitcher: houseSwitcher,
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
    required this.houseSwitcher,
    this.title,
    this.showBrand = true,
    this.leading,
  });

  final RoomiesStrings strings;
  final List<Widget> actions;
  final Widget houseSwitcher;
  final String? title;
  final bool showBrand;
  final Widget? leading;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(8, 8, 12, 4),
      child: Row(
        children: [
          if (leading != null) leading!,
          if (showBrand)
            Padding(
              padding: const EdgeInsets.only(right: 10),
              child: Text(
                strings.brand,
                style: Theme.of(context).textTheme.titleLarge?.copyWith(
                      color: RoomiesPalette.of(context).tealDeep,
                      fontFamily: 'Fraunces',
                    ),
              ),
            ),
          Flexible(child: houseSwitcher),
          if (title != null) ...[
            const SizedBox(width: 8),
            ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 140),
              child: Semantics(
                header: true,
                child: Text(
                  title!,
                  overflow: TextOverflow.ellipsis,
                  textAlign: TextAlign.end,
                  style: Theme.of(context).textTheme.titleSmall?.copyWith(
                        color: RoomiesPalette.of(context).inkMuted,
                      ),
                ),
              ),
            ),
          ],
          ...actions,
        ],
      ),
    );
  }
}

class _HouseSwitcher extends StatelessWidget {
  const _HouseSwitcher({
    required this.strings,
    required this.houses,
    required this.selectedHouseId,
    required this.defaultHouseId,
    required this.onSelected,
    required this.onCreate,
  });

  final RoomiesStrings strings;
  final List<House> houses;
  final String? selectedHouseId;
  final String? defaultHouseId;
  final ValueChanged<String> onSelected;
  final VoidCallback onCreate;

  @override
  Widget build(BuildContext context) {
    final p = RoomiesPalette.of(context);
    if (houses.isEmpty) {
      return Align(
        alignment: Alignment.centerLeft,
        child: TextButton.icon(
          onPressed: onCreate,
          icon: const Icon(Icons.add_home_outlined, size: 18),
          label: Text(strings.createNewHouse),
        ),
      );
    }

    House? selected;
    for (final house in houses) {
      if (house.id == selectedHouseId) {
        selected = house;
        break;
      }
    }
    selected ??= houses.first;

    return Align(
      alignment: Alignment.centerLeft,
      child: PopupMenuButton<String>(
        tooltip: strings.switchHouse,
        offset: const Offset(0, 40),
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(14)),
        onSelected: (value) {
          if (value == '__create__') {
            onCreate();
            return;
          }
          onSelected(value);
        },
        itemBuilder: (context) => [
          for (final house in houses)
            PopupMenuItem(
              value: house.id,
              child: Row(
                children: [
                  Icon(
                    house.id == selected!.id
                        ? Icons.home_rounded
                        : Icons.home_outlined,
                    size: 18,
                    color: house.id == selected.id ? p.tealDeep : p.inkMuted,
                  ),
                  const SizedBox(width: 10),
                  Expanded(
                    child: Text(
                      house.name,
                      overflow: TextOverflow.ellipsis,
                    ),
                  ),
                  if (house.id == defaultHouseId)
                    Padding(
                      padding: const EdgeInsets.only(left: 8),
                      child: Text(
                        strings.defaultBadge,
                        style: Theme.of(context).textTheme.bodySmall?.copyWith(
                              color: p.tealDeep,
                              fontWeight: FontWeight.w600,
                            ),
                      ),
                    ),
                ],
              ),
            ),
          const PopupMenuDivider(),
          PopupMenuItem(
            value: '__create__',
            child: Row(
              children: [
                Icon(Icons.add, size: 18, color: p.inkMuted),
                const SizedBox(width: 10),
                Text(strings.createNewHouse),
              ],
            ),
          ),
        ],
        child: Semantics(
          button: true,
          label: '${strings.switchHouse}: ${selected.name}',
          child: AnimatedContainer(
            duration: const Duration(milliseconds: 200),
            curve: Curves.easeOutCubic,
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
            decoration: BoxDecoration(
              color: p.surface.withValues(alpha: 0.92),
              borderRadius: BorderRadius.circular(12),
              border: Border.all(color: p.line),
            ),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Icon(Icons.home_rounded, size: 18, color: p.tealDeep),
                const SizedBox(width: 8),
                Flexible(
                  child: Text(
                    selected.name,
                    overflow: TextOverflow.ellipsis,
                    style: Theme.of(context).textTheme.titleSmall?.copyWith(
                          fontWeight: FontWeight.w600,
                        ),
                  ),
                ),
                const SizedBox(width: 4),
                Icon(Icons.expand_more_rounded, size: 18, color: p.inkMuted),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _ActionNav extends StatelessWidget {
  const _ActionNav({
    required this.strings,
    required this.currentPath,
    required this.activeTab,
    required this.onHome,
    required this.onAction,
    required this.onSettings,
    required this.onLogout,
  });

  final RoomiesStrings strings;
  final String currentPath;
  final String? activeTab;
  final VoidCallback onHome;
  final ValueChanged<String> onAction;
  final VoidCallback onSettings;
  final Future<void> Function() onLogout;

  void _tap(BuildContext context, VoidCallback action) {
    final scaffold = Scaffold.maybeOf(context);
    if (scaffold?.isDrawerOpen ?? false) {
      Navigator.of(context).pop();
    }
    action();
  }

  @override
  Widget build(BuildContext context) {
    final onDash = currentPath.startsWith('/dashboard');
    final onSettingsPath = currentPath.startsWith('/settings');
    final onHouse = currentPath.startsWith('/house/');
    final tab = activeTab;

    final actions = <_ActionItem>[
      _ActionItem(
        icon: Icons.sticky_note_2_outlined,
        label: strings.tabNotes,
        tab: HouseTabs.notes,
      ),
      _ActionItem(
        icon: Icons.shopping_basket_outlined,
        label: strings.tabGroceries,
        tab: HouseTabs.groceries,
      ),
      _ActionItem(
        icon: Icons.payments_outlined,
        label: strings.tabExpenses,
        tab: HouseTabs.expenses,
      ),
      _ActionItem(
        icon: Icons.checklist_outlined,
        label: strings.tabChores,
        tab: HouseTabs.chores,
      ),
      _ActionItem(
        icon: Icons.event_outlined,
        label: strings.tabCalendar,
        tab: HouseTabs.calendar,
      ),
      _ActionItem(
        icon: Icons.account_balance_wallet_outlined,
        label: strings.tabBalances,
        tab: HouseTabs.balances,
      ),
    ];

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
                  fontFamily: 'Fraunces',
                ),
          ),
          const SizedBox(height: 4),
          Text(
            strings.thingsToDo,
            style: Theme.of(context).textTheme.bodyMedium,
          ),
          const SizedBox(height: 16),
          _NavTile(
            selected: onDash,
            icon: Icons.grid_view_rounded,
            label: strings.home,
            onTap: () => _tap(context, onHome),
          ),
          const SizedBox(height: 10),
          Text(
            strings.thingsToDo,
            style: Theme.of(context).textTheme.titleSmall?.copyWith(
                  color: RoomiesPalette.of(context).inkMuted,
                ),
          ),
          const SizedBox(height: 6),
          Expanded(
            child: ListView.separated(
              itemCount: actions.length,
              separatorBuilder: (_, __) => const SizedBox(height: 4),
              itemBuilder: (context, index) {
                final item = actions[index];
                final selected = onHouse && tab == item.tab;
                return _NavTile(
                  selected: selected,
                  icon: item.icon,
                  label: item.label,
                  onTap: () => _tap(context, () => onAction(item.tab)),
                );
              },
            ),
          ),
          const Divider(height: 24),
          _NavTile(
            selected: onSettingsPath,
            icon: Icons.settings_outlined,
            label: strings.settings,
            onTap: () => _tap(context, onSettings),
          ),
          _NavTile(
            selected: false,
            icon: Icons.logout_rounded,
            label: strings.logout,
            onTap: () => _tap(context, () {
              onLogout();
            }),
          ),
        ],
      ),
    );
  }
}

class _ActionItem {
  const _ActionItem({
    required this.icon,
    required this.label,
    required this.tab,
  });

  final IconData icon;
  final String label;
  final String tab;
}

class _NavTile extends StatelessWidget {
  const _NavTile({
    required this.selected,
    required this.icon,
    required this.label,
    required this.onTap,
  });

  final bool selected;
  final IconData icon;
  final String label;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return Semantics(
      button: true,
      label: label,
      excludeSemantics: true,
      child: Material(
        color: selected
            ? RoomiesPalette.of(context).tealSoft
            : Colors.transparent,
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
                  color: selected
                      ? RoomiesPalette.of(context).tealDeep
                      : RoomiesPalette.of(context).inkMuted,
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
              ],
            ),
          ),
        ),
      ),
    );
  }
}
