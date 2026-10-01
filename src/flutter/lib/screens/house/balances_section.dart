import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/money.dart';
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
    final locale = app.localeCode;
    return RoomiesTabPanel(
      name: s.balances,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          RoomiesHeading(s.balances, level: 2),
          Text(s.balancesRefreshHint),
          const SizedBox(height: 12),
          if (balances == null)
            Text(s.selectTabToLoadBalances)
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
                        formatMoney(entry.paid, localeCode: locale),
                        formatMoney(entry.owed, localeCode: locale),
                        formatMoney(entry.net, localeCode: locale),
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
                    formatMoney(settlement.amount, localeCode: locale),
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
