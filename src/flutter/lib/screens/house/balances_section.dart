import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../models/models.dart';
import '../../state/app_state.dart';
import '../../widgets/roomies_ui.dart';

class BalancesSection extends StatelessWidget {
  const BalancesSection({super.key, this.balances});

  final BalanceResponse? balances;

  @override
  Widget build(BuildContext context) {
    final app = context.watch<AppState>();
    final s = app.strings;
    return RoomiesTabPanel(
      name: s.balances,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          RoomiesHeading(s.balances, level: 2),
          if (balances == null)
            const Padding(
              padding: EdgeInsets.symmetric(vertical: 32),
              child: Center(child: CircularProgressIndicator(strokeWidth: 2.5)),
            )
          else if (balances!.balances.isEmpty)
            Text(s.noExpensesYet)
          else ...[
            ...balances!.balances.map(
              (entry) => RoomiesArticleCard(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    RoomiesHeading(entry.userName, level: 3),
                    Text(
                      s.paidOwedNet(
                        app.money(entry.paid),
                        app.money(entry.owed),
                        app.money(entry.net),
                      ),
                    ),
                  ],
                ),
              ),
            ),
            ...balances!.settlements.map(
              (settlement) => Padding(
                padding: const EdgeInsets.symmetric(vertical: 6),
                child: Text(
                  s.paysAmount(
                    settlement.fromUserName,
                    settlement.toUserName,
                    app.money(settlement.amount),
                  ),
                ),
              ),
            ),
          ],
        ],
      ),
    );
  }
}
