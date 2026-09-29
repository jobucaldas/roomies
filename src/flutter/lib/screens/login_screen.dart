import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../api/api_error.dart';
import '../auth/open_url.dart';
import '../models/models.dart';
import '../state/app_state.dart';
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
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const RoomiesHeading('Roomies'),
          const RoomiesHeading('Login', level: 2),
          if (_error != null) RoomiesError(_error!),
          if (_configLoading)
            const Text('Loading sign-in options…')
          else if (authkit) ...[
            const Text('Sign in with WorkOS AuthKit to continue.'),
            FilledButton(
              onPressed: _loading
                  ? null
                  : () => _startAuthKit(screenHint: 'sign-in'),
              child: Text(_loading ? 'Redirecting…' : 'Sign in with AuthKit'),
            ),
            TextButton(
              onPressed:
                  _loading ? null : () => _startAuthKit(screenHint: 'sign-up'),
              child: const Text("Don't have an account? Sign up"),
            ),
          ] else ...[
            TextField(
              controller: _email,
              decoration: const InputDecoration(hintText: 'Email'),
              keyboardType: TextInputType.emailAddress,
            ),
            TextField(
              controller: _password,
              decoration: const InputDecoration(hintText: 'Password'),
              obscureText: true,
            ),
            FilledButton(
              onPressed: _loading ? null : _submitPassword,
              child: Text(_loading ? 'Logging in...' : 'Login'),
            ),
            TextButton(
              onPressed: () => context.go('/register'),
              child: const Text("Don't have an account? Register"),
            ),
          ],
        ],
      ),
    );
  }
}
