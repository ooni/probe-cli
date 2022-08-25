import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:shared/pages/home/bloc/run_test_bloc.dart';

class TestProgressIndicator extends StatelessWidget {
  const TestProgressIndicator({Key? key}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    return BlocBuilder<RunTestBloc, RunTestState>(
      builder: (context, state) {
        if (state.state == TestState.running) {
          return SizedBox(
            width: double.infinity,
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const LinearProgressIndicator(),
                Padding(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 8.0,
                    vertical: 16,
                  ),
                  child: Text(
                    state.displayMessage,
                    style: Theme.of(context).textTheme.headline6,
                    maxLines: 1,
                  ),
                ),
              ],
            ),
          );
        }
        return const SizedBox(
          width: double.infinity,
          height: 0,
        );
      },
    );
  }
}
