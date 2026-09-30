import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../api/api_error.dart';
import '../../core/money.dart';
import '../../core/roles.dart';
import '../../models/models.dart';
import '../../state/app_state.dart';
import '../../widgets/roomies_ui.dart';

class ExpensesSection extends StatefulWidget {
  const ExpensesSection({
    super.key,
    required this.houseId,
    required this.role,
    required this.userId,
  });

  final String houseId;
  final HouseRole? role;
  final String userId;

  @override
  State<ExpensesSection> createState() => _ExpensesSectionState();
}

class _ExpensesSectionState extends State<ExpensesSection> {
  List<Expense> _expenses = [];
  List<HouseMember> _members = [];
  var _loading = true;
  var _showForm = false;
  String? _error;
  final _amount = TextEditingController();
  final _description = TextEditingController();
  final _category = TextEditingController();
  final _date = TextEditingController();
  var _visibility = 'shared';
  final _recipients = TextEditingController();
  final _customSplits = TextEditingController();

  @override
  void initState() {
    super.initState();
    _date.text = DateTime.now().toIso8601String().substring(0, 10);
    _load();
  }

  @override
  void dispose() {
    _amount.dispose();
    _description.dispose();
    _category.dispose();
    _date.dispose();
    _recipients.dispose();
    _customSplits.dispose();
    super.dispose();
  }

  bool get _canCreate =>
      widget.role != null && canCreateExpenseOrNote(widget.role!);

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final api = context.read<AppState>().api;
      final results = await Future.wait([
        api.getExpenses(widget.houseId),
        api.getMembers(widget.houseId),
      ]);
      if (mounted) {
        setState(() {
          _expenses = results[0] as List<Expense>;
          _members = results[1] as List<HouseMember>;
          _loading = false;
        });
      }
    } catch (error) {
      if (mounted) {
        setState(() {
          _error = error.toString();
          _loading = false;
        });
      }
    }
  }

  Future<void> _saveExpense() async {
    final cents = parseMoneyCents(_amount.text);
    if (cents == null || cents <= 0) {
      setState(() =>
          _error = 'Enter a positive amount with at most two decimals');
      return;
    }
    if (_description.text.trim().isEmpty) {
      setState(() => _error = 'Description is required');
      return;
    }
    List<Map<String, dynamic>> split = [];
    if (_customSplits.text.trim().isNotEmpty) {
      try {
        split = parseSplitEntries(_customSplits.text, cents);
      } catch (error) {
        setState(() => _error = error.toString());
        return;
      }
    }
    final visibleTo = _recipients.text
        .split(',')
        .map((s) => s.trim())
        .where((s) => s.isNotEmpty)
        .toList();
    try {
      await context.read<AppState>().api.createExpense(widget.houseId, {
        'amount': centsToApiAmount(cents),
        'description': _description.text.trim(),
        'category': _category.text.trim(),
        'date': _date.text,
        'visibility': _visibility,
        'visible_to': visibleTo,
        'split': split,
      });
      setState(() => _showForm = false);
      await _load();
    } on ApiError catch (error) {
      setState(() => _error = error.message);
    }
  }

  bool _editable(Expense expense) {
    final role = widget.role;
    if (role == null) return false;
    final owner = expense.payerId == widget.userId;
    return canMutateOwned(role, owner) || role == HouseRole.admin;
  }

  Future<void> _openDetail(Expense expense) async {
    final detail = await context
        .read<AppState>()
        .api
        .getExpense(widget.houseId, expense.id);
    if (!mounted) return;
    await showDialog<void>(
      context: context,
      builder: (context) => _ExpenseDialog(
        houseId: widget.houseId,
        detail: detail,
        members: _members,
        onClose: () {
          Navigator.pop(context);
          _load();
        },
      ),
    );
  }

  Future<void> _delete(String id) async {
    await context.read<AppState>().api.deleteExpense(widget.houseId, id);
    await _load();
  }

  Future<void> _confirmDelete(String id) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Delete expense?'),
        content: const Text('This cannot be undone.'),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: const Text('Cancel'),
          ),
          TextButton(
            onPressed: () => Navigator.pop(context, true),
            child: const Text('Confirm delete'),
          ),
        ],
      ),
    );
    if (confirmed == true) await _delete(id);
  }

  @override
  Widget build(BuildContext context) {
    return Semantics(
      liveRegion: true,
      container: true,
      explicitChildNodes: true,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const RoomiesHeading('Expenses', level: 2),
          if (_error != null) RoomiesError(_error!),
          if (!_canCreate)
            const Text('Your monitor role is view-only.'),
          if (_canCreate) ...[
            RoomiesPrimaryButton(
              label: _showForm ? 'Cancel' : 'Add expense',
              onPressed: () => setState(() => _showForm = !_showForm),
            ),
            if (_showForm)
              RoomiesCard(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    const RoomiesHeading('New expense', level: 3),
                    RoomiesLabeledField(
                      label: 'Amount',
                      child: TextField(controller: _amount),
                    ),
                    RoomiesLabeledField(
                      label: 'Description',
                      child: TextField(controller: _description),
                    ),
                    RoomiesLabeledField(
                      label: 'Category (optional)',
                      child: TextField(controller: _category),
                    ),
                    RoomiesLabeledField(
                      label: 'Date',
                      child: TextField(controller: _date),
                    ),
                    RoomiesLabeledField(
                      label: 'Visibility',
                      child: DropdownButtonFormField<String>(
                        value: _visibility,
                        items: const [
                          DropdownMenuItem(value: 'shared', child: Text('Shared')),
                          DropdownMenuItem(value: 'private', child: Text('Private')),
                        ],
                        onChanged: (v) => setState(() => _visibility = v ?? 'shared'),
                      ),
                    ),
                    if (_visibility == 'private')
                      RoomiesLabeledField(
                        label:
                            'Recipient user IDs (comma separated; payer is always included)',
                        child: TextField(controller: _recipients),
                      ),
                    RoomiesLabeledField(
                      label:
                          'Custom splits (optional: user-id:12.34, user-id:5.00). Leave empty to split equally among active members.',
                      child: TextField(controller: _customSplits),
                    ),
                    RoomiesPrimaryButton(
                      label: 'Save expense',
                      onPressed: _saveExpense,
                    ),
                  ],
                ),
              ),
          ],
          if (_loading)
            const Text('Loading expenses…')
          else if (_expenses.isEmpty)
            const Text('No expenses yet.')
          else
            ..._expenses.map((expense) {
              return RoomiesArticleCard(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    RoomiesHeading(expense.description, level: 3),
                    Text(
                      '${expense.date} · \$${expense.amount.toStringAsFixed(2)} · ${expense.payerName} · ${expense.visibility}',
                    ),
                    if (_editable(expense)) ...[
                      RoomiesPrimaryButton(
                        label: 'Details / edit',
                        onPressed: () => _openDetail(expense),
                      ),
                      RoomiesPrimaryButton(
                        label: 'Delete',
                        onPressed: () => _confirmDelete(expense.id),
                      ),
                    ],
                  ],
                ),
              );
            }),
        ],
      ),
    );
  }
}

class _ExpenseDialog extends StatefulWidget {
  const _ExpenseDialog({
    required this.houseId,
    required this.detail,
    required this.members,
    required this.onClose,
  });

  final String houseId;
  final ExpenseDetail detail;
  final List<HouseMember> members;
  final VoidCallback onClose;

  @override
  State<_ExpenseDialog> createState() => _ExpenseDialogState();
}

class _ExpenseDialogState extends State<_ExpenseDialog> {
  late final TextEditingController _description;
  String? _error;

  @override
  void initState() {
    super.initState();
    _description =
        TextEditingController(text: widget.detail.expense.description);
  }

  @override
  void dispose() {
    _description.dispose();
    super.dispose();
  }

  Future<void> _save() async {
    try {
      await context.read<AppState>().api.updateExpense(
            widget.houseId,
            widget.detail.expense.id,
            {'description': _description.text.trim()},
          );
      widget.onClose();
    } on ApiError catch (error) {
      setState(() => _error = error.message);
    }
  }

  @override
  Widget build(BuildContext context) {
    final activeIds =
        widget.members.map((m) => m.userId).join(', ');
    return AlertDialog(
      title: const Text('Expense details'),
      content: SingleChildScrollView(
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          mainAxisSize: MainAxisSize.min,
          children: [
            if (_error != null) RoomiesError(_error!),
            RoomiesLabeledField(
              label: 'Description',
              child: TextField(controller: _description),
            ),
            Text('Active members: $activeIds'),
          ],
        ),
      ),
      actions: [
        TextButton(onPressed: _save, child: const Text('Save changes')),
        TextButton(onPressed: widget.onClose, child: const Text('Close')),
      ],
    );
  }
}
