part of 'run_test_bloc.dart';

abstract class RunTestEvent extends Equatable {
  const RunTestEvent();

  @override
  List<Object> get props => [];
}

class StartTest extends RunTestEvent {
  final OONIRunV2Descriptor test;

  const StartTest({required this.test});

  @override
  List<Object> get props => [test];
}

class InterruptTest extends RunTestEvent {
  const InterruptTest();
}
