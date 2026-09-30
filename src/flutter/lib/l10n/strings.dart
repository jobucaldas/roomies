/// Bilingual UI strings (English + Brazilian Portuguese), matching sibling 217.
class RoomiesStrings {
  const RoomiesStrings(this.localeCode);

  /// `en` or `pt`.
  final String localeCode;

  bool get isPortuguese => localeCode == 'pt';

  static String resolveDeviceLocale([String? platformLocale]) {
    final raw = (platformLocale ?? '').toLowerCase();
    if (raw.startsWith('pt')) return 'pt';
    return 'en';
  }

  String get brand => 'Roomies';
  String get brandTagline => isPortuguese
      ? 'Casas compartilhadas, contas e tarefas mais claras.'
      : 'Shared homes, clearer money and chores.';
  String get privacyLine => isPortuguese
      ? 'Privado por padrão · tudo no escopo da casa'
      : 'Private by default · house-scoped everything';

  String get continueWorkOS =>
      isPortuguese ? 'Continuar com WorkOS' : 'Continue with WorkOS';
  String get continueWorkOSHint => isPortuguese
      ? 'Inclui Google e outras opções no AuthKit hospedado.'
      : 'Includes Google and other options on hosted AuthKit.';
  String get redirecting =>
      isPortuguese ? 'Redirecionando…' : 'Redirecting…';
  String get signInSubtitleAuthKit => isPortuguese
      ? 'Entre com WorkOS AuthKit para continuar.'
      : 'Sign in with WorkOS AuthKit to continue.';
  String get signInSubtitlePassword => isPortuguese
      ? 'Entre no espaço compartilhado da sua casa.'
      : 'Sign in to your shared house workspace.';
  String get login => isPortuguese ? 'Entrar' : 'Login';
  String get loggingIn => isPortuguese ? 'Entrando…' : 'Logging in…';
  String get register => isPortuguese ? 'Criar conta' : 'Register';
  String get registering => isPortuguese ? 'Criando…' : 'Creating…';
  String get noAccountSignUp => isPortuguese
      ? 'Não tem conta? Cadastre-se'
      : "Don't have an account? Sign up";
  String get noAccountRegister => isPortuguese
      ? 'Não tem conta? Registrar'
      : "Don't have an account? Register";
  String get haveAccountLogin => isPortuguese
      ? 'Já tem conta? Entrar'
      : 'Already have an account? Login';
  String get email => 'Email';
  String get password => isPortuguese ? 'Senha' : 'Password';
  String get name => isPortuguese ? 'Nome' : 'Name';

  String get dashboard => 'Dashboard';
  String welcomeBack(String name) => isPortuguese
      ? 'Bem-vindo de volta, $name.'
      : 'Welcome back, $name.';
  String get welcomeGuest =>
      isPortuguese ? 'Bem-vindo ao Roomies!' : 'Welcome to Roomies!';
  String get createNewHouse =>
      isPortuguese ? 'Criar nova casa' : 'Create New House';
  String get createHouseHint => isPortuguese
      ? 'Comece uma casa para despesas, tarefas e notas compartilhadas.'
      : 'Start a house for expenses, chores, and shared notes.';
  String get houseName => isPortuguese ? 'Nome da casa' : 'House name';
  String get createHouse =>
      isPortuguese ? 'Criar casa' : 'Create House';
  String get yourHouses =>
      isPortuguese ? 'Suas casas' : 'Your Houses';
  String get noHousesYet => isPortuguese
      ? 'Nenhuma casa ainda. Crie uma acima!'
      : 'No houses yet. Create one above!';
  String get viewHouse => isPortuguese ? 'Ver casa' : 'View House';
  String get defaultBadge =>
      isPortuguese ? 'Padrão' : 'Default';
  String get setAsDefault =>
      isPortuguese ? 'Definir como padrão' : 'Set as default';

  String get settings =>
      isPortuguese ? 'Configurações' : 'Settings';
  String get language => isPortuguese ? 'Idioma' : 'Language';
  String get languageSystem => isPortuguese
      ? 'Idioma do dispositivo'
      : 'Device language';
  String get languageEnglish => 'English';
  String get languagePortuguese => 'Português (Brasil)';
  String get defaultHouse =>
      isPortuguese ? 'Casa padrão' : 'Default house';
  String get defaultHouseHint => isPortuguese
      ? 'Sessões futuras abrem esta casa após o login.'
      : 'Later sessions open this house after sign-in.';
  String get noDefaultHouse => isPortuguese
      ? 'Nenhuma (abrir Dashboard)'
      : 'None (open Dashboard)';
  String get switchHouse =>
      isPortuguese ? 'Trocar de casa' : 'Switch house';
  String get houses => isPortuguese ? 'Casas' : 'Houses';
  String get logout => isPortuguese ? 'Sair' : 'Logout';
  String get back => isPortuguese ? 'Voltar' : 'Back';
  String get backToLogin =>
      isPortuguese ? 'Voltar ao login' : 'Back to login';
  String get loading => isPortuguese ? 'Carregando…' : 'Loading…';
  String get restoringSession =>
      isPortuguese ? 'Restaurando sessão…' : 'Restoring session…';
  String get signingIn => isPortuguese ? 'Entrando' : 'Signing in';
  String get completingSignIn =>
      isPortuguese ? 'Concluindo entrada…' : 'Completing sign-in…';
  String signInFailed(String detail) => isPortuguese
      ? 'Falha ao entrar: $detail'
      : 'Sign-in failed: $detail';
  String get missingAuthCode => isPortuguese
      ? 'código de autorização ausente.'
      : 'missing authorization code.';
  String get redirectingToLogin => isPortuguese
      ? 'Redirecionando para o login…'
      : 'Redirecting to login…';
  String get navigation => isPortuguese ? 'Navegação' : 'Navigation';
  String get openMenu => isPortuguese ? 'Abrir menu' : 'Open menu';
  String get closeMenu => isPortuguese ? 'Fechar menu' : 'Close menu';
  String get save => isPortuguese ? 'Salvar' : 'Save';
  String get saved => isPortuguese ? 'Salvo' : 'Saved';
  String get retry => isPortuguese ? 'Tentar de novo' : 'Retry';
  String get toggleLanguage =>
      isPortuguese ? 'English' : 'Português';

  String get tabExpenses => isPortuguese ? 'Despesas' : 'Expenses';
  String get tabNotes => isPortuguese ? 'Notas' : 'Notes';
  String get tabGroceries =>
      isPortuguese ? 'Compras' : 'Groceries';
  String get tabChores => isPortuguese ? 'Tarefas' : 'Chores';
  String get tabCalendar =>
      isPortuguese ? 'Calendário' : 'Calendar';
  String get tabChat => 'Chat';
  String get tabBalances =>
      isPortuguese ? 'Saldos' : 'Balances';
  String get tabNotifications =>
      isPortuguese ? 'Notificações / Agenda' : 'Notifications / Schedule';
  String get tabMembers =>
      isPortuguese ? 'Membros' : 'Members';

  List<String> get houseTabs => [
        tabExpenses,
        tabNotes,
        tabGroceries,
        tabChores,
        tabCalendar,
        tabChat,
        tabBalances,
        tabNotifications,
        tabMembers,
      ];
}
