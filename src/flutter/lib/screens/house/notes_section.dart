import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/roles.dart';
import '../../models/models.dart';
import '../../state/app_state.dart';
import '../../widgets/roomies_ui.dart';

class NotesSection extends StatefulWidget {
  const NotesSection({
    super.key,
    required this.houseId,
    required this.role,
    required this.userId,
  });

  final String houseId;
  final HouseRole? role;
  final String userId;

  @override
  State<NotesSection> createState() => _NotesSectionState();
}

class _NotesSectionState extends State<NotesSection> {
  List<Note> _notes = [];
  var _loading = true;
  String? _error;
  final _title = TextEditingController();
  final _content = TextEditingController();

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    _title.dispose();
    _content.dispose();
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
      final notes =
          await context.read<AppState>().api.getNotes(widget.houseId);
      if (mounted) {
        setState(() {
          _notes = notes;
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

  Future<void> _save() async {
    if (_title.text.trim().isEmpty || _content.text.trim().isEmpty) {
      setState(() => _error = 'Title and content are required');
      return;
    }
    await context.read<AppState>().api.createNote(
          widget.houseId,
          _title.text.trim(),
          _content.text.trim(),
        );
    _title.clear();
    _content.clear();
    await _load();
  }

  bool _canEdit(Note note) {
    final role = widget.role;
    if (role == null) return false;
    return canMutateNote(role, note.authorId == widget.userId);
  }

  Future<void> _delete(String id) async {
    await context.read<AppState>().api.deleteNote(widget.houseId, id);
    await _load();
  }

  Future<void> _editNote(Note note) async {
    await showDialog<void>(
      context: context,
      builder: (context) => _EditNoteDialog(
        houseId: widget.houseId,
        note: note,
        onClose: () {
          Navigator.pop(context);
          _load();
        },
      ),
    );
  }

  Future<void> _confirmDelete(String id) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Delete note?'),
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
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const RoomiesHeading('Notes', level: 2),
        if (_error != null) RoomiesError(_error!),
        if (!_canCreate)
          const Text('Your monitor role is view-only.')
        else
          RoomiesCard(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const RoomiesHeading('New note', level: 3),
                RoomiesLabeledField(
                  label: 'Title',
                  child: TextField(controller: _title),
                ),
                RoomiesLabeledField(
                  label: 'Content',
                  child: TextField(
                    controller: _content,
                    maxLines: 5,
                  ),
                ),
                RoomiesPrimaryButton(label: 'Save note', onPressed: _save),
              ],
            ),
          ),
        if (_loading)
          const Text('Loading notes…')
        else if (_notes.isEmpty)
          const Text('No notes yet.')
        else
          ..._notes.map((note) {
            return RoomiesArticleCard(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  RoomiesHeading(note.title, level: 3),
                  Text(note.content),
                  Text('By ${note.authorName} · updated ${note.updatedAt}',
                      style: const TextStyle(color: Colors.grey, fontSize: 12)),
                  if (_canEdit(note)) ...[
                    RoomiesPrimaryButton(
                      label: 'Edit',
                      onPressed: () => _editNote(note),
                    ),
                    RoomiesPrimaryButton(
                      label: 'Delete',
                      onPressed: () => _confirmDelete(note.id),
                    ),
                  ],
                ],
              ),
            );
          }),
      ],
    );
  }
}

class _EditNoteDialog extends StatefulWidget {
  const _EditNoteDialog({
    required this.houseId,
    required this.note,
    required this.onClose,
  });

  final String houseId;
  final Note note;
  final VoidCallback onClose;

  @override
  State<_EditNoteDialog> createState() => _EditNoteDialogState();
}

class _EditNoteDialogState extends State<_EditNoteDialog> {
  late final TextEditingController _title;
  late final TextEditingController _content;

  @override
  void initState() {
    super.initState();
    _title = TextEditingController(text: widget.note.title);
    _content = TextEditingController(text: widget.note.content);
  }

  @override
  void dispose() {
    _title.dispose();
    _content.dispose();
    super.dispose();
  }

  Future<void> _save() async {
    await context.read<AppState>().api.updateNote(
          widget.houseId,
          widget.note.id,
          _title.text.trim(),
          _content.text.trim(),
        );
    widget.onClose();
  }

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: const Text('Edit note'),
      content: SingleChildScrollView(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            RoomiesLabeledField(
              label: 'Title',
              child: TextField(controller: _title),
            ),
            RoomiesLabeledField(
              label: 'Content',
              child: TextField(controller: _content, maxLines: 5),
            ),
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
