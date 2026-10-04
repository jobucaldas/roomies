import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';
import '../api/api_error.dart';
import '../state/app_state.dart';
import '../widgets/roomies_ui.dart';

class AcceptInvitationScreen extends StatefulWidget {
  const AcceptInvitationScreen({super.key, this.initialToken});

  final String? initialToken;

  @override
  State<AcceptInvitationScreen> createState() => _AcceptInvitationScreenState();
}

class _AcceptInvitationScreenState extends State<AcceptInvitationScreen> {
  var _loading = false;
  var _attempted = false;
  var _retryable = false;
  String? _status;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) => _bootstrap());
  }

  Future<void> _bootstrap() async {
    final app = context.read<AppState>();
    if (widget.initialToken != null && widget.initialToken!.isNotEmpty) {
      await app.api.savePendingInvitation(widget.initialToken!);
      if (mounted) context.go('/accept-invitation');
    }
    if (!app.api.isAuthenticated) {
      if (mounted) context.go('/');
      return;
    }
    await _accept();
  }

  Future<void> _accept() async {
    if (_attempted && !_retryable) return;
    final app = context.read<AppState>();
    setState(() {
      _attempted = true;
      _loading = true;
      _status = app.strings.joiningHouse;
    });
    final token = await app.api.loadPendingInvitation();
    if (token == null || token.isEmpty) {
      setState(() {
        _loading = false;
        _status = app.strings.invitationMissing;
      });
      return;
    }
    try {
      final response = await app.api.acceptInvitation(token);
      await app.api.clearPendingInvitation();
      setState(() {
        _loading = false;
        _status = app.strings.invitationJoined;
      });
      if (mounted) {
        context.go('/house/${response.invitation.houseId}');
      }
    } on ApiError catch (error) {
      if (error.status == 401 && mounted) {
        context.go('/');
        return;
      }
      setState(() {
        _loading = false;
        _retryable = true;
        _status = app.strings.invitationFailed(error.message);
      });
    } catch (error) {
      setState(() {
        _loading = false;
        _retryable = true;
        _status = app.strings.invitationFailed('$error');
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    final s = context.watch<AppState>().strings;
    return RoomiesAuthFrame(
      strings: s,
      showTagline: false,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          if (_loading) ...[
            const Center(child: CircularProgressIndicator(strokeWidth: 2.5)),
            const SizedBox(height: 16),
          ],
          Semantics(
            liveRegion: true,
            child: Text(
              _status ?? s.checkingInvitation,
              textAlign: TextAlign.center,
              style: Theme.of(context).textTheme.bodyLarge,
            ),
          ),
          if (_retryable) ...[
            const SizedBox(height: 16),
            RoomiesPrimaryButton(
              label: s.retryInvitation,
              enabled: !_loading,
              onPressed: _accept,
            ),
          ],
        ],
      ),
    );
  }
}
