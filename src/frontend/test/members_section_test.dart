import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:provider/provider.dart';
import 'package:roomies/api/api_client.dart';
import 'package:roomies/models/models.dart';
import 'package:roomies/screens/house/members_section.dart';
import 'package:roomies/state/app_state.dart';

const _link = 'https://example.com/invites/new-token';

HouseInvitation _invitation(String id, {String? link}) => HouseInvitation(
      id: id,
      houseId: 'h1',
      email: '$id@example.com',
      role: 'member',
      status: 'pending',
      createdBy: 'u1',
      createdAt: '2026-10-01T00:00:00Z',
      expiresAt: '2026-10-08T00:00:00Z',
      manualAcceptanceUrl: link,
    );

class _FakeApi extends ApiClient {
  _FakeApi() : super(baseUrl: 'http://localhost/api');

  final invitations = <HouseInvitation>[_invitation('older')];
  bool failRevoke = false;
  bool failRefresh = false;

  @override
  Future<List<HouseInvitation>> listInvitations(String houseId) async {
    if (failRefresh) throw Exception('refresh failed');
    return List.of(invitations);
  }

  @override
  Future<HouseInvitation> createInvitation(
      String houseId, String email, String role) async {
    final invite = _invitation('new', link: _link);
    invitations.add(invite);
    return invite;
  }

  @override
  Future<HouseInvitation> revokeInvitation(
      String houseId, String invitationId) async {
    if (failRevoke) throw Exception('revoke failed');
    final invite = invitations.firstWhere((i) => i.id == invitationId);
    invitations.remove(invite);
    return invite;
  }
}

Future<void> _createInvite(WidgetTester tester, _FakeApi api) async {
  await tester.pumpWidget(ChangeNotifierProvider(
    create: (_) => AppState(api, deviceLocale: 'en'),
    child: MaterialApp(
      home: Scaffold(
        body: SingleChildScrollView(
          child: MembersSection(
            houseId: 'h1',
            admin: true,
            members: const [],
            onRefresh: () {},
          ),
        ),
      ),
    ),
  ));
  await tester.pumpAndSettle();
  await tester.enterText(find.byType(TextField), 'new@example.com');
  await tester.tap(find.text('Send invitation'));
  await tester.pumpAndSettle();
  expect(find.text(_link), findsOneWidget);
  expect(find.text('Copy link'), findsOneWidget);
}

Future<void> _revoke(WidgetTester tester, int index) async {
  final button = find.widgetWithText(TextButton, 'Revoke').at(index);
  await tester.ensureVisible(button);
  await tester.tap(button);
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('create then revoke removes the link and Copy button',
      (tester) async {
    await _createInvite(tester, _FakeApi());
    await _revoke(tester, 1);
    expect(find.text(_link), findsNothing);
    expect(find.text('Copy link'), findsNothing);
  });

  testWidgets('revoking another invitation preserves the displayed link',
      (tester) async {
    await _createInvite(tester, _FakeApi());
    await _revoke(tester, 0);
    expect(find.text(_link), findsOneWidget);
    expect(find.text('Copy link'), findsOneWidget);
  });

  testWidgets('failed revocation preserves the displayed link', (tester) async {
    final api = _FakeApi();
    await _createInvite(tester, api);
    api.failRevoke = true;
    await _revoke(tester, 1);
    expect(find.text(_link), findsOneWidget);
    expect(find.text('Copy link'), findsOneWidget);
  });

  testWidgets('successful revoke clears the link even if refresh fails',
      (tester) async {
    final api = _FakeApi();
    await _createInvite(tester, api);
    api.failRefresh = true;
    await _revoke(tester, 1);
    expect(find.text(_link), findsNothing);
    expect(find.text('Copy link'), findsNothing);
  });
}
