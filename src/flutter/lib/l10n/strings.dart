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

  String get dashboard => isPortuguese ? 'Painel' : 'Dashboard';
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
  String get recentEvents =>
      isPortuguese ? 'Eventos recentes' : 'Recent events';
  String get noRecentEvents => isPortuguese
      ? 'Nenhum evento próximo.'
      : 'No upcoming events.';
  String get monthMoney =>
      isPortuguese ? 'Gastos do mês' : 'This month';
  String get noMonthExpenses => isPortuguese
      ? 'Nenhuma despesa neste mês.'
      : 'No expenses this month.';
  String monthSpendTotal(String amount) => isPortuguese
      ? 'Total: $amount'
      : 'Total: $amount';
  String get recentNotes =>
      isPortuguese ? 'Notas recentes' : 'Recent notes';
  String get noRecentNotes => isPortuguese
      ? 'Nenhuma nota ainda.'
      : 'No notes yet.';
  String get openHouse => isPortuguese ? 'Abrir casa' : 'Open house';
  String get focusHouse =>
      isPortuguese ? 'Casa em foco' : 'Focus house';
  String get summaryUnavailable => isPortuguese
      ? 'Não foi possível carregar o resumo.'
      : 'Could not load summary.';

  String get settings =>
      isPortuguese ? 'Configurações' : 'Settings';
  String get language => isPortuguese ? 'Idioma' : 'Language';
  String get languageSystem => isPortuguese
      ? 'Dispositivo'
      : 'Device';
  String get languageEnglish => 'English';
  String get languagePortuguese => 'Português';
  String get appearance =>
      isPortuguese ? 'Aparência' : 'Appearance';
  String get themeSystem => isPortuguese
      ? 'Dispositivo'
      : 'Device';
  String get themeLight => isPortuguese ? 'Claro' : 'Light';
  String get themeDark => isPortuguese ? 'Escuro' : 'Dark';
  String get defaultHouse =>
      isPortuguese ? 'Casa padrão' : 'Default house';
  String get defaultHouseHint => isPortuguese
      ? 'Abre nesta casa nas próximas sessões.'
      : 'Opens this house next time you sign in.';
  String get noDefaultHouse => isPortuguese
      ? 'Nenhuma (Painel)'
      : 'None (Dashboard)';
  String get switchHouse =>
      isPortuguese ? 'Ir para' : 'Go to';
  String get houses => isPortuguese ? 'Casas' : 'Houses';
  String get renameHouse =>
      isPortuguese ? 'Renomear casa' : 'Rename house';
  String get renameHouseHint => isPortuguese
      ? 'Altere o nome da casa selecionada.'
      : 'Change the name of the selected house.';
  String get houseRenamed =>
      isPortuguese ? 'Nome salvo' : 'Name saved';
  String get selectHouseToRename => isPortuguese
      ? 'Selecione uma casa para renomear.'
      : 'Select a house to rename.';
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
  String get cancel => isPortuguese ? 'Cancelar' : 'Cancel';
  String get delete => isPortuguese ? 'Excluir' : 'Delete';
  String get edit => isPortuguese ? 'Editar' : 'Edit';
  String get close => isPortuguese ? 'Fechar' : 'Close';
  String get confirm => isPortuguese ? 'Confirmar' : 'Confirm';
  String get toggleLanguage =>
      isPortuguese ? 'EN' : 'PT';

  String get tabExpenses => isPortuguese ? 'Despesas' : 'Expenses';
  String get tabNotes => isPortuguese ? 'Notas' : 'Notes';
  String get tabGroceries =>
      isPortuguese ? 'Compras' : 'Groceries';
  String get tabChores => isPortuguese ? 'Tarefas' : 'Chores';
  String get tabCalendar =>
      isPortuguese ? 'Calendário' : 'Calendar';
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
        tabBalances,
        tabNotifications,
        tabMembers,
      ];

  String get viewOnlyRole => isPortuguese
      ? 'Seu papel de monitor é só leitura.'
      : 'Your monitor role is view-only.';
  String get amount => isPortuguese ? 'Valor' : 'Amount';
  String get description =>
      isPortuguese ? 'Descrição' : 'Description';
  String get title => isPortuguese ? 'Título' : 'Title';
  String get content => isPortuguese ? 'Conteúdo' : 'Content';
  String get date => isPortuguese ? 'Data' : 'Date';
  String get categoryOptional =>
      isPortuguese ? 'Categoria (opcional)' : 'Category (optional)';
  String get visibility =>
      isPortuguese ? 'Visibilidade' : 'Visibility';
  String get shared => isPortuguese ? 'Compartilhada' : 'Shared';
  String get privateLabel => isPortuguese ? 'Privada' : 'Private';
  String get customSplits =>
      isPortuguese ? 'Divisão personalizada' : 'Custom splits';
  String get recipientUserIds => isPortuguese
      ? 'IDs dos destinatários'
      : 'Recipient user IDs';
  String get addExpense =>
      isPortuguese ? 'Adicionar despesa' : 'Add expense';
  String get newExpense =>
      isPortuguese ? 'Nova despesa' : 'New expense';
  String get saveExpense =>
      isPortuguese ? 'Salvar despesa' : 'Save expense';
  String get detailsEdit =>
      isPortuguese ? 'Detalhes / editar' : 'Details / edit';
  String get expenseDetails =>
      isPortuguese ? 'Detalhes da despesa' : 'Expense details';
  String get saveChanges =>
      isPortuguese ? 'Salvar alterações' : 'Save changes';
  String get loadingExpenses =>
      isPortuguese ? 'Carregando despesas…' : 'Loading expenses…';
  String get noExpensesYet =>
      isPortuguese ? 'Nenhuma despesa ainda.' : 'No expenses yet.';
  String get positiveAmountRequired => isPortuguese
      ? 'Informe um valor positivo com até duas casas decimais'
      : 'Enter a positive amount with at most two decimals';
  String get descriptionRequired => isPortuguese
      ? 'Descrição é obrigatória'
      : 'Description is required';
  String get deleteExpenseTitle =>
      isPortuguese ? 'Excluir despesa?' : 'Delete expense?';
  String get cannotUndo => isPortuguese
      ? 'Isso não pode ser desfeito.'
      : 'This cannot be undone.';
  String get confirmDelete =>
      isPortuguese ? 'Confirmar exclusão' : 'Confirm delete';
  String get activeMembers =>
      isPortuguese ? 'Membros ativos' : 'Active members';

  String get newNote => isPortuguese ? 'Nova nota' : 'New note';
  String get saveNote =>
      isPortuguese ? 'Salvar nota' : 'Save note';
  String get editNote =>
      isPortuguese ? 'Editar nota' : 'Edit note';
  String get loadingNotes =>
      isPortuguese ? 'Carregando notas…' : 'Loading notes…';
  String get noNotesYet =>
      isPortuguese ? 'Nenhuma nota ainda.' : 'No notes yet.';
  String get titleContentRequired => isPortuguese
      ? 'Título e conteúdo são obrigatórios'
      : 'Title and content are required';
  String get deleteNoteTitle =>
      isPortuguese ? 'Excluir nota?' : 'Delete note?';
  String noteBy(String author, String updated) => isPortuguese
      ? 'Por $author · atualizado $updated'
      : 'By $author · updated $updated';

  String get balances => isPortuguese ? 'Saldos' : 'Balances';
  String get balancesRefreshHint => isPortuguese
      ? 'Os saldos atualizam ao abrir esta aba.'
      : 'Balances refresh when this tab is selected.';
  String get selectTabToLoadBalances => isPortuguese
      ? 'Abra esta aba para carregar os saldos. Ex-membros continuam identificados pelo nome.'
      : 'Select this tab to load balances. Former users remain identified by name.';
  String paidOwedNet(String paid, String owed, String net) => isPortuguese
      ? 'Pagou $paid; deve $owed; líquido $net'
      : 'Paid $paid; owed $owed; net $net';
  String paysAmount(String from, String to, String amount) => isPortuguese
      ? '$from paga $to $amount'
      : '$from pays $to $amount';

  String get addGrocery =>
      isPortuguese ? 'Adicionar item' : 'Add grocery';
  String get grocerySaved =>
      isPortuguese ? 'Item salvo.' : 'Grocery saved.';
  String get groceryDeleted =>
      isPortuguese ? 'Item excluído.' : 'Grocery deleted.';
  String get noGroceriesYet =>
      isPortuguese ? 'Nenhuma compra ainda.' : 'No groceries yet.';
  String get needed => isPortuguese ? 'Pendente' : 'Needed';
  String get check => isPortuguese ? 'Marcar' : 'Check';
  String get uncheck => isPortuguese ? 'Desmarcar' : 'Uncheck';
  String get checked => isPortuguese ? 'Marcado' : 'Checked';
  String get createChore =>
      isPortuguese ? 'Criar tarefa' : 'Create chore';
  String get choreSaved =>
      isPortuguese ? 'Tarefa salva.' : 'Chore saved.';
  String get choreDeleted =>
      isPortuguese ? 'Tarefa excluída.' : 'Chore deleted.';
  String get choreEnabledSaved => isPortuguese
      ? 'Estado da tarefa salvo.'
      : 'Chore enabled state saved.';
  String get dueLocal =>
      isPortuguese ? 'Vencimento local' : 'Due local';
  String get disable => isPortuguese ? 'Desativar' : 'Disable';
  String get enable => isPortuguese ? 'Ativar' : 'Enable';
  String get createCalendarEvent => isPortuguese
      ? 'Criar evento'
      : 'Create calendar event';
  String get calendarEventSaved => isPortuguese
      ? 'Evento salvo.'
      : 'Calendar event saved.';
  String get calendarEventDeleted => isPortuguese
      ? 'Evento excluído.'
      : 'Calendar event deleted.';
  String get startLocal =>
      isPortuguese ? 'Início local' : 'Start local';
  String get endLocal =>
      isPortuguese ? 'Fim local' : 'End local';

  String get emailInvitations =>
      isPortuguese ? 'Convites por email' : 'Email invitations';
  String get inviteByEmail =>
      isPortuguese ? 'Convidar por email' : 'Invite by email';
  String get emailAddress =>
      isPortuguese ? 'Endereço de email' : 'Email address';
  String get role => isPortuguese ? 'Papel' : 'Role';
  String get roleMember => isPortuguese ? 'Membro' : 'Member';
  String get roleAdmin => isPortuguese ? 'Admin' : 'Admin';
  String get roleMonitor => isPortuguese ? 'Monitor' : 'Monitor';
  String get sendInvitation =>
      isPortuguese ? 'Enviar convite' : 'Send invitation';
  String get invitationHistory =>
      isPortuguese ? 'Histórico de convites' : 'Invitation history';
  String get loadingInvitations =>
      isPortuguese ? 'Carregando convites…' : 'Loading invitations…';
  String get noInvitationsYet =>
      isPortuguese ? 'Nenhum convite ainda.' : 'No invitations yet.';
  String get addExistingMember =>
      isPortuguese ? 'Adicionar membro existente' : 'Add existing member';
  String get userId => isPortuguese ? 'ID do usuário' : 'User ID';
  String get addMember =>
      isPortuguese ? 'Adicionar membro' : 'Add member';
  String get changeRole =>
      isPortuguese ? 'Alterar papel' : 'Change role';
  String get remove => isPortuguese ? 'Remover' : 'Remove';
  String get userIdRequired =>
      isPortuguese ? 'ID do usuário é obrigatório' : 'User ID is required';
  String get memberAdded =>
      isPortuguese ? 'Membro adicionado.' : 'Member added.';
  String get memberRemoved =>
      isPortuguese ? 'Membro removido.' : 'Member removed.';
  String get emailRequired =>
      isPortuguese ? 'Email é obrigatório.' : 'Email is required.';
  String inviteFailed(String detail) => isPortuguese
      ? 'Não foi possível enviar o convite: $detail'
      : 'Unable to send invitation: $detail';

  String get notificationsSchedule => isPortuguese
      ? 'Notificações e agenda'
      : 'Notifications & schedule';
  String get loadingNotificationSettings => isPortuguese
      ? 'Carregando preferências…'
      : 'Loading notification settings…';
  String get retryNotificationSettings => isPortuguese
      ? 'Tentar carregar de novo'
      : 'Retry loading notification settings';
  String get yourNotificationPreferences => isPortuguese
      ? 'Suas preferências de notificação'
      : 'Your notification preferences';
  String get savePreferences =>
      isPortuguese ? 'Salvar preferências' : 'Save preferences';
  String get scheduledEvents =>
      isPortuguese ? 'Eventos agendados' : 'Scheduled events';
  String get editScheduledEvent =>
      isPortuguese ? 'Editar evento agendado' : 'Edit scheduled event';
}
