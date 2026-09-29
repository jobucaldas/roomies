import 'package:flutter/material.dart';

/// Matches Dioxus `.card` / `article.card` layout for Playwright and visual parity.
class RoomiesCard extends StatelessWidget {
  const RoomiesCard({super.key, required this.child, this.semanticLabel});

  final Widget child;
  final String? semanticLabel;

  @override
  Widget build(BuildContext context) {
    return Semantics(
      container: true,
      label: semanticLabel,
      child: Container(
        width: double.infinity,
        margin: const EdgeInsets.symmetric(vertical: 10),
        padding: const EdgeInsets.all(16),
        decoration: BoxDecoration(
          color: Colors.white,
          borderRadius: BorderRadius.circular(8),
          boxShadow: [
            BoxShadow(
              color: Colors.black.withValues(alpha: 0.1),
              blurRadius: 3,
              offset: const Offset(0, 1),
            ),
          ],
        ),
        child: child,
      ),
    );
  }
}

class RoomiesArticleCard extends RoomiesCard {
  const RoomiesArticleCard({super.key, required super.child, super.semanticLabel});
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
        padding: const EdgeInsets.all(12),
        margin: const EdgeInsets.symmetric(vertical: 8),
        decoration: BoxDecoration(
          color: const Color(0xFFFDE8E8),
          borderRadius: BorderRadius.circular(4),
        ),
        child: Text(message, style: const TextStyle(color: Color(0xFFC53030))),
      ),
    );
  }
}

class RoomiesPage extends StatelessWidget {
  const RoomiesPage({super.key, required this.child});

  final Widget child;

  @override
  Widget build(BuildContext context) {
    return ColoredBox(
      color: const Color(0xFFF0F2F5),
      child: SafeArea(
        child: Center(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 960),
            child: Padding(
              padding: const EdgeInsets.all(20),
              child: child,
            ),
          ),
        ),
      ),
    );
  }
}

class RoomiesHeading extends StatelessWidget {
  const RoomiesHeading(this.text, {super.key, this.level = 1});

  final String text;
  final int level;

  @override
  Widget build(BuildContext context) {
    final style = switch (level) {
      1 => const TextStyle(
          fontSize: 28,
          fontWeight: FontWeight.bold,
          color: Color(0xFF1A73E8),
        ),
      2 => const TextStyle(
          fontSize: 22,
          fontWeight: FontWeight.w600,
          color: Color(0xFF444444),
        ),
      3 => const TextStyle(
          fontSize: 18,
          fontWeight: FontWeight.w600,
          color: Color(0xFF555555),
        ),
      _ => const TextStyle(fontSize: 16),
    };
    return Semantics(
      header: true,
      child: Padding(
        padding: const EdgeInsets.only(bottom: 8, top: 8),
        child: Text(text, style: style),
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
      padding: const EdgeInsets.symmetric(vertical: 4, horizontal: 4),
      child: ElevatedButton(
        onPressed: enabled ? onPressed : null,
        style: ElevatedButton.styleFrom(
          backgroundColor: const Color(0xFF1A73E8),
          foregroundColor: Colors.white,
          disabledBackgroundColor: const Color(0xFFCCCCCC),
        ),
        child: Text(label),
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
    return Semantics(
      label: label,
      textField: child is TextField || child is TextFormField,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(label, style: const TextStyle(fontWeight: FontWeight.w500)),
          const SizedBox(height: 4),
          child,
          const SizedBox(height: 8),
        ],
      ),
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
                    child: TextButton(
                      style: TextButton.styleFrom(
                        backgroundColor: selected == tab
                            ? const Color(0xFF1A73E8)
                            : const Color(0xFFE0E0E0),
                        foregroundColor: selected == tab
                            ? Colors.white
                            : const Color(0xFF555555),
                        shape: RoundedRectangleBorder(
                          borderRadius: BorderRadius.circular(4),
                        ),
                      ),
                      onPressed: () => onSelected(tab),
                      child: Text(tab, textAlign: TextAlign.center),
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
      label: name,
      child: child,
    );
  }
}
