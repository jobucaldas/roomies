import 'package:flutter/foundation.dart';
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
  String _status = 'Checking invitation…';

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
    setState(() {
      _attempted = true;
      _loading = true;
      _status = 'Joining house…';
    });
    final app = context.read<AppState>();
    final token = await app.api.loadPendingInvitation();
    if (token == null || token.isEmpty) {
      setState(() {
        _loading = false;
        _status = 'This invitation link is missing or has already been used.';
      });
      return;
    }
    try {
      final response = await app.api.acceptInvitation(token);
      await app.api.clearPendingInvitation();
      setState(() {
        _loading = false;
        _status = 'You joined the house. Refreshing your membership…';
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
        _status = 'Unable to accept this invitation: ${error.message}';
      });
    } catch (error) {
      setState(() {
        _loading = false;
        _retryable = true;
        _status = 'Unable to accept this invitation: $error';
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return RoomiesPage(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const RoomiesHeading('Roomies invitation', level: 2),
          Semantics(
            liveRegion: true,
            child: Text(_status),
          ),
          if (_retryable)
            RoomiesPrimaryButton(
              label: _loading ? 'Retrying invitation acceptance…' : 'Retry acceptance',
              enabled: !_loading,
              onPressed: _accept,
            ),
        ],
      ),
    );
  }
}
