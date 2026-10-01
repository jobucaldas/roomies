import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../api/api_error.dart';
import '../auth/open_url.dart';
import '../l10n/strings.dart';
import '../models/models.dart';
import '../state/app_state.dart';
import '../theme/roomies_theme.dart';
import '../widgets/roomies_ui.dart';

/// Auth entry — OAuth-first via WorkOS hosted AuthKit when configured.
///
/// Scan path: brand → primary **Sign in** → optional create-account link.
/// Password is collapsed and only shown when AuthKit is unset (local/CI).
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
  var _showPasswordForm = false;
  String? _error;

  @override
  void initState() {
    super.initState();
    _loadConfig();
  }

  @override
  void dispose() {
    _email.dispose();
    _password.dispose();
    super.dispose();
  }

  Future<void> _loadConfig() async {
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
      setState(() {
        _config = AuthConfig(authkit: false, password: true);
        _configLoading = false;
        _showPasswordForm = false;
        _error = error.toString();
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
      setState(() => _error = error.message);
    } catch (error) {
      setState(() => _error = error.toString());
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
      setState(() => _error = error.message);
    } catch (error) {
      setState(() => _error = error.toString());
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

    return RoomiesPage(
      maxWidth: 400,
      centered: true,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Align(
            alignment: Alignment.centerRight,
            child: TextButton(
              style: TextButton.styleFrom(
                foregroundColor: RoomiesPalette.of(context).inkMuted,
                visualDensity: VisualDensity.compact,
                padding: const EdgeInsets.symmetric(horizontal: 8),
              ),
              onPressed: app.toggleLanguageQuick,
              child: Text(s.toggleLanguage),
            ),
          ),
          const SizedBox(height: 16),
          RoomiesBrandMark(tagline: s.brandTagline),
          const SizedBox(height: 36),
          if (_error != null) RoomiesError(_error!),
          if (_configLoading)
            const Padding(
              padding: EdgeInsets.symmetric(vertical: 40),
              child: Center(child: CircularProgressIndicator(strokeWidth: 2.5)),
            )
          else if (authkit) ...[
            _AuthPrimaryButton(
              label: _loading ? s.redirecting : s.signIn,
              onPressed: _loading
                  ? null
                  : () => _startAuthKit(screenHint: 'sign-in'),
            ),
            const SizedBox(height: 4),
            Center(
              child: TextButton(
                onPressed: _loading
                    ? null
                    : () => _startAuthKit(screenHint: 'sign-up'),
                child: Text(s.createAccount),
              ),
            ),
          ] else ...[
            if (!_showPasswordForm)
              OutlinedButton(
                onPressed: passwordAvailable
                    ? () => setState(() => _showPasswordForm = true)
                    : null,
                style: OutlinedButton.styleFrom(
                  foregroundColor: RoomiesPalette.of(context).ink,
                  side: BorderSide(color: RoomiesPalette.of(context).line),
                  padding:
                      const EdgeInsets.symmetric(horizontal: 18, vertical: 14),
                  shape: RoundedRectangleBorder(
                    borderRadius: BorderRadius.circular(12),
                  ),
                ),
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
          const SizedBox(height: 48),
          Text(
            s.privacyLine,
            textAlign: TextAlign.center,
            style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                  color: RoomiesPalette.of(context).inkMuted,
                  fontSize: 12,
                ),
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
    return SizedBox(
      width: double.infinity,
      height: 48,
      child: FilledButton(
        onPressed: onPressed,
        style: FilledButton.styleFrom(
          backgroundColor: RoomiesPalette.of(context).teal,
          foregroundColor: Theme.of(context).colorScheme.onPrimary,
          minimumSize: const Size.fromHeight(52),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(14),
          ),
          textStyle: Theme.of(context).textTheme.labelLarge,
        ),
        child: Text(label),
      ),
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
