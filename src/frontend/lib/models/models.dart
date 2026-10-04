class User {
  User({
    required this.id,
    required this.name,
    required this.email,
    required this.createdAt,
  });

  final String id;
  final String name;
  final String email;
  final String createdAt;

  factory User.fromJson(Map<String, dynamic> json) => User(
        id: json['id'] as String,
        name: json['name'] as String,
        email: json['email'] as String,
        createdAt: json['created_at'] as String,
      );
}

class House {
  House({required this.id, required this.name, required this.createdAt});

  final String id;
  final String name;
  final String createdAt;

  factory House.fromJson(Map<String, dynamic> json) => House(
        id: json['id'] as String,
        name: json['name'] as String,
        createdAt: json['created_at'] as String,
      );
}

class HouseMember {
  HouseMember({
    required this.id,
    required this.houseId,
    required this.userId,
    required this.role,
    required this.joinedAt,
    this.userName = '',
    this.userEmail = '',
  });

  final String id;
  final String houseId;
  final String userId;
  final String role;
  final String joinedAt;
  final String userName;
  final String userEmail;

  factory HouseMember.fromJson(Map<String, dynamic> json) => HouseMember(
        id: json['id'] as String,
        houseId: json['house_id'] as String,
        userId: json['user_id'] as String,
        role: json['role'] as String,
        joinedAt: json['joined_at'] as String,
        userName: json['user_name'] as String? ?? '',
        userEmail: json['user_email'] as String? ?? '',
      );
}

class Expense {
  Expense({
    required this.id,
    required this.houseId,
    required this.payerId,
    required this.amount,
    required this.description,
    this.category = '',
    required this.date,
    required this.visibility,
    required this.createdAt,
    this.payerName = '',
  });

  final String id;
  final String houseId;
  final String payerId;
  final double amount;
  final String description;
  final String category;
  final String date;
  final String visibility;
  final String createdAt;
  final String payerName;

  factory Expense.fromJson(Map<String, dynamic> json) => Expense(
        id: json['id'] as String,
        houseId: json['house_id'] as String,
        payerId: json['payer_id'] as String,
        amount: (json['amount'] as num).toDouble(),
        description: json['description'] as String,
        category: json['category'] as String? ?? '',
        date: json['date'] as String,
        visibility: json['visibility'] as String,
        createdAt: json['created_at'] as String,
        payerName: json['payer_name'] as String? ?? '',
      );
}

class ExpenseSplit {
  ExpenseSplit({
    required this.id,
    required this.expenseId,
    required this.userId,
    required this.shareAmount,
    this.userName = '',
  });

  final String id;
  final String expenseId;
  final String userId;
  final double shareAmount;
  final String userName;

  factory ExpenseSplit.fromJson(Map<String, dynamic> json) => ExpenseSplit(
        id: json['id'] as String,
        expenseId: json['expense_id'] as String,
        userId: json['user_id'] as String,
        shareAmount: (json['share_amount'] as num).toDouble(),
        userName: json['user_name'] as String? ?? '',
      );
}

class ExpenseDetail {
  ExpenseDetail({required this.expense, this.splits = const []});

  final Expense expense;
  final List<ExpenseSplit> splits;

  factory ExpenseDetail.fromJson(Map<String, dynamic> json) => ExpenseDetail(
        expense: Expense.fromJson(json['expense'] as Map<String, dynamic>),
        splits: (json['splits'] as List<dynamic>? ?? [])
            .map((e) => ExpenseSplit.fromJson(e as Map<String, dynamic>))
            .toList(),
      );
}

class Note {
  Note({
    required this.id,
    required this.houseId,
    required this.authorId,
    required this.title,
    required this.content,
    required this.createdAt,
    required this.updatedAt,
    this.authorName = '',
  });

  final String id;
  final String houseId;
  final String authorId;
  final String title;
  final String content;
  final String createdAt;
  final String updatedAt;
  final String authorName;

  factory Note.fromJson(Map<String, dynamic> json) => Note(
        id: json['id'] as String,
        houseId: json['house_id'] as String,
        authorId: json['author_id'] as String,
        title: json['title'] as String,
        content: json['content'] as String,
        createdAt: json['created_at'] as String,
        updatedAt: json['updated_at'] as String,
        authorName: json['author_name'] as String? ?? '',
      );
}

class BalanceEntry {
  BalanceEntry({
    required this.userId,
    required this.userName,
    required this.paid,
    required this.owed,
    required this.net,
  });

  final String userId;
  final String userName;
  final double paid;
  final double owed;
  final double net;

  factory BalanceEntry.fromJson(Map<String, dynamic> json) => BalanceEntry(
        userId: json['user_id'] as String,
        userName: json['user_name'] as String,
        paid: (json['paid'] as num).toDouble(),
        owed: (json['owed'] as num).toDouble(),
        net: (json['net'] as num).toDouble(),
      );
}

class BalanceSettlement {
  BalanceSettlement({
    required this.fromUserId,
    required this.fromUserName,
    required this.toUserId,
    required this.toUserName,
    required this.amount,
  });

  final String fromUserId;
  final String fromUserName;
  final String toUserId;
  final String toUserName;
  final double amount;

  factory BalanceSettlement.fromJson(Map<String, dynamic> json) =>
      BalanceSettlement(
        fromUserId: json['from_user_id'] as String,
        fromUserName: json['from_user_name'] as String,
        toUserId: json['to_user_id'] as String,
        toUserName: json['to_user_name'] as String,
        amount: (json['amount'] as num).toDouble(),
      );
}

class BalanceResponse {
  BalanceResponse({this.balances = const [], this.settlements = const []});

  final List<BalanceEntry> balances;
  final List<BalanceSettlement> settlements;

  factory BalanceResponse.fromJson(Map<String, dynamic> json) =>
      BalanceResponse(
        balances: (json['balances'] as List<dynamic>? ?? [])
            .map((e) => BalanceEntry.fromJson(e as Map<String, dynamic>))
            .toList(),
        settlements: (json['settlements'] as List<dynamic>? ?? [])
            .map((e) => BalanceSettlement.fromJson(e as Map<String, dynamic>))
            .toList(),
      );
}

class AuthResponse {
  AuthResponse({required this.token, required this.user});

  final String token;
  final User user;

  factory AuthResponse.fromJson(Map<String, dynamic> json) => AuthResponse(
        token: json['token'] as String? ?? '',
        user: User.fromJson(json['user'] as Map<String, dynamic>),
      );
}

class AuthConfig {
  AuthConfig({
    required this.authkit,
    required this.password,
    this.redirectUri = '',
  });

  final bool authkit;
  final bool password;
  final String redirectUri;

  factory AuthConfig.fromJson(Map<String, dynamic> json) => AuthConfig(
        authkit: json['authkit'] as bool? ?? false,
        password: json['password'] as bool? ?? true,
        redirectUri: json['redirect_uri'] as String? ?? '',
      );
}

class MessageResponse {
  MessageResponse({required this.message});

  final String message;

  factory MessageResponse.fromJson(Map<String, dynamic> json) =>
      MessageResponse(message: json['message'] as String);
}

class HouseInvitation {
  HouseInvitation({
    required this.id,
    required this.houseId,
    required this.email,
    required this.role,
    required this.status,
    required this.createdBy,
    required this.createdAt,
    required this.expiresAt,
    this.acceptedBy,
    this.acceptedAt,
    this.revokedBy,
    this.revokedAt,
    this.manualAcceptanceUrl,
  });

  final String id;
  final String houseId;
  final String email;
  final String role;
  final String status;
  final String createdBy;
  final String createdAt;
  final String expiresAt;
  final String? acceptedBy;
  final String? acceptedAt;
  final String? revokedBy;
  final String? revokedAt;

  /// One-time acceptance link; only present in the response that created
  /// the invitation.
  final String? manualAcceptanceUrl;

  factory HouseInvitation.fromJson(Map<String, dynamic> json) =>
      HouseInvitation(
        id: json['id'] as String,
        houseId: json['house_id'] as String,
        email: json['email'] as String,
        role: json['role'] as String,
        status: json['status'] as String,
        createdBy: json['created_by'] as String,
        createdAt: json['created_at'] as String,
        expiresAt: json['expires_at'] as String,
        acceptedBy: json['accepted_by'] as String?,
        acceptedAt: json['accepted_at'] as String?,
        revokedBy: json['revoked_by'] as String?,
        revokedAt: json['revoked_at'] as String?,
        manualAcceptanceUrl: json['manual_acceptance_url'] as String?,
      );
}

class InvitationAcceptanceResponse {
  InvitationAcceptanceResponse({required this.invitation});

  final HouseInvitation invitation;

  factory InvitationAcceptanceResponse.fromJson(Map<String, dynamic> json) =>
      InvitationAcceptanceResponse(
        invitation: HouseInvitation.fromJson(
            json['invitation'] as Map<String, dynamic>),
      );
}

class NotificationPreferences {
  NotificationPreferences({
    this.houseId = '',
    this.userId = '',
    this.expenseCreatedEnabled = true,
    this.reminderEnabled = true,
    this.cadence = 'immediate',
    this.timezone = 'UTC',
    this.quietStartMinutes,
    this.quietEndMinutes,
    this.digestMinutes = 540,
  });

  final String houseId;
  final String userId;
  final bool expenseCreatedEnabled;
  final bool reminderEnabled;
  final String cadence;
  final String timezone;
  final int? quietStartMinutes;
  final int? quietEndMinutes;
  final int digestMinutes;

  factory NotificationPreferences.fromJson(Map<String, dynamic> json) =>
      NotificationPreferences(
        houseId: json['house_id'] as String? ?? '',
        userId: json['user_id'] as String? ?? '',
        expenseCreatedEnabled: json['expense_created_enabled'] as bool? ?? true,
        reminderEnabled: json['reminder_enabled'] as bool? ?? true,
        cadence: json['cadence'] as String? ?? 'immediate',
        timezone: json['timezone'] as String? ?? 'UTC',
        quietStartMinutes: json['quiet_start_minutes'] as int?,
        quietEndMinutes: json['quiet_end_minutes'] as int?,
        digestMinutes: json['digest_minutes'] as int? ?? 540,
      );

  Map<String, dynamic> toJson() => {
        'expense_created_enabled': expenseCreatedEnabled,
        'reminder_enabled': reminderEnabled,
        'cadence': cadence,
        'timezone': timezone,
        if (quietStartMinutes != null) 'quiet_start_minutes': quietStartMinutes,
        if (quietEndMinutes != null) 'quiet_end_minutes': quietEndMinutes,
        'digest_minutes': digestMinutes,
      };

  NotificationPreferences copyWith({
    String? houseId,
    String? userId,
    bool? expenseCreatedEnabled,
    bool? reminderEnabled,
    String? cadence,
    String? timezone,
    int? quietStartMinutes,
    bool clearQuietStart = false,
    int? quietEndMinutes,
    bool clearQuietEnd = false,
    int? digestMinutes,
  }) {
    return NotificationPreferences(
      houseId: houseId ?? this.houseId,
      userId: userId ?? this.userId,
      expenseCreatedEnabled:
          expenseCreatedEnabled ?? this.expenseCreatedEnabled,
      reminderEnabled: reminderEnabled ?? this.reminderEnabled,
      cadence: cadence ?? this.cadence,
      timezone: timezone ?? this.timezone,
      quietStartMinutes: clearQuietStart
          ? null
          : (quietStartMinutes ?? this.quietStartMinutes),
      quietEndMinutes:
          clearQuietEnd ? null : (quietEndMinutes ?? this.quietEndMinutes),
      digestMinutes: digestMinutes ?? this.digestMinutes,
    );
  }
}

class NotificationSubscription {
  NotificationSubscription({
    required this.id,
    required this.houseId,
    required this.userId,
    required this.platform,
    required this.deviceLabel,
    required this.createdAt,
    required this.lastSeenAt,
    this.revokedAt,
  });

  final String id;
  final String houseId;
  final String userId;
  final String platform;
  final String deviceLabel;
  final String createdAt;
  final String lastSeenAt;
  final String? revokedAt;

  factory NotificationSubscription.fromJson(Map<String, dynamic> json) =>
      NotificationSubscription(
        id: json['id'] as String,
        houseId: json['house_id'] as String,
        userId: json['user_id'] as String,
        platform: json['platform'] as String,
        deviceLabel: json['device_label'] as String,
        createdAt: json['created_at'] as String,
        lastSeenAt: json['last_seen_at'] as String,
        revokedAt: json['revoked_at'] as String?,
      );
}

class ScheduledHouseEvent {
  ScheduledHouseEvent({
    required this.id,
    required this.houseId,
    required this.creatorId,
    required this.title,
    required this.timezone,
    required this.dtstartLocal,
    required this.rrule,
    this.exdates = const [],
    this.nextOccurrenceAt,
    required this.enabled,
    required this.createdAt,
    required this.updatedAt,
  });

  final String id;
  final String houseId;
  final String creatorId;
  final String title;
  final String timezone;
  final String dtstartLocal;
  final String rrule;
  final List<String> exdates;
  final String? nextOccurrenceAt;
  final bool enabled;
  final String createdAt;
  final String updatedAt;

  factory ScheduledHouseEvent.fromJson(Map<String, dynamic> json) =>
      ScheduledHouseEvent(
        id: json['id'] as String,
        houseId: json['house_id'] as String,
        creatorId: json['creator_id'] as String,
        title: json['title'] as String,
        timezone: json['timezone'] as String,
        dtstartLocal: json['dtstart_local'] as String,
        rrule: json['rrule'] as String,
        exdates: (json['exdates'] as List<dynamic>? ?? [])
            .map((e) => e as String)
            .toList(),
        nextOccurrenceAt: json['next_occurrence_at'] as String?,
        enabled: json['enabled'] as bool,
        createdAt: json['created_at'] as String,
        updatedAt: json['updated_at'] as String,
      );
}

class GroceryItem {
  GroceryItem({
    required this.id,
    required this.houseId,
    required this.creatorId,
    required this.name,
    required this.quantity,
    required this.unit,
    required this.note,
    this.assigneeId,
    required this.checked,
    this.checkedBy,
    this.checkedAt,
    required this.position,
    required this.version,
    required this.createdAt,
    required this.updatedAt,
  });

  final String id;
  final String houseId;
  final String creatorId;
  final String name;
  final String quantity;
  final String unit;
  final String note;
  final String? assigneeId;
  final bool checked;
  final String? checkedBy;
  final String? checkedAt;
  final int position;
  final int version;
  final String createdAt;
  final String updatedAt;

  factory GroceryItem.fromJson(Map<String, dynamic> json) => GroceryItem(
        id: json['id'] as String,
        houseId: json['house_id'] as String,
        creatorId: json['creator_id'] as String,
        name: json['name'] as String,
        quantity: json['quantity'] as String,
        unit: json['unit'] as String,
        note: json['note'] as String? ?? '',
        assigneeId: json['assignee_id'] as String?,
        checked: json['checked'] as bool,
        checkedBy: json['checked_by'] as String?,
        checkedAt: json['checked_at'] as String?,
        position: json['position'] as int,
        version: json['version'] as int,
        createdAt: json['created_at'] as String,
        updatedAt: json['updated_at'] as String,
      );
}

class Chore {
  Chore({
    required this.id,
    required this.houseId,
    required this.creatorId,
    required this.title,
    required this.description,
    this.assigneeId,
    required this.timezone,
    required this.dueLocal,
    required this.rrule,
    this.exdates = const [],
    required this.enabled,
    required this.version,
    required this.createdAt,
    required this.updatedAt,
  });

  final String id;
  final String houseId;
  final String creatorId;
  final String title;
  final String description;
  final String? assigneeId;
  final String timezone;
  final String dueLocal;
  final String rrule;
  final List<String> exdates;
  final bool enabled;
  final int version;
  final String createdAt;
  final String updatedAt;

  factory Chore.fromJson(Map<String, dynamic> json) => Chore(
        id: json['id'] as String,
        houseId: json['house_id'] as String,
        creatorId: json['creator_id'] as String,
        title: json['title'] as String,
        description: json['description'] as String? ?? '',
        assigneeId: json['assignee_id'] as String?,
        timezone: json['timezone'] as String,
        dueLocal: json['due_local'] as String,
        rrule: json['rrule'] as String,
        exdates: (json['exdates'] as List<dynamic>? ?? [])
            .map((e) => e as String)
            .toList(),
        enabled: json['enabled'] as bool,
        version: json['version'] as int,
        createdAt: json['created_at'] as String,
        updatedAt: json['updated_at'] as String,
      );
}

class CalendarEvent {
  CalendarEvent({
    required this.id,
    required this.houseId,
    required this.creatorId,
    required this.title,
    required this.description,
    required this.timezone,
    required this.startLocal,
    required this.endLocal,
    required this.allDay,
    required this.rrule,
    this.exdates = const [],
    required this.version,
    required this.createdAt,
    required this.updatedAt,
  });

  final String id;
  final String houseId;
  final String creatorId;
  final String title;
  final String description;
  final String timezone;
  final String startLocal;
  final String endLocal;
  final bool allDay;
  final String rrule;
  final List<String> exdates;
  final int version;
  final String createdAt;
  final String updatedAt;

  factory CalendarEvent.fromJson(Map<String, dynamic> json) => CalendarEvent(
        id: json['id'] as String,
        houseId: json['house_id'] as String,
        creatorId: json['creator_id'] as String,
        title: json['title'] as String,
        description: json['description'] as String? ?? '',
        timezone: json['timezone'] as String,
        startLocal: json['start_local'] as String,
        endLocal: json['end_local'] as String,
        allDay: json['all_day'] as bool,
        rrule: json['rrule'] as String,
        exdates: (json['exdates'] as List<dynamic>? ?? [])
            .map((e) => e as String)
            .toList(),
        version: json['version'] as int,
        createdAt: json['created_at'] as String,
        updatedAt: json['updated_at'] as String,
      );
}

class ChatMessage {
  ChatMessage({
    required this.cursor,
    required this.id,
    required this.houseId,
    required this.authorId,
    this.body,
    required this.createdAt,
    required this.updatedAt,
    this.deletedAt,
    this.redactedAt,
  });

  final int cursor;
  final String id;
  final String houseId;
  final String authorId;
  final String? body;
  final String createdAt;
  final String updatedAt;
  final String? deletedAt;
  final String? redactedAt;

  factory ChatMessage.fromJson(Map<String, dynamic> json) => ChatMessage(
        cursor: json['cursor'] as int,
        id: json['id'] as String,
        houseId: json['house_id'] as String,
        authorId: json['author_id'] as String,
        body: json['body'] as String?,
        createdAt: json['created_at'] as String,
        updatedAt: json['updated_at'] as String,
        deletedAt: json['deleted_at'] as String?,
        redactedAt: json['redacted_at'] as String?,
      );
}

class ChatPage {
  ChatPage({this.messages = const [], this.nextCursor = ''});

  final List<ChatMessage> messages;
  final String nextCursor;

  factory ChatPage.fromJson(Map<String, dynamic> json) => ChatPage(
        messages: (json['messages'] as List<dynamic>? ?? [])
            .map((e) => ChatMessage.fromJson(e as Map<String, dynamic>))
            .toList(),
        nextCursor: json['next_cursor'] as String? ?? '',
      );
}

class VapidPublicKey {
  VapidPublicKey({this.publicKey = ''});

  final String publicKey;

  factory VapidPublicKey.fromJson(Map<String, dynamic> json) =>
      VapidPublicKey(publicKey: json['public_key'] as String? ?? '');
}
