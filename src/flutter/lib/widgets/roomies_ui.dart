import 'package:flutter/material.dart';

import '../theme/roomies_theme.dart';

/// Matches Dioxus `.card` / `article.card` layout for Playwright and visual parity.
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
          borderRadius: BorderRadius.circular(20),
          border: Border.all(color: RoomiesPalette.of(context).line),
          boxShadow: [
            BoxShadow(
              color: RoomiesPalette.of(context).shadow,
              blurRadius: 28,
              offset: const Offset(0, 12),
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

class RoomiesAtmosphere extends StatelessWidget {
  const RoomiesAtmosphere({super.key, required this.child});

  final Widget child;

  @override
  Widget build(BuildContext context) {
    final p = RoomiesPalette.of(context);
    return DecoratedBox(
      decoration: BoxDecoration(
        gradient: LinearGradient(
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
          colors: p.atmosphere,
          stops: const [0.0, 0.35, 0.7, 1.0],
        ),
      ),
      child: Stack(
        children: [
          // Keep blobs mostly off-canvas so they don't crowd top-right CTAs
          // (e.g. Open house) on CanvasKit phone web.
          Positioned(
            top: -140,
            right: -120,
            child: _Blob(size: 168, color: p.blobPrimary),
          ),
          Positioned(
            bottom: -130,
            left: -140,
            child: _Blob(size: 150, color: p.blobSecondary),
          ),
          child,
        ],
      ),
    );
  }
}

class _Blob extends StatelessWidget {
  const _Blob({required this.size, required this.color});

  final double size;
  final Color color;

  @override
  Widget build(BuildContext context) {
    return IgnorePointer(
      child: Container(
        width: size,
        height: size,
        decoration: BoxDecoration(
          shape: BoxShape.circle,
          color: color,
        ),
      ),
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

class RoomiesBrandMark extends StatelessWidget {
  const RoomiesBrandMark({
    super.key,
    this.compact = false,
    this.tagline,
  });

  final bool compact;
  final String? tagline;

  @override
  Widget build(BuildContext context) {
    final line = tagline ?? 'Shared homes, clearer money and chores.';
    return Semantics(
      header: true,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            'Roomies',
            style: Theme.of(context).textTheme.headlineLarge?.copyWith(
                  fontSize: compact ? 34 : 52,
                  height: 0.98,
                ),
          ),
          if (!compact) ...[
            const SizedBox(height: 12),
            Text(
              line,
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

class RoomiesTabStrip extends StatelessWidget {
  const RoomiesTabStrip({
    super.key,
    required this.tabs,
    required this.selected,
    required this.onSelected,
  });

  final List<String> tabs;
  final String selected;
  final ValueChanged<String> onSelected;

  @override
  Widget build(BuildContext context) {
    return Semantics(
      container: true,
      explicitChildNodes: true,
      child: LayoutBuilder(
        builder: (context, constraints) {
          final narrow = constraints.maxWidth < 600;
          final children = tabs
              .map(
                (tab) => Expanded(
                  flex: narrow ? 0 : 1,
                  child: Semantics(
                    button: true,
                    selected: selected == tab,
                    label: tab,
                    child: Padding(
                      padding: const EdgeInsets.all(2),
                      child: TextButton(
                        style: TextButton.styleFrom(
                          backgroundColor: selected == tab
                              ? RoomiesPalette.of(context).teal
                              : RoomiesPalette.of(context).mist,
                          foregroundColor: selected == tab
                              ? Theme.of(context).colorScheme.onPrimary
                              : RoomiesPalette.of(context).inkMuted,
                          shape: RoundedRectangleBorder(
                            borderRadius: BorderRadius.circular(12),
                          ),
                        ),
                        onPressed: () => onSelected(tab),
                        child: Text(tab, textAlign: TextAlign.center),
                      ),
                    ),
                  ),
                ),
              )
              .toList();
          if (narrow) {
            return Wrap(spacing: 4, runSpacing: 4, children: children);
          }
          return Row(children: children);
        },
      ),
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
