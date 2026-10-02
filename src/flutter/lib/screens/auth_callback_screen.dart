import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../api/api_error.dart';
import '../state/app_state.dart';
import '../widgets/roomies_ui.dart';

class AuthCallbackScreen extends StatefulWidget {
  const AuthCallbackScreen({super.key, this.code, this.oauthState, this.error});

  final String? code;
  final String? oauthState;
  final String? error;

  @override
  State<AuthCallbackScreen> createState() => _AuthCallbackScreenState();
}

class _AuthCallbackScreenState extends State<AuthCallbackScreen> {
  String _status = '';
  var _failed = false;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) => _complete());
  }

  Future<void> _complete() async {
    final s = context.read<AppState>().strings;
    if (widget.error != null && widget.error!.isNotEmpty) {
      setState(() {
        _failed = true;
        _status = s.signInFailed(widget.error!);
      });
      return;
    }
    final code = widget.code?.trim() ?? '';
    final oauthState = widget.oauthState?.trim() ?? '';
    if (code.isEmpty || oauthState.isEmpty) {
      setState(() {
        _failed = true;
        _status = s.signInFailed(s.missingAuthCode);
      });
      return;
    }
    setState(() => _status = s.completingSignIn);
    final app = context.read<AppState>();
    final navigator = GoRouter.of(context);
    try {
      final auth = await app.api.completeWorkOSCallback(code, oauthState);
      await app.setUser(auth.user);
      if (!mounted) return;
      final pending = await app.api.loadPendingInvitation();
      if (pending != null && pending.isNotEmpty) {
        navigator.go('/accept-invitation');
      } else {
        // Fresh AuthKit login → Dashboard first.
        navigator.go('/dashboard');
      }
    } on ApiError catch (error) {
      if (!mounted) return;
      setState(() {
        _failed = true;
        _status = s.signInFailed(error.message);
      });
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _failed = true;
        _status = s.signInFailed('$error');
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
          if (_failed) ...[
            RoomiesError(_status),
            const SizedBox(height: 12),
            FilledButton(
              style: FilledButton.styleFrom(
                minimumSize: const Size.fromHeight(52),
              ),
              onPressed: () => context.go('/'),
              child: Text(s.backToLogin),
            ),
          ] else ...[
            const Center(child: CircularProgressIndicator(strokeWidth: 2.5)),
            const SizedBox(height: 16),
            Semantics(
              liveRegion: true,
              child: Text(
                _status.isEmpty ? s.completingSignIn : _status,
                textAlign: TextAlign.center,
                style: Theme.of(context).textTheme.bodyLarge,
              ),
            ),
          ],
        ],
      ),
    );
  }
}
