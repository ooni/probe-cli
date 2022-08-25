import 'package:flutter/material.dart';
import 'package:flutter_markdown/flutter_markdown.dart';

import 'package:easy_localization/easy_localization.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import 'package:shared/data/models/models.dart';
import 'package:shared/pages/home/bloc/run_test_bloc.dart';
import 'package:shared/pages/home/views/test_progress_indicator.dart';

class TestOverViewPage extends StatelessWidget {
  const TestOverViewPage({Key? key, required this.descriptor})
      : super(key: key);

  final RunDescriptor descriptor;

  static MaterialPageRoute route({required RunDescriptor descriptor}) {
    return MaterialPageRoute(builder: (context) {
      return TestOverViewPage(descriptor: descriptor);
    });
  }

  @override
  Widget build(BuildContext context) {
    var infoStyle =
        Theme.of(context).textTheme.bodyLarge?.copyWith(color: Colors.white);
    return Scaffold(
      body: Stack(
        children: [
          CustomScrollView(
            slivers: <Widget>[
              SliverAppBar(
                expandedHeight: MediaQuery.of(context).size.height * 0.4,
                title: Text(descriptor.name),
                pinned: true,
                flexibleSpace: SafeArea(
                  child: FlexibleSpaceBar(
                    background: Column(
                      children: [
                        const SizedBox(height: 60),
                        Icon(
                          descriptor.icon,
                          size: 80,
                          color: Colors.white,
                        ),
                        Padding(
                          padding: const EdgeInsets.only(top: 16),
                          child: Row(
                            mainAxisAlignment: MainAxisAlignment.center,
                            children: [
                              Text('Dashboard.Overview.Estimated',
                                      style: infoStyle)
                                  .tr(),
                              const Text('  '),
                              Text('data', style: infoStyle),
                            ],
                          ),
                        ),
                        Padding(
                          padding: const EdgeInsets.only(top: 8),
                          child: Row(
                            mainAxisAlignment: MainAxisAlignment.center,
                            children: [
                              Text('Dashboard.Overview.LatestTest',
                                      style: infoStyle)
                                  .tr(),
                              const Text('  '),
                              Text('data', style: infoStyle),
                            ],
                          ),
                        ),
                      ],
                    ),
                    centerTitle: true,
                    title: ElevatedButton(
                      style:
                          Theme.of(context).elevatedButtonTheme.style?.copyWith(
                                backgroundColor:
                                    MaterialStateProperty.all(Colors.white),
                              ),
                      child: Text(
                        'Dashboard.Card.Run',
                        style: TextStyle(
                          color: Theme.of(context).primaryColor,
                        ),
                      ).tr(),
                      onPressed: () async {
                        context.read<RunTestBloc>().add(
                              StartTest(test: descriptor.runV2Descriptor),
                            );
                      },
                    ),
                  ),
                ),
              ),
              SliverToBoxAdapter(
                child: Padding(
                  padding: const EdgeInsets.all(16.0),
                  child: Markdown(
                    data: descriptor.methodology,
                    shrinkWrap: true,
                    physics: const NeverScrollableScrollPhysics(),
                  ),
                ),
              ),
            ],
          ),
          const Positioned(
            bottom: 0,
            left: 0,
            right: 0,
            child: TestProgressIndicator(),
          )
        ],
      ),
    );
  }
}
