import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../api/api_error.dart';
import '../auth/oauth_pending.dart';
import '../state/app_state.dart';
import '../widgets/roomies_ui.dart';

class AuthCallbackScreen extends StatefulWidget {
  const AuthCallbackScreen({
    super.key,
    this.code,
    this.state,
    this.error,
  });

  final String? code;
  final String? state;
  final String? error;

  @override
  State<AuthCallbackScreen> createState() => _AuthCallbackScreenState();
}

class _AuthCallbackScreenState extends State<AuthCallbackScreen> {
  String _status = 'Completing sign-in…';
  var _failed = false;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) => _complete());
  }

  Future<void> _complete() async {
    if (widget.error != null && widget.error!.isNotEmpty) {
      setState(() {
        _failed = true;
        _status = 'Sign-in failed: ${widget.error}';
      });
      return;
    }
    final code = widget.code?.trim() ?? '';
    if (code.isEmpty) {
      setState(() {
        _failed = true;
        _status = 'Sign-in failed: missing authorization code.';
      });
      return;
    }
    final app = context.read<AppState>();
    final navigator = GoRouter.of(context);
    final pending = await loadOAuthPending();
    final expectedState = pending.state?.trim() ?? '';
    final codeVerifier = pending.codeVerifier?.trim() ?? '';
    final returnedState = widget.state?.trim() ?? '';
    if (expectedState.isEmpty ||
        codeVerifier.isEmpty ||
        returnedState.isEmpty ||
        returnedState != expectedState) {
      await clearOAuthPending();
      if (!mounted) return;
      setState(() {
        _failed = true;
        _status = 'Sign-in failed: invalid or missing OAuth state.';
      });
      return;
    }
    try {
      final auth = await app.api.completeWorkOSCallback(
        code: code,
        state: returnedState,
        codeVerifier: codeVerifier,
      );
      await clearOAuthPending();
      await app.setUser(auth.user);
      if (!mounted) return;
      final invitation = await app.api.loadPendingInvitation();
      if (invitation != null && invitation.isNotEmpty) {
        navigator.go('/accept-invitation');
      } else {
        navigator.go('/dashboard');
      }
    } on ApiError catch (error) {
      await clearOAuthPending();
      if (!mounted) return;
      setState(() {
        _failed = true;
        _status = 'Sign-in failed: ${error.message}';
      });
    } catch (error) {
      await clearOAuthPending();
      if (!mounted) return;
      setState(() {
        _failed = true;
        _status = 'Sign-in failed: $error';
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return RoomiesPage(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const RoomiesHeading('Roomies'),
          const RoomiesHeading('Signing in', level: 2),
          Semantics(liveRegion: true, child: Text(_status)),
          if (_failed)
            FilledButton(
              onPressed: () => context.go('/'),
              child: const Text('Back to login'),
            ),
        ],
      ),
    );
  }
}
