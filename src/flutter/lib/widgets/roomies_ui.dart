import 'package:flutter/material.dart';

import '../l10n/strings.dart';
import '../theme/roomies_theme.dart';

/// Outlined surface card used across house sections and the dashboard.
class RoomiesCard extends StatelessWidget {
  const RoomiesCard({super.key, required this.child, this.semanticLabel});

  final Widget child;
  final String? semanticLabel;

  @override
  Widget build(BuildContext context) {
    return Semantics(
      container: true,
      explicitChildNodes: true,
      label: semanticLabel,
      child: AnimatedContainer(
        duration: const Duration(milliseconds: 220),
        curve: Curves.easeOutCubic,
        width: double.infinity,
        margin: const EdgeInsets.symmetric(vertical: 10),
        padding: const EdgeInsets.all(20),
        decoration: BoxDecoration(
          color: RoomiesPalette.of(context).surface,
          borderRadius: BorderRadius.circular(18),
          border: Border.all(color: RoomiesPalette.of(context).line),
          boxShadow: [
            BoxShadow(
              color: RoomiesPalette.of(context).shadow,
              blurRadius: 12,
              offset: const Offset(0, 2),
            ),
          ],
        ),
        child: child,
      ),
    );
  }
}

class RoomiesArticleCard extends RoomiesCard {
  const RoomiesArticleCard({
    super.key,
    required super.child,
    super.semanticLabel,
  });
}

class RoomiesError extends StatelessWidget {
  const RoomiesError(this.message, {super.key});

  final String message;

  @override
  Widget build(BuildContext context) {
    return Semantics(
      liveRegion: true,
      child: Container(
        width: double.infinity,
        padding: const EdgeInsets.all(14),
        margin: const EdgeInsets.symmetric(vertical: 8),
        decoration: BoxDecoration(
          color: RoomiesPalette.of(context).dangerSoft,
          borderRadius: BorderRadius.circular(12),
          border: Border.all(
            color: RoomiesPalette.of(context).danger.withValues(alpha: 0.35),
          ),
        ),
        child: Text(
          message,
          style: Theme.of(context).textTheme.bodyLarge?.copyWith(
                color: RoomiesPalette.of(context).danger,
              ),
        ),
      ),
    );
  }
}

/// Page backdrop — flat brand canvas only (no decorative blobs/circles).
class RoomiesAtmosphere extends StatelessWidget {
  const RoomiesAtmosphere({super.key, required this.child});

  final Widget child;

  @override
  Widget build(BuildContext context) {
    final p = RoomiesPalette.of(context);
    return ColoredBox(
      color: p.canvas,
      child: child,
    );
  }
}

class RoomiesPage extends StatelessWidget {
  const RoomiesPage({
    super.key,
    required this.child,
    this.maxWidth = 960,
    this.atmosphere = true,
    this.centered = false,
  });

  final Widget child;
  final double maxWidth;
  final bool atmosphere;
  final bool centered;

  @override
  Widget build(BuildContext context) {
    // Material ancestor is required for TextField/InputDecorator; GoRouter
    // route builders do not wrap pages in Scaffold by default.
    final content = SafeArea(
      child: Center(
        child: ConstrainedBox(
          constraints: BoxConstraints(maxWidth: maxWidth),
          child: SingleChildScrollView(
            padding: EdgeInsets.symmetric(
              horizontal: centered ? 24 : 20,
              vertical: centered ? 48 : 24,
            ),
            // Translate-only entrance: avoid opacity:0 which Playwright treats
            // as not visible while Flutter web is still settling.
            child: TweenAnimationBuilder<double>(
              tween: Tween(begin: 0, end: 1),
              duration: const Duration(milliseconds: 320),
              curve: Curves.easeOutCubic,
              builder: (context, value, animatedChild) {
                return Transform.translate(
                  offset: Offset(0, (1 - value) * 10),
                  child: animatedChild,
                );
              },
              child: child,
            ),
          ),
        ),
      ),
    );

    return Material(
      color: RoomiesPalette.of(context).canvas,
      child: atmosphere ? RoomiesAtmosphere(child: content) : content,
    );
  }
}

class RoomiesHeading extends StatelessWidget {
  const RoomiesHeading(this.text, {super.key, this.level = 1});

  final String text;
  final int level;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context).textTheme;
    final style = switch (level) {
      1 => theme.headlineLarge,
      2 => theme.headlineMedium,
      3 => theme.titleLarge,
      _ => theme.bodyLarge,
    };
    return Semantics(
      header: true,
      child: Padding(
        padding: EdgeInsets.only(
          bottom: level == 1 ? 12 : 8,
          top: level == 1 ? 4 : 8,
        ),
        child: Text(text, style: style),
      ),
    );
  }
}

/// The "Roomies" wordmark. Every surface that shows the brand (boot splash,
/// sign-in, sidebar, drawer) goes through this so the typeface never drifts.
class RoomiesWordmark extends StatelessWidget {
  const RoomiesWordmark({super.key, this.size = 24});

  final double size;

  @override
  Widget build(BuildContext context) {
    return Text(
      'Roomies',
      style: TextStyle(
        fontFamily: roomiesDisplayFamily,
        fontWeight: FontWeight.w700,
        fontSize: size,
        height: 1.0,
        letterSpacing: -0.02 * size,
        color: RoomiesPalette.of(context).tealDeep,
      ),
    );
  }
}

/// Wordmark plus optional tagline for sign-in and callback screens.
class RoomiesBrandMark extends StatelessWidget {
  const RoomiesBrandMark({
    super.key,
    this.compact = false,
    this.centered = false,
    this.tagline,
  });

  final bool compact;
  final bool centered;
  final String? tagline;

  @override
  Widget build(BuildContext context) {
    return Semantics(
      header: true,
      child: Column(
        crossAxisAlignment:
            centered ? CrossAxisAlignment.center : CrossAxisAlignment.start,
        children: [
          RoomiesWordmark(size: compact ? 34 : 52),
          if (!compact && tagline != null) ...[
            const SizedBox(height: 14),
            Text(
              tagline!,
              textAlign: centered ? TextAlign.center : TextAlign.start,
              style: Theme.of(context).textTheme.bodyLarge?.copyWith(
                    color: RoomiesPalette.of(context).inkMuted,
                  ),
            ),
          ],
        ],
      ),
    );
  }
}

class RoomiesPrimaryButton extends StatelessWidget {
  const RoomiesPrimaryButton({
    super.key,
    required this.label,
    required this.onPressed,
    this.enabled = true,
  });

  final String label;
  final VoidCallback? onPressed;
  final bool enabled;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 8),
      child: SizedBox(
        width: double.infinity,
        child: FilledButton(
          onPressed: enabled ? onPressed : null,
          style: FilledButton.styleFrom(
            minimumSize: const Size.fromHeight(48),
          ),
          child: Text(label),
        ),
      ),
    );
  }
}

class RoomiesLabeledField extends StatelessWidget {
  const RoomiesLabeledField({
    super.key,
    required this.label,
    required this.child,
  });

  final String label;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        ExcludeSemantics(
          child: Text(
            label,
            style: Theme.of(context).textTheme.titleMedium,
          ),
        ),
        const SizedBox(height: 6),
        // One text field node. A parent textField semantics node is a second,
        // disabled input and Playwright cannot tell them apart.
        Semantics(
          label: label,
          child: child,
        ),
        const SizedBox(height: 10),
      ],
    );
  }
}

class RoomiesTabPanel extends StatelessWidget {
  const RoomiesTabPanel({
    super.key,
    required this.name,
    required this.child,
  });

  final String name;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    return Semantics(
      container: true,
      explicitChildNodes: true,
      label: name,
      child: child,
    );
  }
}

/// Shared layout for signed-out screens (sign-in, register, AuthKit callback,
/// invitation): centered wordmark, tagline and one block of actions, so the
/// hand-off to and from hosted AuthKit looks like one flow.
class RoomiesAuthFrame extends StatelessWidget {
  const RoomiesAuthFrame({
    super.key,
    required this.strings,
    required this.child,
    this.onToggleLanguage,
    this.footer,
    this.showTagline = true,
  });

  final RoomiesStrings strings;
  final Widget child;
  final VoidCallback? onToggleLanguage;
  final String? footer;
  final bool showTagline;

  @override
  Widget build(BuildContext context) {
    final p = RoomiesPalette.of(context);
    return Material(
      color: p.canvas,
      child: SafeArea(
        child: Column(
          children: [
            SizedBox(
              height: 56,
              child: Align(
                alignment: Alignment.centerRight,
                child: onToggleLanguage == null
                    ? null
                    : Padding(
                        padding: const EdgeInsets.only(right: 12),
                        child: TextButton(
                          style: TextButton.styleFrom(
                            foregroundColor: p.inkMuted,
                            visualDensity: VisualDensity.compact,
                          ),
                          onPressed: onToggleLanguage,
                          child: Text(strings.toggleLanguage),
                        ),
                      ),
              ),
            ),
            Expanded(
              child: Center(
                child: SingleChildScrollView(
                  padding: const EdgeInsets.symmetric(horizontal: 28),
                  child: ConstrainedBox(
                    constraints: const BoxConstraints(maxWidth: 380),
                    child: Column(
                      mainAxisSize: MainAxisSize.min,
                      crossAxisAlignment: CrossAxisAlignment.stretch,
                      children: [
                        RoomiesBrandMark(
                          centered: true,
                          tagline: showTagline ? strings.brandTagline : null,
                        ),
                        const SizedBox(height: 36),
                        child,
                      ],
                    ),
                  ),
                ),
              ),
            ),
            SizedBox(
              height: 56,
              child: footer == null
                  ? null
                  : Center(
                      child: Text(
                        footer!,
                        style: Theme.of(context).textTheme.bodySmall,
                      ),
                    ),
            ),
          ],
        ),
      ),
    );
  }
}

/// One secondary action on a list item (edit, toggle, delete…).
class RoomiesItemAction {
  const RoomiesItemAction({
    required this.label,
    required this.onPressed,
    this.destructive = false,
  });

  final String label;
  final VoidCallback? onPressed;
  final bool destructive;
}

/// Compact, right-aligned row of item actions. Primary filled buttons are
/// reserved for the one main action of a section (add, save).
class RoomiesItemActions extends StatelessWidget {
  const RoomiesItemActions(this.actions, {super.key});

  final List<RoomiesItemAction> actions;

  @override
  Widget build(BuildContext context) {
    final p = RoomiesPalette.of(context);
    return Padding(
      padding: const EdgeInsets.only(top: 8),
      child: Wrap(
        alignment: WrapAlignment.end,
        spacing: 8,
        runSpacing: 4,
        children: [
          for (final action in actions)
            action.destructive
                ? TextButton(
                    style: TextButton.styleFrom(foregroundColor: p.danger),
                    onPressed: action.onPressed,
                    child: Text(action.label),
                  )
                : OutlinedButton(
                    style: OutlinedButton.styleFrom(
                      minimumSize: const Size(48, 40),
                      padding: const EdgeInsets.symmetric(horizontal: 16),
                    ),
                    onPressed: action.onPressed,
                    child: Text(action.label),
                  ),
        ],
      ),
    );
  }
}
