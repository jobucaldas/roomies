import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../api/api_error.dart';
import '../auth/open_url.dart';
import '../l10n/strings.dart';
import '../models/models.dart';
import '../state/app_state.dart';
import '../widgets/roomies_ui.dart';

/// Auth entry — OAuth-first via WorkOS hosted AuthKit when configured.
///
/// Scan path: brand → one **Sign in** that hands off to hosted AuthKit
/// (which also offers sign-up). Password is collapsed and only shown when
/// AuthKit is unset (local/CI).
class LoginScreen extends StatefulWidget {
  const LoginScreen({super.key});

  @override
  State<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends State<LoginScreen> {
  final _email = TextEditingController();
  final _password = TextEditingController();
  AuthConfig? _config;
  var _loading = false;
  var _configLoading = true;
  var _configFailed = false;
  var _showPasswordForm = false;
  String? _error;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) => _loadConfig());
  }

  @override
  void dispose() {
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
        // Always start collapsed — expand only on explicit tap.
        _showPasswordForm = false;
      });
    } catch (error) {
      if (!mounted) return;
      // Don't guess a sign-in method (AuthKit may be on); offer a retry.
      setState(() {
        _configLoading = false;
        _configFailed = true;
        _showPasswordForm = false;
        _error = describeError(
          error,
          unreachable: context.read<AppState>().strings.serverUnreachable,
        );
      });
    }
  }

  Future<void> _startAuthKit({required String screenHint}) async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final url = await context
          .read<AppState>()
          .api
          .workosAuthorizeUrl(screenHint: screenHint);
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

  Future<void> _submitPassword() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    final app = context.read<AppState>();
    final navigator = GoRouter.of(context);
    try {
      final auth = await app.api.login(_email.text, _password.text);
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
    final passwordAvailable = _config?.password == true;

    return RoomiesAuthFrame(
      strings: s,
      onOpenSettings: () => context.go('/settings'),
      footer: s.privacyLine,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          if (_error != null) RoomiesError(_error!),
          if (_configLoading)
            const Padding(
              padding: EdgeInsets.symmetric(vertical: 12),
              child: Center(child: CircularProgressIndicator(strokeWidth: 2.5)),
            )
          else if (_configFailed)
            OutlinedButton(onPressed: _loadConfig, child: Text(s.retry))
          else if (authkit)
            // AuthKit's hosted page handles sign-in, sign-up and social
            // providers, so Roomies shows one entry point.
            _AuthPrimaryButton(
              label: _loading ? s.redirecting : s.signIn,
              onPressed: _loading
                  ? null
                  : () => _startAuthKit(screenHint: 'sign-in'),
            )
          else if (!_showPasswordForm)
            OutlinedButton(
              onPressed: passwordAvailable
                  ? () => setState(() => _showPasswordForm = true)
                  : null,
              child: Text(s.useEmailPassword),
            )
          else
            _MinimalPasswordForm(
              strings: s,
              email: _email,
              password: _password,
              loading: _loading,
              onSubmit: _submitPassword,
              onHide: () => setState(() => _showPasswordForm = false),
              onRegister: () => context.go('/register'),
            ),
        ],
      ),
    );
  }
}

class _AuthPrimaryButton extends StatelessWidget {
  const _AuthPrimaryButton({required this.label, required this.onPressed});

  final String label;
  final VoidCallback? onPressed;

  @override
  Widget build(BuildContext context) {
    return FilledButton(
      onPressed: onPressed,
      style: FilledButton.styleFrom(
        minimumSize: const Size.fromHeight(52),
        textStyle: Theme.of(context).textTheme.labelLarge?.copyWith(
              fontSize: 16,
            ),
      ),
      child: Text(label),
    );
  }
}

class _MinimalPasswordForm extends StatelessWidget {
  const _MinimalPasswordForm({
    required this.strings,
    required this.email,
    required this.password,
    required this.loading,
    required this.onSubmit,
    required this.onHide,
    required this.onRegister,
  });

  final RoomiesStrings strings;
  final TextEditingController email;
  final TextEditingController password;
  final bool loading;
  final VoidCallback onSubmit;
  final VoidCallback onHide;
  final VoidCallback onRegister;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        TextField(
          controller: email,
          decoration: InputDecoration(
            hintText: strings.email,
            labelText: strings.email,
          ),
          keyboardType: TextInputType.emailAddress,
          autofillHints: const [AutofillHints.email],
        ),
        const SizedBox(height: 10),
        TextField(
          controller: password,
          decoration: InputDecoration(
            hintText: strings.password,
            labelText: strings.password,
          ),
          obscureText: true,
          autofillHints: const [AutofillHints.password],
          onSubmitted: (_) => onSubmit(),
        ),
        const SizedBox(height: 14),
        _AuthPrimaryButton(
          label: loading ? strings.loggingIn : strings.login,
          onPressed: loading ? null : onSubmit,
        ),
        Row(
          children: [
            TextButton(onPressed: onHide, child: Text(strings.hideEmailPassword)),
            const Spacer(),
            TextButton(onPressed: onRegister, child: Text(strings.createAccount)),
          ],
        ),
      ],
    );
  }
}
