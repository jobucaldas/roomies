import 'dart:async';
import 'dart:js_interop';

import 'package:web/web.dart' as web;

/// Flutter 3.27 Material [TabBar] exposes tabs as generic `flt-tappable`
/// nodes (label like "Expenses Tab 1 of 9") without ARIA `role=tab`.
/// Playwright's household e2e locators require `role=tab` / `tabpanel`.
///
/// This bridge patches the semantics DOM after Flutter paints it.

List<String> _tabs = <String>[];
int _selected = 0;
web.MutationObserver? _observer;
bool _patching = false;
bool _scheduled = false;

void installHouseTabA11y(List<String> tabs) {
  _tabs = List<String>.from(tabs);
  _ensureObserver();
  _schedulePatch();
}

void updateHouseTabA11ySelection(int index) {
  _selected = index;
  _schedulePatch();
}

void uninstallHouseTabA11y() {
  _observer?.disconnect();
  _observer = null;
  _tabs = <String>[];
  _selected = 0;
}

void _ensureObserver() {
  if (_observer != null) return;
  _observer = web.MutationObserver(
    (JSArray<web.MutationRecord> _, web.MutationObserver __) {
      _schedulePatch();
    }.toJS,
  );
  final host = web.document.querySelector('flt-semantics-host');
  if (host != null) {
    _observer!.observe(
      host,
      web.MutationObserverInit(
        childList: true,
        subtree: true,
        attributes: true,
        characterData: true,
      ),
    );
  } else {
    // Semantics host appears after first frame with ensureSemantics().
    Timer(const Duration(milliseconds: 50), () {
      _ensureObserver();
      _schedulePatch();
    });
  }
}

void _schedulePatch() {
  if (_scheduled) return;
  _scheduled = true;
  web.window.requestAnimationFrame(
    (num _) {
      _scheduled = false;
      _patch();
    }.toJS,
  );
}

final RegExp _tabLabelPattern = RegExp(
  r'^(.*?)\s+Tab\s+(\d+)\s+of\s+(\d+)\s*$',
);

void _patch() {
  if (_patching || _tabs.isEmpty) return;
  _patching = true;
  try {
    web.Element? tablist;

    final nodes = web.document.querySelectorAll('flt-semantics');
    for (var i = 0; i < nodes.length; i++) {
      final el = nodes.item(i);
      if (el == null) continue;
      final htmlEl = el as web.HTMLElement;
      final text = (htmlEl.textContent ?? '').trim().replaceAll('\n', ' ');
      final match = _tabLabelPattern.firstMatch(text);
      if (match != null && htmlEl.hasAttribute('flt-tappable')) {
        final name = match.group(1)!.trim();
        final index = int.parse(match.group(2)!) - 1;
        if (!_tabs.contains(name)) continue;
        _setAttr(htmlEl, 'role', 'tab');
        _setAttr(htmlEl, 'aria-label', name);
        _setAttr(
          htmlEl,
          'aria-selected',
          index == _selected ? 'true' : 'false',
        );
        tablist ??= htmlEl.parentElement;
        continue;
      }

      final role = htmlEl.getAttribute('role');
      final label = (htmlEl.getAttribute('aria-label') ?? '')
          .split('\n')
          .first
          .trim();
      if ((role == 'group' || role == 'tabpanel') && _tabs.contains(label)) {
        _setAttr(htmlEl, 'role', 'tabpanel');
        _setAttr(htmlEl, 'aria-label', label);
      }
    }

    if (tablist != null) {
      _setAttr(tablist as web.HTMLElement, 'role', 'tablist');
    }

    _wrapArticleCards();
  } finally {
    _patching = false;
  }
}

/// Dioxus used `<article class="card">`; Flutter paints cards on canvas.
/// Wrap identified semantics nodes so Playwright `article.card` locators work.
void _wrapArticleCards() {
  final cards = web.document.querySelectorAll(
    'flt-semantics[flt-semantics-identifier="roomies-article-card"]',
  );
  for (var i = 0; i < cards.length; i++) {
    final el = cards.item(i);
    if (el == null) continue;
    final htmlEl = el as web.HTMLElement;
    final existingParent = htmlEl.parentElement;
    if (existingParent != null &&
        existingParent.tagName.toLowerCase() == 'article' &&
        existingParent.classList.contains('card')) {
      _ensureCardMetaParagraph(existingParent as web.HTMLElement, htmlEl);
      continue;
    }
    final article = web.document.createElement('article') as web.HTMLElement;
    article.classList.add('card');
    existingParent?.insertBefore(article, htmlEl);
    article.append(htmlEl);
    _ensureCardMetaParagraph(article, htmlEl);
  }
}

void _ensureCardMetaParagraph(web.HTMLElement article, web.HTMLElement source) {
  // Members e2e asserts role text via article.card >> p.
  final existing = article.querySelector(':scope > p.roomies-card-meta');
  final text = (source.textContent ?? '').trim();
  if (text.isEmpty) return;
  if (existing != null) {
    if (existing.textContent != text) existing.textContent = text;
    return;
  }
  final p = web.document.createElement('p') as web.HTMLElement;
  p.classList.add('roomies-card-meta');
  p.textContent = text;
  // Keep meta out of the accessibility tree; Flutter semantics already expose it.
  p.setAttribute('aria-hidden', 'true');
  article.append(p);
}

void _setAttr(web.HTMLElement el, String name, String value) {
  if (el.getAttribute(name) == value) return;
  el.setAttribute(name, value);
}
