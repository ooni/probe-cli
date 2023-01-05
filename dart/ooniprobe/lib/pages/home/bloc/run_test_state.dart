part of 'run_test_bloc.dart';

enum TestState {
  idle,
  running,
  finished,
  error,
}

class RunTestState extends Equatable {
  final TestState state;
  final GeneratedMessage? message;
  const RunTestState({this.state = TestState.idle, this.message});

  RunTestState copyWith({
    TestState? state,
    GeneratedMessage? message,
  }) {
    return RunTestState(
      state: state ?? this.state,
      message: message ?? this.message,
    );
  }

  get displayMessage {
    var msg = message;
    if (msg is LogEvent) {
      return msg.message;
    } else if (msg is ProgressEvent) {
      return "PROGRESS: ${msg.percentage * 100}%: ${msg.message}";
    } else if (msg is DataUsageEvent) {
      return "DATA_USAGE: sent ${msg.kibiBytesSent} KiB recv ${msg.kibiBytesReceived} KiB";
    } else if (msg is SubmitEvent) {
      return "SUBMIT: #${msg.index}... ${msg.failure == "" ? "ok" : msg.failure}";
    } else {
      return state.name;
    }
  }

  @override
  List<Object?> get props => [state, message];
}
