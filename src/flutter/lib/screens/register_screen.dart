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
  String? _error;

  @override
  void initState() {
    super.initState();
    _loadConfig();
  }

  @override
  void dispose() {
    _name.dispose();
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
      if (config.authkit) {
        await _startAuthKit();
      }
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _config = AuthConfig(authkit: false, password: true);
        _configLoading = false;
        _error = error.toString();
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
      setState(() => _error = error.message);
    } catch (error) {
      setState(() => _error = error.toString());
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
    return RoomiesPage(
      maxWidth: 480,
      centered: true,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Align(
            alignment: Alignment.centerRight,
            child: TextButton(
              onPressed: app.toggleLanguageQuick,
              child: Text(s.toggleLanguage),
            ),
          ),
          RoomiesBrandMark(tagline: s.brandTagline),
          const SizedBox(height: 28),
          RoomiesCard(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                RoomiesHeading(s.register, level: 2),
                if (_error != null) RoomiesError(_error!),
                if (_configLoading || (authkit && _loading))
                  Padding(
                    padding: const EdgeInsets.symmetric(vertical: 24),
                    child: Center(child: Text(s.redirecting)),
                  )
                else if (authkit) ...[
                  Text(s.signInSubtitleAuthKit),
                  const SizedBox(height: 12),
                  RoomiesPrimaryButton(
                    label: s.continueWorkOS,
                    onPressed: _loading ? null : _startAuthKit,
                    enabled: !_loading,
                  ),
                  TextButton(
                    onPressed: () => context.go('/'),
                    child: Text(s.haveAccountLogin),
                  ),
                ] else ...[
                  TextField(
                    controller: _name,
                    decoration: InputDecoration(hintText: s.name),
                  ),
                  const SizedBox(height: 10),
                  TextField(
                    controller: _email,
                    decoration: InputDecoration(hintText: s.email),
                  ),
                  const SizedBox(height: 10),
                  TextField(
                    controller: _password,
                    decoration: InputDecoration(hintText: s.password),
                    obscureText: true,
                  ),
                  const SizedBox(height: 8),
                  RoomiesPrimaryButton(
                    label: _loading ? s.registering : s.register,
                    onPressed: _loading ? null : _submit,
                    enabled: !_loading,
                  ),
                  TextButton(
                    onPressed: () => context.go('/'),
                    child: Text(s.haveAccountLogin),
                  ),
                ],
              ],
            ),
          ),
        ],
      ),
    );
  }
}
