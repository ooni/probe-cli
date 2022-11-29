import 'dart:async';
import 'package:bloc/bloc.dart';
import 'package:equatable/equatable.dart';
import 'package:protobuf/protobuf.dart';
import 'package:ooniengine/abi.pb.dart';
import 'dart:io';

import "package:ooniengine/engine.dart";
import 'package:path/path.dart' as filepath;
import 'package:path_provider/path_provider.dart';
import 'package:fixnum/fixnum.dart' as $fixnum;

part 'run_test_event.dart';
part 'run_test_state.dart';

/// Returns the HOME directory path.
Future<String> homeDirectory() async {
  return (await getTemporaryDirectory()).path;
}

/// Returns the location of the OONI_HOME directory. If [maybeHome]
/// is not the empty string, we'll use it for computing the OONI_HOME,
/// otherwise we use $HOME (or %USERPROFILE%) as the base dir.
Future<String> ooniHome(String maybeHome) async {
  if (maybeHome == "") {
    maybeHome = await homeDirectory();
  }
  return filepath.join(maybeHome, ".ooniprobe");
}

/// Creates and exports directories that should be used by a session.
class SessionDirectories {
  /// Directory that contains state.
  late String stateDir;

  /// Directory that contains temporary data.
  late String tempDir;

  /// Directory that contains tunnel state.
  late String tunnelDir;

  /// Generates the directory names and possibly creates them.
  SessionDirectories(String ooniHome) {
    stateDir = filepath.join(ooniHome, "engine");
    Directory(stateDir).create(recursive: true);
    tunnelDir = filepath.join(ooniHome, "tunnel");
    Directory(tunnelDir).create(recursive: true);
    tempDir = Directory.systemTemp.createTempSync().path;
  }
}

/// Creates a new instance of the OONI Engine and runs it inside an Isolate.
Engine newEngine() {
  final goos = operatingSystem();
  final goarch = architecture();
  final ext = libraryExtension();
  if (goos == 'darwin') {
    return Engine("${goarch}/libooniengine.${ext}");
  } else if (goos == 'windows') {
    return Engine("..\\..\\LIBRARY\\windows\\amd64\\libooniengine.dll");
  } else if (goos == 'linux') {
    return Engine("../../LIBRARY/linux/amd64/libooniengine.so");
  } else {
    return Engine("libooniengine.${ext}");
  }
}

/// Name of this software
const softwareName = "ooniengine-tool";

/// Version of this software
const softwareVersion = "0.1.0-dev";

/// Returns the correct proxy URL given command line flags.
String proxyURL() {
  final tunnel = '';
  var proxy = '';
  if (tunnel != "" && proxy != "") {
    throw Exception("you cannot specify --tunnel and --proxy-url together");
  }
  if (tunnel != "") {
    proxy = tunnel + ":///";
  }
  return proxy;
}

/// Creates a SessionConfig given command line flags.
Future<SessionConfig> newSessionConfig() async {
  final dirs = SessionDirectories(await ooniHome(''));
  return SessionConfig(
    logLevel: LogLevel.DEBUG,
    probeServicesUrl: '',
    proxyUrl: proxyURL(),
    softwareName: softwareName,
    softwareVersion: softwareVersion,
    stateDir: dirs.stateDir,
    tempDir: dirs.tempDir,
    torArgs: [],
    torBinary: '',
    tunnelDir: dirs.tunnelDir,
  );
}

/// Stops the given task on SIGINT. You must cancel the subscription
/// when you're done executing the given task, or Dart will hang.
StreamSubscription<ProcessSignal> freeOnSIGINT(Task task) {
  return ProcessSignal.sigint.watch().listen((event) {
    task.free();
  });
}

/// Parses key-value pairs from a list of strings.
Map<String, String> keyValuePairs(List<String> inputs) {
  var out = Map<String, String>();
  for (final input in inputs) {
    final v = input.split("=");
    if (v.length < 2) {
      throw Exception("cannot find the `=' separator in: ${input}");
    }
    final key = v[0];
    final value = v.sublist(1).join("=");
    out[key] = value;
  }
  return out;
}

class RunTestBloc extends Bloc<RunTestEvent, RunTestState> {
  RunTestBloc() : super(RunTestState()) {
    on<StartTest>(onStartTestChanged);
  }

  Future<void> onStartTestChanged(
    StartTest event,
    Emitter<RunTestState> emit,
  ) async {
    final cfg = OONIRunV2MeasureDescriptorConfig(
      maxRuntime: $fixnum.Int64(0),
      noCollector: null,
      noJson: null,
      random: false,
      reportFile: 'report.jsonl',
      session: await newSessionConfig(),
      v2Descriptor: event.test,
    );
    Engine? engine;
    Task? task;
    StreamSubscription<ProcessSignal>? subscription;
    try {
      emit(state.copyWith(state: TestState.running));
      engine = newEngine();
      task = await engine.startOONIRunV2MeasureDescriptorTask(cfg);
      subscription = freeOnSIGINT(task);
      await task.foreachEvent((ev) {
        emit(
          state.copyWith(message: ev),
        );
        return printMessage(ev);
      });
    } finally {
      subscription?.cancel();
      task?.free();
      engine?.shutdown();
      emit(state.copyWith(state: TestState.idle));
    }
  }
}
