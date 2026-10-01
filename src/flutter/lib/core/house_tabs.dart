/// Stable house-tab keys for deep links (`/house/:id?tab=notes`).
abstract final class HouseTabs {
  static const expenses = 'expenses';
  static const notes = 'notes';
  static const groceries = 'groceries';
  static const chores = 'chores';
  static const calendar = 'calendar';
  static const balances = 'balances';
  static const notifications = 'notifications';
  static const members = 'members';

  static const ordered = <String>[
    expenses,
    notes,
    groceries,
    chores,
    calendar,
    balances,
    notifications,
    members,
  ];

  static int indexOf(String? raw) {
    if (raw == null || raw.isEmpty) return 0;
    final key = raw.toLowerCase().trim();
    final i = ordered.indexOf(key);
    return i < 0 ? 0 : i;
  }

  static String keyAt(int index) {
    if (index < 0 || index >= ordered.length) return expenses;
    return ordered[index];
  }
}
