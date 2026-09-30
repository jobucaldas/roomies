import 'package:flutter/material.dart';

import '../../core/money.dart';
import '../../models/models.dart';
import '../../widgets/roomies_ui.dart';

class BalancesSection extends StatelessWidget {
  const BalancesSection({super.key, this.balances});

  final BalanceResponse? balances;

  @override
  Widget build(BuildContext context) {
    return RoomiesTabPanel(
      name: 'Balances',
      child: Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const RoomiesHeading('Balances', level: 2),
        const Text('Balances refresh when this tab is selected.'),
        if (balances == null)
          const Text(
            'Select this tab to load balances. Former users remain identified by name.',
          )
        else if (balances!.balances.isEmpty)
          const Text('No expenses yet.')
        else ...[
          ...balances!.balances.map(
            (entry) => RoomiesArticleCard(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  RoomiesHeading(entry.userName, level: 3),
                  Text(
                    'Paid \$${formatMoney(entry.paid)}; owed \$${formatMoney(entry.owed)}; net \$${formatMoney(entry.net)}',
                  ),
                ],
              ),
            ),
          ),
          ...balances!.settlements.map(
            (s) => Text(
              '${s.fromUserName} pays ${s.toUserName} \$${formatMoney(s.amount)}',
            ),
          ),
        ],
      ],
      ),
    );
  }
}
