import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../api/api_error.dart';
import '../auth/open_url.dart';
import '../models/models.dart';
import '../state/app_state.dart';
import '../widgets/roomies_ui.dart';

class RegisterScreen extends StatefulWidget {
  const RegisterScreen({super.key});

  @override
  State<RegisterScreen> createState() => _RegisterScreenState();
}

class _RegisterScreenState extends State<RegisterScreen> {
  final _name = TextEditingController();
  final _email = TextEditingController();
  final _password = TextEditingController();
  AuthConfig? _config;
  var _loading = false;
  var _configLoading = true;
  var _configFailed = false;
  String? _error;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) => _loadConfig());
  }

  @override
  void dispose() {
    _name.dispose();
    _email.dispose();
    _password.dispose();
    super.dispose();
  }

  Future<void> _loadConfig() async {
    setState(() {
      _configLoading = true;
      _configFailed = false;
      _error = null;
    });
    try {
      final config = await context.read<AppState>().api.getAuthConfig();
      if (!mounted) return;
      setState(() {
        _config = config;
        _configLoading = false;
      });
      if (config.authkit) {
        await _startAuthKit();
      }
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _configLoading = false;
        _configFailed = true;
        _error = describeError(
          error,
          unreachable: context.read<AppState>().strings.serverUnreachable,
        );
      });
    }
  }

  Future<void> _startAuthKit() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final url = await context
          .read<AppState>()
          .api
          .workosAuthorizeUrl(screenHint: 'sign-up');
      openExternalUrl(url);
    } on ApiError catch (error) {
      setState(() => _error = describeError(
            error,
            unreachable: context.read<AppState>().strings.serverUnreachable,
          ));
    } catch (error) {
      setState(() => _error = describeError(
            error,
            unreachable: context.read<AppState>().strings.serverUnreachable,
          ));
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _submit() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    final app = context.read<AppState>();
    final navigator = GoRouter.of(context);
    try {
      final auth =
          await app.api.register(_name.text, _email.text, _password.text);
      await app.setUser(auth.user);
      if (!mounted) return;
      final pending = await app.api.loadPendingInvitation();
      if (pending != null && pending.isNotEmpty) {
        navigator.go('/accept-invitation');
      } else {
        navigator.go('/dashboard');
      }
    } on ApiError catch (error) {
      setState(() => _error = describeError(
            error,
            unreachable: context.read<AppState>().strings.serverUnreachable,
          ));
    } catch (error) {
      setState(() => _error = describeError(
            error,
            unreachable: context.read<AppState>().strings.serverUnreachable,
          ));
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final app = context.watch<AppState>();
    final s = app.strings;
    final authkit = _config?.authkit == true;
    return RoomiesAuthFrame(
      strings: s,
      onOpenSettings: () => context.go('/settings'),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          if (_error != null) RoomiesError(_error!),
          if (_configLoading || (authkit && _loading))
            const Padding(
              padding: EdgeInsets.symmetric(vertical: 12),
              child: Center(child: CircularProgressIndicator(strokeWidth: 2.5)),
            )
          else if (_configFailed)
            OutlinedButton(onPressed: _loadConfig, child: Text(s.retry))
          else if (authkit) ...[
            FilledButton(
              style: FilledButton.styleFrom(
                minimumSize: const Size.fromHeight(52),
              ),
              onPressed: _loading ? null : _startAuthKit,
              child: Text(s.createAccount),
            ),
            const SizedBox(height: 8),
            TextButton(
              onPressed: () => context.go('/'),
              child: Text(s.haveAccountLogin),
            ),
          ] else ...[
            TextField(
              controller: _name,
              decoration: InputDecoration(labelText: s.name),
              autofillHints: const [AutofillHints.name],
            ),
            const SizedBox(height: 10),
            TextField(
              controller: _email,
              decoration: InputDecoration(labelText: s.email),
              keyboardType: TextInputType.emailAddress,
              autofillHints: const [AutofillHints.email],
            ),
            const SizedBox(height: 10),
            TextField(
              controller: _password,
              decoration: InputDecoration(labelText: s.password),
              obscureText: true,
              autofillHints: const [AutofillHints.newPassword],
              onSubmitted: (_) => _submit(),
            ),
            const SizedBox(height: 16),
            FilledButton(
              style: FilledButton.styleFrom(
                minimumSize: const Size.fromHeight(52),
              ),
              onPressed: _loading ? null : _submit,
              child: Text(_loading ? s.registering : s.createAccount),
            ),
            const SizedBox(height: 8),
            TextButton(
              onPressed: () => context.go('/'),
              child: Text(s.haveAccountLogin),
            ),
          ],
        ],
      ),
    );
  }
}
