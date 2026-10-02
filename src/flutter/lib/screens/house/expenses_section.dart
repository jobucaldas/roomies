import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../api/api_error.dart';
import '../../core/datetime_format.dart';
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
    final s = context.read<AppState>().strings;
    final cents = parseMoneyCents(_amount.text);
    if (cents == null || cents <= 0) {
      setState(() => _error = s.positiveAmountRequired);
      return;
    }
    if (_description.text.trim().isEmpty) {
      setState(() => _error = s.descriptionRequired);
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
    final visibleTo = _visibility == 'private'
        ? _recipients.text
            .split(',')
            .map((part) => part.trim())
            .where((part) => part.isNotEmpty)
            .toList()
        : <String>[];
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
    final s = context.read<AppState>().strings;
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: Text(s.deleteExpenseTitle),
        content: Text(s.cannotUndo),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: Text(s.cancel),
          ),
          TextButton(
            onPressed: () => Navigator.pop(context, true),
            child: Text(s.confirmDelete),
          ),
        ],
      ),
    );
    if (confirmed == true) await _delete(id);
  }

  @override
  Widget build(BuildContext context) {
    final app = context.watch<AppState>();
    final s = app.strings;
    final locale = app.localeCode;
    return Semantics(
      label: s.tabExpenses,
      container: true,
      liveRegion: true,
      explicitChildNodes: true,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          RoomiesHeading(s.tabExpenses, level: 2),
          if (_error != null) RoomiesError(_error!),
          if (!_canCreate) Text(s.viewOnlyRole),
          if (_canCreate) ...[
            RoomiesPrimaryButton(
              label: _showForm ? s.cancel : s.addExpense,
              onPressed: () => setState(() => _showForm = !_showForm),
            ),
            if (_showForm)
              RoomiesCard(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    RoomiesHeading(s.newExpense, level: 3),
                    RoomiesLabeledField(
                      label: s.amount,
                      child: TextField(controller: _amount),
                    ),
                    RoomiesLabeledField(
                      label: s.description,
                      child: TextField(controller: _description),
                    ),
                    RoomiesLabeledField(
                      label: s.customSplits,
                      child: TextField(
                        controller: _customSplits,
                        decoration: const InputDecoration(
                          hintText: 'user-id:12.34, user-id:5.00',
                        ),
                      ),
                    ),
                    RoomiesLabeledField(
                      label: s.categoryOptional,
                      child: TextField(controller: _category),
                    ),
                    RoomiesLabeledField(
                      label: s.date,
                      child: TextField(controller: _date),
                    ),
                    RoomiesLabeledField(
                      label: s.visibility,
                      child: DropdownButtonFormField<String>(
                        value: _visibility,
                        items: [
                          DropdownMenuItem(
                              value: 'shared', child: Text(s.shared)),
                          DropdownMenuItem(
                              value: 'private', child: Text(s.privateLabel)),
                        ],
                        onChanged: (v) =>
                            setState(() => _visibility = v ?? 'shared'),
                      ),
                    ),
                    RoomiesLabeledField(
                      label: s.recipientUserIds,
                      child: TextField(controller: _recipients),
                    ),
                    RoomiesPrimaryButton(
                      label: s.saveExpense,
                      onPressed: _saveExpense,
                    ),
                  ],
                ),
              ),
          ],
          if (_loading)
            Text(s.loadingExpenses)
          else if (_expenses.isEmpty)
            Text(s.noExpensesYet)
          else
            ..._expenses.map((expense) {
              return RoomiesArticleCard(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    RoomiesHeading(expense.description, level: 3),
                    Text(
                      [
                        formatDisplayDate(expense.date, localeCode: locale),
                        formatMoney(expense.amount, localeCode: locale),
                        expense.payerName,
                        expense.visibility == 'private'
                            ? s.privateLabel
                            : s.shared,
                      ].join(' · '),
                    ),
                    if (_editable(expense))
                      RoomiesItemActions([
                        RoomiesItemAction(
                          label: s.detailsEdit,
                          onPressed: () => _openDetail(expense),
                        ),
                        RoomiesItemAction(
                          label: s.delete,
                          destructive: true,
                          onPressed: () => _confirmDelete(expense.id),
                        ),
                      ]),
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
    final s = context.watch<AppState>().strings;
    final activeIds = widget.members.map((m) => m.userId).join(', ');
    return AlertDialog(
      title: Text(s.expenseDetails),
      content: SingleChildScrollView(
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          mainAxisSize: MainAxisSize.min,
          children: [
            if (_error != null) RoomiesError(_error!),
            RoomiesLabeledField(
              label: s.description,
              child: TextField(controller: _description),
            ),
            Text('${s.activeMembers}: $activeIds'),
          ],
        ),
      ),
      actions: [
        TextButton(onPressed: _save, child: Text(s.saveChanges)),
        TextButton(onPressed: widget.onClose, child: Text(s.close)),
      ],
    );
  }
}
