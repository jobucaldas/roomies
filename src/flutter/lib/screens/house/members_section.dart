import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../models/models.dart';
import '../../state/app_state.dart';
import '../../widgets/roomies_ui.dart';

class MembersSection extends StatefulWidget {
  const MembersSection({
    super.key,
    required this.houseId,
    required this.admin,
    required this.members,
    required this.onRefresh,
  });

  final String houseId;
  final bool admin;
  final List<HouseMember> members;
  final VoidCallback onRefresh;

  @override
  State<MembersSection> createState() => _MembersSectionState();
}

class _MembersSectionState extends State<MembersSection> {
  final _userId = TextEditingController();
  var _role = 'member';
  String _status = '';
  final _pendingRoles = <String, String>{};

  final _inviteEmail = TextEditingController();
  var _inviteRole = 'member';
  var _invitesLoading = false;
  List<HouseInvitation> _invites = [];
  String? _inviteError;

  @override
  void initState() {
    super.initState();
    if (widget.admin) _loadInvites();
  }

  @override
  void dispose() {
    _userId.dispose();
    _inviteEmail.dispose();
    super.dispose();
  }

  Future<void> _loadInvites() async {
    setState(() => _invitesLoading = true);
    try {
      final invites = await context
          .read<AppState>()
          .api
          .listInvitations(widget.houseId);
      if (mounted) {
        setState(() {
          _invites = invites;
          _invitesLoading = false;
        });
      }
    } catch (error) {
      if (mounted) {
        setState(() {
          _inviteError = error.toString();
          _invitesLoading = false;
        });
      }
    }
  }

  Future<void> _addMember() async {
    if (_userId.text.trim().isEmpty) {
      setState(() => _status = 'User ID is required');
      return;
    }
    await context.read<AppState>().api.addMember(
          widget.houseId,
          _userId.text.trim(),
          _role,
        );
    setState(() => _status = 'Member added.');
    widget.onRefresh();
  }

  Future<void> _changeRole(HouseMember member, String role) async {
    await context.read<AppState>().api.updateMemberRole(
          widget.houseId,
          member.userId,
          role,
        );
    widget.onRefresh();
  }

  Future<void> _remove(HouseMember member) async {
    await context
        .read<AppState>()
        .api
        .removeMember(widget.houseId, member.userId);
    setState(() => _status = 'Member removed.');
    widget.onRefresh();
  }

  Future<void> _sendInvite() async {
    if (_inviteEmail.text.trim().isEmpty) {
      setState(() => _inviteError = 'Email is required.');
      return;
    }
    try {
      await context.read<AppState>().api.createInvitation(
            widget.houseId,
            _inviteEmail.text.trim(),
            _inviteRole,
          );
      _inviteEmail.clear();
      await _loadInvites();
    } catch (error) {
      setState(() => _inviteError = 'Unable to send invitation: $error');
    }
  }

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const RoomiesHeading('Members', level: 2),
        if (widget.admin) ...[
          Semantics(
            label: 'Email invitations',
            container: true,
            child: RoomiesCard(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  const RoomiesHeading('Invite by email', level: 3),
                  RoomiesLabeledField(
                    label: 'Email address',
                    child: TextField(controller: _inviteEmail),
                  ),
                  RoomiesLabeledField(
                    label: 'Role',
                    child: DropdownButtonFormField<String>(
                      value: _inviteRole,
                      items: const [
                        DropdownMenuItem(value: 'member', child: Text('Member')),
                        DropdownMenuItem(value: 'admin', child: Text('Admin')),
                        DropdownMenuItem(value: 'monitor', child: Text('Monitor')),
                      ],
                      onChanged: (v) => setState(() => _inviteRole = v ?? 'member'),
                    ),
                  ),
                  RoomiesPrimaryButton(
                    label: 'Send invitation',
                    onPressed: _sendInvite,
                  ),
                  if (_inviteError != null) RoomiesError(_inviteError!),
                  const RoomiesHeading('Invitation history', level: 3),
                  if (_invitesLoading)
                    const Text('Loading invitations…')
                  else if (_invites.isEmpty)
                    const Text('No invitations yet.'),
                ],
              ),
            ),
          ),
          RoomiesCard(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const RoomiesHeading('Add existing member', level: 3),
                RoomiesLabeledField(
                  label: 'User ID',
                  child: TextField(controller: _userId),
                ),
                DropdownButtonFormField<String>(
                  value: _role,
                  items: const [
                    DropdownMenuItem(value: 'admin', child: Text('Admin')),
                    DropdownMenuItem(value: 'member', child: Text('Member')),
                    DropdownMenuItem(value: 'monitor', child: Text('Monitor')),
                  ],
                  onChanged: (v) => setState(() => _role = v ?? 'member'),
                ),
                RoomiesPrimaryButton(label: 'Add member', onPressed: _addMember),
                if (_status.isNotEmpty) Text(_status),
              ],
            ),
          ),
        ],
        ...widget.members.map((member) {
          return RoomiesArticleCard(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                RoomiesHeading(member.userName, level: 3),
                Text('${member.userEmail} · ${member.role}'),
                if (widget.admin)
                  Row(
                    children: [
                      DropdownButton<String>(
                        value: _pendingRoles[member.userId] ?? member.role,
                        items: const [
                          DropdownMenuItem(value: 'admin', child: Text('Admin')),
                          DropdownMenuItem(value: 'member', child: Text('Member')),
                          DropdownMenuItem(value: 'monitor', child: Text('Monitor')),
                        ],
                        onChanged: (role) {
                          if (role != null) {
                            setState(() => _pendingRoles[member.userId] = role);
                          }
                        },
                      ),
                      RoomiesPrimaryButton(
                        label: 'Change role',
                        onPressed: () => _changeRole(
                          member,
                          _pendingRoles[member.userId] ?? member.role,
                        ),
                      ),
                      RoomiesPrimaryButton(
                        label: 'Remove',
                        onPressed: () => _remove(member),
                      ),
                    ],
                  ),
              ],
            ),
          );
        }),
      ],
    );
  }
}
