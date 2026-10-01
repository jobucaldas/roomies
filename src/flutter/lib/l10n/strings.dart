/// Bilingual UI strings (English + Brazilian Portuguese).
/// Prefer short labels — screens should scan without reading paragraphs.
class RoomiesStrings {
  const RoomiesStrings(this.localeCode);

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
      ? 'Privado por padrão'
      : 'Private by default';

  String get signIn => isPortuguese ? 'Entrar' : 'Sign in';
  String get redirecting =>
      isPortuguese ? 'Redirecionando…' : 'Redirecting…';
  String get createAccount =>
      isPortuguese ? 'Criar conta' : 'Create account';
  String get useEmailPassword => isPortuguese
      ? 'Usar email e senha'
      : 'Use email and password';
  String get hideEmailPassword =>
      isPortuguese ? 'Ocultar' : 'Hide';
  String get login => signIn;
  String get loggingIn => isPortuguese ? 'Entrando…' : 'Signing in…';
  String get register => createAccount;
  String get registering => isPortuguese ? 'Criando…' : 'Creating…';
  String get noAccountSignUp => createAccount;
  String get noAccountRegister => createAccount;
  String get haveAccountLogin => isPortuguese
      ? 'Já tem conta? Entrar'
      : 'Already have an account? Sign in';
  String get email => 'Email';
  String get password => isPortuguese ? 'Senha' : 'Password';
  String get name => isPortuguese ? 'Nome' : 'Name';

  String get dashboard => 'Dashboard';
  String welcomeBack(String name) => isPortuguese
      ? 'Olá, $name'
      : 'Hi, $name';
  String get welcomeGuest =>
      isPortuguese ? 'Bem-vindo' : 'Welcome';
  String get createNewHouse =>
      isPortuguese ? 'Nova casa' : 'New house';
  String get houseName => isPortuguese ? 'Nome da casa' : 'House name';
  String get createHouse =>
      isPortuguese ? 'Criar' : 'Create';
  String get yourHouses =>
      isPortuguese ? 'Suas casas' : 'Your houses';
  String get noHousesYet => isPortuguese
      ? 'Crie a primeira casa para começar.'
      : 'Create your first house to get started.';
  String get viewHouse => isPortuguese ? 'Abrir' : 'Open';
  String get defaultBadge =>
      isPortuguese ? 'Padrão' : 'Default';
  String get setAsDefault =>
      isPortuguese ? 'Tornar padrão' : 'Make default';
  String houseSetAsDefault(String name) => isPortuguese
      ? '$name é a casa padrão'
      : '$name is now your default';

  String get settings =>
      isPortuguese ? 'Configurações' : 'Settings';
  String get language => isPortuguese ? 'Idioma' : 'Language';
  String get languageSystem => isPortuguese
      ? 'Dispositivo'
      : 'Device';
  String get languageEnglish => 'English';
  String get languagePortuguese => 'Português';
  String get defaultHouse =>
      isPortuguese ? 'Casa padrão' : 'Default house';
  String get defaultHouseHint => isPortuguese
      ? 'Abre nesta casa nas próximas sessões.'
      : 'Opens this house next time you sign in.';
  String get noDefaultHouse => isPortuguese
      ? 'Nenhuma (Dashboard)'
      : 'None (Dashboard)';
  String get switchHouse =>
      isPortuguese ? 'Ir para' : 'Go to';
  String get houses => isPortuguese ? 'Casas' : 'Houses';
  String get logout => isPortuguese ? 'Sair' : 'Log out';
  String get back => isPortuguese ? 'Voltar' : 'Back';
  String get backToLogin =>
      isPortuguese ? 'Voltar' : 'Back';
  String get loading => isPortuguese ? 'Carregando…' : 'Loading…';
  String get restoringSession =>
      isPortuguese ? 'Restaurando…' : 'Restoring…';
  String get signingIn => isPortuguese ? 'Entrando' : 'Signing in';
  String get completingSignIn =>
      isPortuguese ? 'Concluindo…' : 'Finishing…';
  String signInFailed(String detail) => isPortuguese
      ? 'Falha ao entrar: $detail'
      : 'Sign-in failed: $detail';
  String get missingAuthCode => isPortuguese
      ? 'código ausente.'
      : 'missing code.';
  String get redirectingToLogin => isPortuguese
      ? 'Redirecionando…'
      : 'Redirecting…';
  String get navigation => isPortuguese ? 'Menu' : 'Menu';
  String get openMenu => isPortuguese ? 'Menu' : 'Menu';
  String get closeMenu => isPortuguese ? 'Fechar' : 'Close';
  String get save => isPortuguese ? 'Salvar' : 'Save';
  String get saved => isPortuguese ? 'Salvo' : 'Saved';
  String get retry => isPortuguese ? 'Tentar de novo' : 'Retry';
  String get toggleLanguage =>
      isPortuguese ? 'EN' : 'PT';

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
      isPortuguese ? 'Agenda' : 'Schedule';
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
