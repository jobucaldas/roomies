import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../l10n/strings.dart';
import '../../models/models.dart';
import '../../state/app_state.dart';
import '../../theme/roomies_theme.dart';
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
      setState(() => _status = context.read<AppState>().strings.userIdRequired);
      return;
    }
    await context.read<AppState>().api.addMember(
          widget.houseId,
          _userId.text.trim(),
          _role,
        );
    setState(() => _status = context.read<AppState>().strings.memberAdded);
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
    setState(() => _status = context.read<AppState>().strings.memberRemoved);
    widget.onRefresh();
  }

  Future<void> _sendInvite() async {
    if (_inviteEmail.text.trim().isEmpty) {
      setState(() => _inviteError = context.read<AppState>().strings.emailRequired);
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
      setState(() => _inviteError = context.read<AppState>().strings.inviteFailed('$error'));
    }
  }

  String _roleLabel(RoomiesStrings s, String role) => switch (role) {
        'admin' => s.roleAdmin,
        'monitor' => s.roleMonitor,
        _ => s.roleMember,
      };

  List<DropdownMenuItem<String>> _roleItems(RoomiesStrings s) => [
        DropdownMenuItem(value: 'member', child: Text(s.roleMember)),
        DropdownMenuItem(value: 'admin', child: Text(s.roleAdmin)),
        DropdownMenuItem(value: 'monitor', child: Text(s.roleMonitor)),
      ];

  @override
  Widget build(BuildContext context) {
    final s = context.watch<AppState>().strings;
    return RoomiesTabPanel(
      name: s.tabMembers,
      child: Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        RoomiesHeading(s.tabMembers, level: 2),
        if (widget.admin) ...[
          Semantics(
            label: s.emailInvitations,
            container: true,
            explicitChildNodes: true,
            child: RoomiesCard(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  RoomiesHeading(s.inviteByEmail, level: 3),
                  RoomiesLabeledField(
                    label: s.emailAddress,
                    child: TextField(controller: _inviteEmail),
                  ),
                  RoomiesLabeledField(
                    label: s.role,
                    child: DropdownButtonFormField<String>(
                      value: _inviteRole,
                      items: _roleItems(s),
                      onChanged: (v) => setState(() => _inviteRole = v ?? 'member'),
                    ),
                  ),
                  RoomiesPrimaryButton(
                    label: s.sendInvitation,
                    onPressed: _sendInvite,
                  ),
                  if (_inviteError != null) RoomiesError(_inviteError!),
                  RoomiesHeading(s.invitationHistory, level: 3),
                  if (_invitesLoading)
                    Text(s.loadingInvitations)
                  else if (_invites.isEmpty)
                    Text(s.noInvitationsYet),
                ],
              ),
            ),
          ),
          RoomiesCard(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                RoomiesHeading(s.addExistingMember, level: 3),
                RoomiesLabeledField(
                  label: s.userId,
                  child: TextField(controller: _userId),
                ),
                DropdownButtonFormField<String>(
                  value: _role,
                  items: _roleItems(s),
                  onChanged: (v) => setState(() => _role = v ?? 'member'),
                ),
                RoomiesPrimaryButton(label: s.addMember, onPressed: _addMember),
                if (_status.isNotEmpty) Text(_status),
              ],
            ),
          ),
        ],
        ...widget.members.map((member) {
          return RoomiesArticleCard(
            semanticLabel: member.userName,
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                RoomiesHeading(member.userName, level: 3),
                Text('${member.userEmail} · ${_roleLabel(s, member.role)}'),
                if (widget.admin)
                  Wrap(
                    crossAxisAlignment: WrapCrossAlignment.center,
                    spacing: 12,
                    runSpacing: 8,
                    children: [
                      DropdownButton<String>(
                        value: _pendingRoles[member.userId] ?? member.role,
                        items: _roleItems(s),
                        onChanged: (role) {
                          if (role != null) {
                            setState(() => _pendingRoles[member.userId] = role);
                          }
                        },
                      ),
                      TextButton(
                        onPressed: () => _changeRole(
                          member,
                          _pendingRoles[member.userId] ?? member.role,
                        ),
                        child: Text(s.changeRole),
                      ),
                      TextButton(
                        style: TextButton.styleFrom(
                          foregroundColor: RoomiesPalette.of(context).danger,
                        ),
                        onPressed: () => _remove(member),
                        child: Text(s.remove),
                      ),
                    ],
                  ),
              ],
            ),
          );
        }),
      ],
      ),
    );
  }
}
