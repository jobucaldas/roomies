import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../api/api_error.dart';
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
  var _loading = false;
  String? _error;

  @override
  void dispose() {
    _email.dispose();
    _password.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
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
    return RoomiesPage(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const RoomiesHeading('Roomies'),
          const RoomiesHeading('Login', level: 2),
          if (_error != null) RoomiesError(_error!),
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
          RoomiesPrimaryButton(
            label: _loading ? 'Logging in...' : 'Login',
            enabled: !_loading,
            onPressed: _submit,
          ),
          TextButton(
            onPressed: () => context.go('/register'),
            child: const Text("Don't have an account? Register"),
          ),
        ],
      ),
    );
  }
}
