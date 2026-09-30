import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../api/api_error.dart';
import '../auth/open_url.dart';
import '../models/models.dart';
import '../state/app_state.dart';
import '../theme/roomies_theme.dart';
import '../widgets/roomies_ui.dart';

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
      });
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _config = AuthConfig(authkit: false, password: true);
        _configLoading = false;
        _error = 'Could not load auth settings: $error';
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
    final authkit = _config?.authkit == true;
    return RoomiesPage(
      maxWidth: 480,
      centered: true,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const RoomiesBrandMark(),
          const SizedBox(height: 28),
          RoomiesCard(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const RoomiesHeading('Login', level: 2),
                Text(
                  authkit
                      ? 'Sign in with WorkOS AuthKit to continue.'
                      : 'Sign in to your shared house workspace.',
                  style: Theme.of(context).textTheme.bodyMedium,
                ),
                const SizedBox(height: 18),
                if (_error != null) RoomiesError(_error!),
                if (_configLoading)
                  const Padding(
                    padding: EdgeInsets.symmetric(vertical: 24),
                    child: Center(child: CircularProgressIndicator()),
                  )
                else if (authkit) ...[
                  RoomiesPrimaryButton(
                    label: _loading ? 'Redirecting…' : 'Sign in with AuthKit',
                    onPressed: _loading
                        ? null
                        : () => _startAuthKit(screenHint: 'sign-in'),
                    enabled: !_loading,
                  ),
                  Align(
                    alignment: Alignment.centerLeft,
                    child: TextButton(
                      onPressed: _loading
                          ? null
                          : () => _startAuthKit(screenHint: 'sign-up'),
                      child: const Text("Don't have an account? Sign up"),
                    ),
                  ),
                ] else ...[
                  Semantics(
                    label: 'Email',
                    textField: true,
                    child: TextField(
                      controller: _email,
                      decoration: const InputDecoration(
                        hintText: 'Email',
                        labelText: 'Email',
                      ),
                      keyboardType: TextInputType.emailAddress,
                      autofillHints: const [AutofillHints.email],
                    ),
                  ),
                  const SizedBox(height: 10),
                  Semantics(
                    label: 'Password',
                    textField: true,
                    child: TextField(
                      controller: _password,
                      decoration: const InputDecoration(
                        hintText: 'Password',
                        labelText: 'Password',
                      ),
                      obscureText: true,
                      autofillHints: const [AutofillHints.password],
                    ),
                  ),
                  const SizedBox(height: 8),
                  RoomiesPrimaryButton(
                    label: _loading ? 'Logging in...' : 'Login',
                    onPressed: _loading ? null : _submitPassword,
                    enabled: !_loading,
                  ),
                  TextButton(
                    onPressed: () => context.go('/register'),
                    child: const Text("Don't have an account? Register"),
                  ),
                ],
              ],
            ),
          ),
          const SizedBox(height: 18),
          Text(
            'Private by default · house-scoped everything',
            style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                  color: RoomiesColors.inkMuted,
                ),
          ),
        ],
      ),
    );
  }
}
