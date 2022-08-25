import 'package:flutter/material.dart';
import 'package:easy_localization/easy_localization.dart';
import 'package:material_design_icons_flutter/material_design_icons_flutter.dart';

class TestResult extends StatelessWidget {
  const TestResult({Key? key}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        centerTitle: true,
        elevation: 0,
        title: const Text('TestResults.Overview.Title').tr(),
        actions: [
          IconButton(
            icon: const Icon(Icons.delete_forever_sharp),
            onPressed: () {},
          ),
        ],
      ),
      body: SingleChildScrollView(
        child: Column(
          children: [
            Stack(
              children: [
                Positioned.fill(
                  child: Container(
                    padding: const EdgeInsets.symmetric(vertical: 16),
                    child: Row(
                      mainAxisAlignment: MainAxisAlignment.spaceEvenly,
                      children: [
                        VerticalDivider(
                          width: 10,
                          color: Colors.grey.shade600,
                        ),
                        VerticalDivider(
                          width: 10,
                          color: Colors.grey.shade600,
                        ),
                      ],
                    ),
                  ),
                ),
                Padding(
                  padding: const EdgeInsets.symmetric(
                    vertical: 16,
                    horizontal: 40,
                  ),
                  child: Row(
                    mainAxisAlignment: MainAxisAlignment.spaceBetween,
                    children: [
                      Column(
                        children: [
                          const Text('TestResults.Overview.Hero.Tests').tr(),
                          const SizedBox(height: 8),
                          Text(
                            '5',
                            style: Theme.of(context).textTheme.headline5,
                          ),
                        ],
                      ),
                      Column(
                        children: [
                          const Text('TestResults.Overview.Hero.Networks').tr(),
                          const SizedBox(height: 8),
                          Text(
                            '2',
                            style: Theme.of(context).textTheme.headline5,
                          ),
                        ],
                      ),
                      Column(
                        children: [
                          const Text('TestResults.Overview.Hero.DataUsage')
                              .tr(),
                          Row(
                            children: const [
                              Icon(MdiIcons.arrowDownBold, size: 14),
                              SizedBox(width: 8),
                              Text('5.5 MB'),
                            ],
                          ),
                          Row(
                            children: const [
                              Icon(MdiIcons.arrowUpBold, size: 14),
                              SizedBox(width: 8),
                              Text('5.5 MB'),
                            ],
                          ),
                        ],
                      ),
                    ],
                  ),
                ),
              ],
            ),
            Container(),
          ],
        ),
      ),
    );
  }
}
