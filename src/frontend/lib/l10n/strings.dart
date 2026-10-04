/// UI strings in English (default), Brazilian Portuguese and Spanish.
/// Prefer short labels — screens should scan without reading paragraphs.
class RoomiesStrings {
  const RoomiesStrings(this.localeCode);

  /// Language codes the UI ships, in fallback order (English first).
  static const supported = <String>['en', 'pt', 'es'];

  final String localeCode;

  bool get isPortuguese => localeCode == 'pt';
  bool get isSpanish => localeCode == 'es';

  /// Picks the first supported language from the device's preferred list
  /// (e.g. browser `navigator.languages`, OS language order). A device set to
  /// `fr, es` gets Spanish; anything unsupported falls back to English.
  static String resolveDeviceLocale(Iterable<String?> preferred) {
    for (final tag in preferred) {
      final language =
          (tag ?? '').trim().toLowerCase().split(RegExp('[-_]')).first;
      if (supported.contains(language)) return language;
    }
    return 'en';
  }

  /// Normalizes a stored override; unknown values mean "follow the device".
  static String? normalizeOverride(String? code) =>
      supported.contains(code) ? code : null;

  String _t(String en, String pt, String es) => switch (localeCode) {
        'pt' => pt,
        'es' => es,
        _ => en,
      };

  String get brand => 'Roomies';
  String get brandTagline => _t(
      'Shared homes, clearer money and chores.',
      'Casas compartilhadas, contas e tarefas mais claras.',
      'Casas compartidas, cuentas y tareas más claras.');
  String get privacyLine =>
      _t('Private by default', 'Privado por padrão', 'Privado por defecto');

  String get signIn => _t('Sign in', 'Entrar', 'Iniciar sesión');
  String get redirecting =>
      _t('Redirecting…', 'Redirecionando…', 'Redirigiendo…');
  String get createAccount =>
      _t('Create account', 'Criar conta', 'Crear cuenta');
  String get useEmailPassword => _t('Use email and password',
      'Usar email e senha', 'Usar correo y contraseña');
  String get hideEmailPassword => _t('Hide', 'Ocultar', 'Ocultar');
  String get login => signIn;
  String get loggingIn => _t('Signing in…', 'Entrando…', 'Iniciando sesión…');
  String get register => createAccount;
  String get registering => _t('Creating…', 'Criando…', 'Creando…');
  String get noAccountSignUp => createAccount;
  String get noAccountRegister => createAccount;
  String get haveAccountLogin => _t('Already have an account? Sign in',
      'Já tem conta? Entrar', '¿Ya tienes cuenta? Inicia sesión');
  String get email => 'Email';
  String get password => _t('Password', 'Senha', 'Contraseña');
  String get name => _t('Name', 'Nome', 'Nombre');

  String get dashboard => _t('Dashboard', 'Painel', 'Panel');
  String welcomeBack(String name) =>
      _t('Hi, $name', 'Olá, $name', 'Hola, $name');
  String get welcomeGuest => _t('Welcome', 'Bem-vindo', 'Bienvenido');
  String get createNewHouse => _t('New house', 'Nova casa', 'Nueva casa');
  String get houseName => _t('House name', 'Nome da casa', 'Nombre de la casa');
  String get createHouse => _t('Create', 'Criar', 'Crear');
  String get yourHouses => _t('Your houses', 'Suas casas', 'Tus casas');
  String get noHousesYet => _t(
      'Create your first house to get started.',
      'Crie a primeira casa para começar.',
      'Crea tu primera casa para empezar.');
  String get viewHouse => _t('Open', 'Abrir', 'Abrir');
  String get defaultBadge => _t('Default', 'Padrão', 'Predeterminada');
  String get setAsDefault =>
      _t('Make default', 'Tornar padrão', 'Hacer predeterminada');
  String houseSetAsDefault(String name) => _t('$name is now your default',
      '$name é a casa padrão', '$name es ahora tu casa predeterminada');
  String get recentEvents =>
      _t('Recent events', 'Eventos recentes', 'Eventos recientes');
  String get noRecentEvents => _t('No upcoming events.',
      'Nenhum evento próximo.', 'No hay eventos próximos.');
  String get monthMoney => _t('This month', 'Gastos do mês', 'Este mes');
  String get noMonthExpenses => _t('No expenses this month.',
      'Nenhuma despesa neste mês.', 'No hay gastos este mes.');
  String monthSpendTotal(String amount) =>
      _t('Total: $amount', 'Total: $amount', 'Total: $amount');
  String get recentNotes =>
      _t('Recent notes', 'Notas recentes', 'Notas recientes');
  String get noRecentNotes =>
      _t('No notes yet.', 'Nenhuma nota ainda.', 'Aún no hay notas.');
  String get openHouse => _t('Open house', 'Abrir casa', 'Abrir casa');
  String get focusHouse => _t('Focus house', 'Casa em foco', 'Casa en foco');
  String get summaryUnavailable => _t('Could not load summary.',
      'Não foi possível carregar o resumo.', 'No se pudo cargar el resumen.');
  String get home => _t('Home', 'Início', 'Inicio');
  String get navHouseSection => _t('House', 'Casa', 'Casa');
  String get switchHouse => _t('Switch house', 'Trocar casa', 'Cambiar casa');
  String get selectHouse =>
      _t('Select house', 'Selecionar casa', 'Seleccionar casa');
  String get upcoming => _t('Upcoming', 'Próximos', 'Próximos');
  String get noUpcoming => _t('Nothing coming up.', 'Nada agendado por agora.',
      'Nada programado por ahora.');
  String get notesShowcase =>
      _t('House notes', 'Notas da casa', 'Notas de la casa');
  String get openNotes => _t('Open notes', 'Ver notas', 'Ver notas');
  String get moneyGraph => _t('This month', 'Gastos do mês', 'Este mes');
  String get needHouseForAction => _t('Create or pick a house first.',
      'Crie ou escolha uma casa primeiro.', 'Crea o elige una casa primero.');

  String get settings => _t('Settings', 'Configurações', 'Configuración');
  String get language => _t('Language', 'Idioma', 'Idioma');
  String get languageSystem => _t('Device', 'Dispositivo', 'Dispositivo');
  String get languageEnglish => 'English';
  String get languagePortuguese => 'Português';
  String get languageSpanish => 'Español';
  String get appearance => _t('Appearance', 'Aparência', 'Apariencia');
  String get currency => _t('Currency', 'Moeda', 'Moneda');
  String get currencyFollowLanguage =>
      _t('Follow language', 'Seguir o idioma', 'Según el idioma');
  String currencyName(String code) => switch (code) {
        'USD' => _t('US dollar', 'Dólar americano', 'Dólar estadounidense'),
        'BRL' => _t('Brazilian real', 'Real brasileiro', 'Real brasileño'),
        'EUR' => _t('Euro', 'Euro', 'Euro'),
        'GBP' => _t('British pound', 'Libra esterlina', 'Libra esterlina'),
        'MXN' => _t('Mexican peso', 'Peso mexicano', 'Peso mexicano'),
        'ARS' => _t('Argentine peso', 'Peso argentino', 'Peso argentino'),
        'CLP' => _t('Chilean peso', 'Peso chileno', 'Peso chileno'),
        'COP' => _t('Colombian peso', 'Peso colombiano', 'Peso colombiano'),
        _ => code,
      };
  String get brandTheme =>
      _t('Accent color', 'Cor de destaque', 'Color de acento');
  String get brandMint => _t('Mint', 'Verde-água', 'Menta');
  String get brandPlum => _t('Plum', 'Ameixa', 'Ciruela');
  String get themeSystem => _t('Device', 'Dispositivo', 'Dispositivo');
  String get themeLight => _t('Light', 'Claro', 'Claro');
  String get themeDark => _t('Dark', 'Escuro', 'Oscuro');
  String get defaultHouse =>
      _t('Default house', 'Casa padrão', 'Casa predeterminada');
  String get defaultHouseHint => _t(
      'The selected house opens when you sign in.',
      'A casa marcada abre quando você entra.',
      'La casa seleccionada se abre al iniciar sesión.');
  String get noDefaultHouse =>
      _t('None (Dashboard)', 'Nenhuma (Painel)', 'Ninguna (Panel)');
  String get houses => _t('Houses', 'Casas', 'Casas');
  String get renameHouse =>
      _t('Rename house', 'Renomear casa', 'Renombrar casa');
  String renameHouseNamed(String name) =>
      _t('Rename $name', 'Renomear $name', 'Renombrar $name');
  String get houseRenamed => _t('Name saved', 'Nome salvo', 'Nombre guardado');
  String get logout => _t('Log out', 'Sair', 'Cerrar sesión');
  String get back => _t('Back', 'Voltar', 'Volver');
  String get backToLogin =>
      _t('Back to sign in', 'Voltar para entrar', 'Volver a iniciar sesión');
  String get loading => _t('Loading…', 'Carregando…', 'Cargando…');
  String get restoringSession =>
      _t('Restoring…', 'Restaurando…', 'Restaurando…');
  String get completingSignIn =>
      _t('Finishing…', 'Concluindo…', 'Finalizando…');
  String signInFailed(String detail) => _t('Sign-in failed: $detail',
      'Falha ao entrar: $detail', 'Error al iniciar sesión: $detail');
  String get missingAuthCode =>
      _t('missing code.', 'código ausente.', 'falta el código.');
  String get redirectingToLogin =>
      _t('Redirecting…', 'Redirecionando…', 'Redirigiendo…');
  String get checkingInvitation => _t('Checking invitation…',
      'Verificando convite…', 'Verificando invitación…');
  String get joiningHouse =>
      _t('Joining house…', 'Entrando na casa…', 'Uniéndote a la casa…');
  String get invitationJoined => _t(
      'You joined the house. Refreshing…',
      'Você entrou na casa. Atualizando…',
      'Te uniste a la casa. Actualizando…');
  String get invitationMissing => _t(
      'This invitation link is missing or has already been used.',
      'Este convite não existe ou já foi usado.',
      'Esta invitación no existe o ya fue usada.');
  String invitationFailed(String detail) => _t(
      'Unable to accept this invitation: $detail',
      'Não foi possível aceitar o convite: $detail',
      'No se pudo aceptar la invitación: $detail');
  String get retryInvitation =>
      _t('Retry acceptance', 'Tentar de novo', 'Reintentar');
  String get navigation => _t('Menu', 'Menu', 'Menú');
  String get openMenu => _t('Menu', 'Menu', 'Menú');
  String get closeMenu => _t('Close', 'Fechar', 'Cerrar');
  String get save => _t('Save', 'Salvar', 'Guardar');
  String get saved => _t('Saved', 'Salvo', 'Guardado');
  String get serverUnreachable => _t(
        "Can't reach Roomies right now. Try again in a moment.",
        'Não foi possível conectar ao Roomies. Tente novamente em instantes.',
        'No se puede conectar con Roomies. Inténtalo de nuevo en un momento.',
      );
  String get retry => _t('Retry', 'Tentar de novo', 'Reintentar');
  String get cancel => _t('Cancel', 'Cancelar', 'Cancelar');
  String get delete => _t('Delete', 'Excluir', 'Eliminar');
  String get edit => _t('Edit', 'Editar', 'Editar');
  String get close => _t('Close', 'Fechar', 'Cerrar');
  String get confirm => _t('Confirm', 'Confirmar', 'Confirmar');

  String get tabExpenses => _t('Expenses', 'Despesas', 'Gastos');
  String get tabNotes => _t('Notes', 'Notas', 'Notas');
  String get tabGroceries => _t('Groceries', 'Compras', 'Compras');
  String get tabChores => _t('Chores', 'Tarefas', 'Tareas');
  String get tabCalendar => _t('Calendar', 'Calendário', 'Calendario');
  String get tabBalances => _t('Balances', 'Saldos', 'Saldos');
  String get tabNotifications =>
      _t('Notifications', 'Notificações', 'Notificaciones');
  String get tabMembers => _t('Members', 'Membros', 'Miembros');

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

  String get viewOnlyRole => _t(
      'Your monitor role is view-only.',
      'Seu papel de monitor é só leitura.',
      'Tu rol de monitor es solo de lectura.');
  String get amount => _t('Amount', 'Valor', 'Monto');
  String get description => _t('Description', 'Descrição', 'Descripción');
  String get title => _t('Title', 'Título', 'Título');
  String get content => _t('Content', 'Conteúdo', 'Contenido');
  String get date => _t('Date', 'Data', 'Fecha');
  String get categoryOptional =>
      _t('Category (optional)', 'Categoria (opcional)', 'Categoría (opcional)');
  String get visibility => _t('Visibility', 'Visibilidade', 'Visibilidad');
  String get shared => _t('Shared', 'Compartilhada', 'Compartido');
  String get privateLabel => _t('Private', 'Privada', 'Privado');
  String get customSplits =>
      _t('Custom splits', 'Divisão personalizada', 'División personalizada');
  String get recipientUserIds => _t('Recipient user IDs',
      'IDs dos destinatários', 'IDs de los destinatarios');
  String get addExpense =>
      _t('Add expense', 'Adicionar despesa', 'Agregar gasto');
  String get newExpense => _t('New expense', 'Nova despesa', 'Nuevo gasto');
  String get saveExpense =>
      _t('Save expense', 'Salvar despesa', 'Guardar gasto');
  String get detailsEdit =>
      _t('Details / edit', 'Detalhes / editar', 'Detalles / editar');
  String get expenseDetails =>
      _t('Expense details', 'Detalhes da despesa', 'Detalles del gasto');
  String get saveChanges =>
      _t('Save changes', 'Salvar alterações', 'Guardar cambios');
  String get loadingExpenses =>
      _t('Loading expenses…', 'Carregando despesas…', 'Cargando gastos…');
  String get noExpensesYet =>
      _t('No expenses yet.', 'Nenhuma despesa ainda.', 'Aún no hay gastos.');
  String get positiveAmountRequired => _t(
      'Enter a positive amount with at most two decimals',
      'Informe um valor positivo com até duas casas decimais',
      'Ingresa un monto positivo con hasta dos decimales');
  String get descriptionRequired => _t('Description is required',
      'Descrição é obrigatória', 'La descripción es obligatoria');
  String get deleteExpenseTitle =>
      _t('Delete expense?', 'Excluir despesa?', '¿Eliminar gasto?');
  String get cannotUndo => _t('This cannot be undone.',
      'Isso não pode ser desfeito.', 'Esto no se puede deshacer.');
  String get confirmDelete =>
      _t('Confirm delete', 'Confirmar exclusão', 'Confirmar eliminación');
  String get activeMembers =>
      _t('Active members', 'Membros ativos', 'Miembros activos');

  String get newNote => _t('New note', 'Nova nota', 'Nueva nota');
  String get saveNote => _t('Save note', 'Salvar nota', 'Guardar nota');
  String get editNote => _t('Edit note', 'Editar nota', 'Editar nota');
  String get loadingNotes =>
      _t('Loading notes…', 'Carregando notas…', 'Cargando notas…');
  String get noNotesYet =>
      _t('No notes yet.', 'Nenhuma nota ainda.', 'Aún no hay notas.');
  String get titleContentRequired => _t(
      'Title and content are required',
      'Título e conteúdo são obrigatórios',
      'El título y el contenido son obligatorios');
  String get deleteNoteTitle =>
      _t('Delete note?', 'Excluir nota?', '¿Eliminar nota?');
  String noteBy(String author, String updated) => _t(
      'By $author · updated $updated',
      'Por $author · atualizado $updated',
      'Por $author · actualizado $updated');

  String lastSeen(String when) => _t('last seen $when',
      'visto por último $when', 'visto por última vez $when');
  String exceptionsList(String dates) =>
      _t('Exceptions: $dates', 'Exceções: $dates', 'Excepciones: $dates');
  String deviceLine(String platform, String label, String when) =>
      '$platform: $label ($when)';

  String get balances => _t('Balances', 'Saldos', 'Saldos');
  String paidOwedNet(String paid, String owed, String net) => _t(
      'Paid $paid; owed $owed; net $net',
      'Pagou $paid; deve $owed; líquido $net',
      'Pagó $paid; debe $owed; neto $net');
  String paysAmount(String from, String to, String amount) => _t(
      '$from pays $to $amount',
      '$from paga $to $amount',
      '$from paga a $to $amount');

  String get addGrocery =>
      _t('Add grocery', 'Adicionar item', 'Agregar artículo');
  String get grocerySaved =>
      _t('Grocery saved.', 'Item salvo.', 'Artículo guardado.');
  String get groceryDeleted =>
      _t('Grocery deleted.', 'Item excluído.', 'Artículo eliminado.');
  String get noGroceriesYet =>
      _t('No groceries yet.', 'Nenhuma compra ainda.', 'Aún no hay compras.');
  String get needed => _t('Needed', 'Pendente', 'Pendiente');
  String get check => _t('Check', 'Marcar', 'Marcar');
  String get uncheck => _t('Uncheck', 'Desmarcar', 'Desmarcar');
  String get checked => _t('Checked', 'Marcado', 'Marcado');
  String get createChore => _t('Create chore', 'Criar tarefa', 'Crear tarea');
  String get choreSaved =>
      _t('Chore saved.', 'Tarefa salva.', 'Tarea guardada.');
  String get choreDeleted =>
      _t('Chore deleted.', 'Tarefa excluída.', 'Tarea eliminada.');
  String get choreEnabledSaved => _t('Chore enabled state saved.',
      'Estado da tarefa salvo.', 'Estado de la tarea guardado.');
  String get dueLocal =>
      _t('Due local', 'Vencimento local', 'Vencimiento local');
  String get disable => _t('Disable', 'Desativar', 'Desactivar');
  String get enable => _t('Enable', 'Ativar', 'Activar');
  String get createCalendarEvent =>
      _t('Create calendar event', 'Criar evento', 'Crear evento');
  String get pickStartAndEnd => _t(
      'Pick a start and an end.',
      'Escolha início e fim.',
      'Elige inicio y fin.');
  String get endAfterStart => _t(
      'End must be after the start.',
      'O fim deve ser depois do início.',
      'El fin debe ser posterior al inicio.');
  String get calendarEventSaved =>
      _t('Calendar event saved.', 'Evento salvo.', 'Evento guardado.');
  String get calendarEventDeleted =>
      _t('Calendar event deleted.', 'Evento excluído.', 'Evento eliminado.');
  String get startLocal => _t('Start local', 'Início local', 'Inicio local');
  String get endLocal => _t('End local', 'Fim local', 'Fin local');

  String get emailInvitations =>
      _t('Email invitations', 'Convites por email', 'Invitaciones por correo');
  String get inviteByEmail =>
      _t('Invite by email', 'Convidar por email', 'Invitar por correo');
  String get emailAddress =>
      _t('Email address', 'Endereço de email', 'Correo electrónico');
  String get role => _t('Role', 'Papel', 'Rol');
  String get roleMember => _t('Member', 'Membro', 'Miembro');
  String get roleAdmin => _t('Admin', 'Admin', 'Admin');
  String get roleMonitor => _t('Monitor', 'Monitor', 'Monitor');
  String get sendInvitation =>
      _t('Send invitation', 'Enviar convite', 'Enviar invitación');
  String get invitationHistory => _t('Invitation history',
      'Histórico de convites', 'Historial de invitaciones');
  String get loadingInvitations => _t(
      'Loading invitations…', 'Carregando convites…', 'Cargando invitaciones…');
  String get noInvitationsYet => _t('No invitations yet.',
      'Nenhum convite ainda.', 'Aún no hay invitaciones.');
  String get addExistingMember => _t('Add existing member',
      'Adicionar membro existente', 'Agregar miembro existente');
  String get userId => _t('User ID', 'ID do usuário', 'ID de usuario');
  String get addMember =>
      _t('Add member', 'Adicionar membro', 'Agregar miembro');
  String get changeRole => _t('Change role', 'Alterar papel', 'Cambiar rol');
  String get remove => _t('Remove', 'Remover', 'Quitar');
  String get userIdRequired => _t('User ID is required',
      'ID do usuário é obrigatório', 'El ID de usuario es obligatorio');
  String get memberAdded =>
      _t('Member added.', 'Membro adicionado.', 'Miembro agregado.');
  String get memberRemoved =>
      _t('Member removed.', 'Membro removido.', 'Miembro quitado.');
  String get emailRequired => _t('Email is required.', 'Email é obrigatório.',
      'El correo es obligatorio.');
  String inviteFailed(String detail) => _t(
      'Unable to send invitation: $detail',
      'Não foi possível enviar o convite: $detail',
      'No se pudo enviar la invitación: $detail');

  String get notificationsSchedule => _t('Notifications & schedule',
      'Notificações e agenda', 'Notificaciones y agenda');
  String get loadingNotificationSettings => _t('Loading notification settings…',
      'Carregando preferências…', 'Cargando preferencias…');
  String get retryNotificationSettings => _t(
      'Retry loading notification settings',
      'Tentar carregar de novo',
      'Reintentar cargar preferencias');
  String unableToLoadNotificationSettings(String detail) => _t(
      'Unable to load notification settings: $detail',
      'Não foi possível carregar as preferências: $detail',
      'No se pudieron cargar las preferencias: $detail');
  String get yourNotificationPreferences => _t('Your notification preferences',
      'Suas preferências de notificação', 'Tus preferencias de notificación');
  String get sharedExpenseAlerts => _t('Shared expense alerts',
      'Alertas de despesas compartilhadas', 'Alertas de gastos compartidos');
  String get scheduledReminderAlerts => _t('Scheduled reminder alerts',
      'Alertas de lembretes agendados', 'Alertas de recordatorios programados');
  String get delivery => _t('Delivery', 'Entrega', 'Entrega');
  String get deliveryImmediate => _t('Immediate', 'Imediata', 'Inmediata');
  String get deliveryDailyDigest =>
      _t('Daily digest', 'Resumo diário', 'Resumen diario');
  String get ianaTimezone =>
      _t('IANA timezone', 'Fuso horário IANA', 'Zona horaria IANA');
  String get quietStartLocal => _t('Quiet start (local)',
      'Início do silêncio (local)', 'Inicio de silencio (local)');
  String get quietEndLocal => _t('Quiet end (local)', 'Fim do silêncio (local)',
      'Fin de silencio (local)');
  String get dailyDigestTimeLocal => _t('Daily digest time (local)',
      'Horário do resumo diário (local)', 'Hora del resumen diario (local)');
  String get savePreferences =>
      _t('Save preferences', 'Salvar preferências', 'Guardar preferencias');
  String get timezoneValidationHint => _t(
      'Use an IANA timezone and times between 00:00 and 23:59.',
      'Use um fuso IANA e horários entre 00:00 e 23:59.',
      'Usa una zona horaria IANA y horas entre 00:00 y 23:59.');
  String get notificationPreferencesSaved => _t(
      'Notification preferences saved.',
      'Preferências de notificação salvas.',
      'Preferencias de notificación guardadas.');
  String couldNotSavePreferences(String detail) => _t(
      'Could not save preferences: $detail',
      'Não foi possível salvar as preferências: $detail',
      'No se pudieron guardar las preferencias: $detail');

  String get browserPush =>
      _t('Browser push', 'Push no navegador', 'Push del navegador');
  String get browserPushOnlyWeb => _t(
      'Browser push is only available in a supported web browser.',
      'O push no navegador só está disponível em um navegador compatível.',
      'El push del navegador solo está disponible en un navegador compatible.');
  String get enableBrowserPush => _t('Enable browser push',
      'Ativar push no navegador', 'Activar push del navegador');
  String get disableBrowserPush => _t('Disable browser push',
      'Desativar push no navegador', 'Desactivar push del navegador');
  String get browserPushEnabled => _t('Browser push enabled.',
      'Push no navegador ativado.', 'Push del navegador activado.');
  String get browserPushDisabled => _t('Browser push disabled.',
      'Push no navegador desativado.', 'Push del navegador desactivado.');
  String subscriptionRefreshFailed(String detail) => _t(
      'Subscription saved but refresh failed: $detail',
      'Inscrição salva, mas a atualização falhou: $detail',
      'Suscripción guardada, pero la actualización falló: $detail');
  String get yourDevices =>
      _t('Your devices', 'Seus dispositivos', 'Tus dispositivos');

  String get scheduledEvents =>
      _t('Scheduled events', 'Eventos agendados', 'Eventos programados');
  String get addScheduledEvent => _t('Add scheduled event',
      'Adicionar evento agendado', 'Agregar evento programado');
  String get editScheduledEvent => _t('Edit scheduled event',
      'Editar evento agendado', 'Editar evento programado');
  String get localStart => _t('Local start', 'Início local', 'Inicio local');
  String get frequency => _t('Frequency', 'Frequência', 'Frecuencia');
  String get frequencyDaily => _t('Daily', 'Diária', 'Diaria');
  String get frequencyWeekly => _t('Weekly', 'Semanal', 'Semanal');
  String get frequencyMonthly => _t('Monthly', 'Mensal', 'Mensual');
  String get intervalRange =>
      _t('Interval (1–366)', 'Intervalo (1–366)', 'Intervalo (1–366)');
  String get countRange => _t(
      'Count (1–366; leave blank to use until)',
      'Contagem (1–366; deixe em branco para usar até)',
      'Cantidad (1–366; deja en blanco para usar hasta)');
  String get untilOptional => _t('Until (optional)', 'Até (opcional)', 'Hasta (opcional)');
  String get exdateLocalTimes => _t(
      'Skipped occurrences (optional)',
      'Ocorrências ignoradas (opcional)',
      'Ocurrencias omitidas (opcional)');
  String get createScheduledEvent => _t('Create scheduled event',
      'Criar evento agendado', 'Crear evento programado');
  String get saveScheduledEvent => _t('Save scheduled event',
      'Salvar evento agendado', 'Guardar evento programado');
  String get savingEllipsis => _t('Saving…', 'Salvando…', 'Guardando…');
  String get cancelEdit =>
      _t('Cancel edit', 'Cancelar edição', 'Cancelar edición');
  String get editingScheduledEvent => _t('Editing scheduled event.',
      'Editando evento agendado.', 'Editando evento programado.');
  String get eventEditingCancelled => _t('Event editing cancelled.',
      'Edição do evento cancelada.', 'Edición del evento cancelada.');
  String get scheduledEventCreated => _t('Scheduled event created.',
      'Evento agendado criado.', 'Evento programado creado.');
  String get scheduledEventUpdated => _t('Scheduled event updated.',
      'Evento agendado atualizado.', 'Evento programado actualizado.');
  String get scheduledEventDeleted => _t('Scheduled event deleted.',
      'Evento agendado excluído.', 'Evento programado eliminado.');
  String couldNotSaveEvent(String detail) => _t(
      'Could not save event: $detail',
      'Não foi possível salvar o evento: $detail',
      'No se pudo guardar el evento: $detail');
  String couldNotDeleteEvent(String detail) => _t(
      'Could not delete event: $detail',
      'Não foi possível excluir o evento: $detail',
      'No se pudo eliminar el evento: $detail');
  String get monitorsViewOnlySchedule => _t(
      'Monitors can view scheduled events but cannot create, edit, or delete them.',
      'Monitores podem ver eventos agendados, mas não criar, editar ou excluir.',
      'Los monitores pueden ver los eventos programados, pero no crearlos, editarlos ni eliminarlos.');
}
