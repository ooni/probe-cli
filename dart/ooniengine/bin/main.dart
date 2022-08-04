import 'dart:async';
import 'dart:convert';
import 'dart:io';

import "package:args/args.dart";
import 'package:fixnum/fixnum.dart' as $fixnum;
import "package:args/command_runner.dart";
import 'package:path/path.dart' as filepath;

import "package:ooniengine/abi.pb.dart";
import "package:ooniengine/engine.dart";

/// Returns the HOME directory path.
String homeDirectory() {
  // See https://en.wikipedia.org/wiki/Home_directory
  if (Platform.isWindows) {
    return Platform.environment["USERPROFILE"]!;
  }
  return Platform.environment["HOME"]!;
}

/// Returns the location of the OONI_HOME directory. If [maybeHome]
/// is not the empty string, we'll use it for computing the OONI_HOME,
/// otherwise we use $HOME (or %USERPROFILE%) as the base dir.
String ooniHome(String maybeHome) {
  if (maybeHome == "") {
    maybeHome = homeDirectory();
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
    new Directory(stateDir).create(recursive: true);
    tunnelDir = filepath.join(ooniHome, "tunnel");
    new Directory(tunnelDir).create(recursive: true);
    tempDir = Directory.systemTemp.createTempSync().path;
  }
}

/// Creates a new instance of the OONI Engine and runs it inside an Isolate.
Engine newEngine() {
  // TODO(bassosimone): make this depend on the OS/ARCH
  return Engine("./LIBRARY/darwin/arm64/libooniengine.dylib");
}

/// Name of this software
const softwareName = "ooniengine-tool";

/// Version of this software
const softwareVersion = "0.1.0-dev";

/// Returns the correct log level given command line flags.
LogLevel logLevel(ArgResults args) {
  if (args["verbose"]) {
    return LogLevel.DEBUG;
  }
  return LogLevel.INFO;
}

/// Returns the correct proxy URL given command line flags.
String proxyURL(ArgResults args) {
  final tunnel = args["tunnel"];
  var proxy = args["proxy"];
  if (tunnel != "" && proxy != "") {
    throw Exception("you cannot specify --tunnel and --proxy-url together");
  }
  if (tunnel != "") {
    proxy = tunnel + ":///";
  }
  return proxy;
}

/// Creates a SessionConfig given command line flags.
SessionConfig newSessionConfig(ArgResults args) {
  final dirs = SessionDirectories(ooniHome(args["ooni-home"]));
  return SessionConfig(
    logLevel: logLevel(args),
    probeServicesUrl: args["probe-services"],
    proxyUrl: proxyURL(args),
    softwareName: softwareName,
    softwareVersion: softwareVersion,
    stateDir: dirs.stateDir,
    tempDir: dirs.tempDir,
    torArgs: args["tor-args"],
    torBinary: args["tor-binary"],
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

/// Implements the 'geoip' CLI command.
class GeoIPCommand extends Command {
  /// The command's name
  final name = "geoip";

  /// The command's description
  final description = "Geolocates the probe.";

  GeoIPCommand() {
    // nothing
  }

  void run() async {
    final cfg = GeoIPConfig(
      session: newSessionConfig(globalResults!),
    );
    Engine? engine;
    Task? task;
    StreamSubscription<ProcessSignal>? subscription;
    try {
      engine = newEngine();
      task = await engine.startGeoIPTask(cfg);
      subscription = freeOnSIGINT(task);
      await task.findFirst<GeoIPEvent>(silent: false);
    } finally {
      subscription?.cancel();
      task?.free();
      engine?.shutdown();
    }
  }
}

/// Queries the OONI engine to get all the available experiments.
Future<List<ExperimentMetaInfoEvent>> allExperiments() async {
  final cfg = ExperimentMetaInfoConfig();
  Engine? engine;
  Task? task;
  List<ExperimentMetaInfoEvent> res;
  try {
    engine = newEngine();
    task = await engine.startExperimentMetaInfoTask(cfg);
    res = await task.collect<ExperimentMetaInfoEvent>(silent: true);
  } finally {
    task?.free();
    engine?.shutdown();
  }
  return res;
}

/// Returns the maxRuntime value to configure.
$fixnum.Int64 maxRuntime(ArgResults args) {
  final v = args["max-runtime"] as String?;
  if (v == null) {
    return $fixnum.Int64(0);
  }
  return $fixnum.Int64.parseInt(v);
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

/// An experiment you can run using the runx subcommand.
class RunxSubcommand extends Command {
  /// The experiment name
  final String name;

  /// The experiment description
  final String description;

  RunxSubcommand(ExperimentMetaInfoEvent exp)
      : name = exp.name,
        description = "runs the ${exp.name} experiment" {
    argParser.addMultiOption(
      "annotation",
      abbr: "A",
      help: "Add annotation to measurement (can be repeated).",
      valueHelp: "KEY=VALUE",
      defaultsTo: [],
    );

    argParser.addMultiOption(
      "option",
      help: "Pass extra option to the experiment (can be repeated).",
      abbr: "O",
      valueHelp: "KEY=VALUE",
      defaultsTo: [],
    );

    argParser.addMultiOption(
      "input",
      help: "Add input for the experiment (can be repeated).",
      abbr: "i",
      valueHelp: "INPUT",
      defaultsTo: [],
      hide: !exp.usesInput,
    );

    argParser.addMultiOption(
      "input-file",
      help: "Read extra input from the given file (can be repeated).",
      abbr: "f",
      valueHelp: "PATH",
      defaultsTo: [],
      hide: !exp.usesInput,
    );

    argParser.addOption(
      "max-runtime",
      help: "Stop scheduling new measurements after N seconds.",
      valueHelp: "N",
      defaultsTo: null,
      hide: !exp.usesInput,
    );

    argParser.addFlag(
      "no-collector",
      abbr: "n",
      help: "Do not submit measurements to a collector.",
    );

    argParser.addFlag(
      "no-json",
      abbr: "N",
      help: "Do not save measurements to disk.",
    );

    argParser.addFlag(
      "random",
      help: "Randomize the input list order.",
      hide: !exp.usesInput,
    );

    argParser.addOption(
      "reportfile",
      help: "Path where to write measurements",
      abbr: "o",
      valueHelp: "PATH",
      defaultsTo: "report.jsonl",
    );
  }

  void run() async {
    final cfg = NettestConfig(
      annotations: keyValuePairs(argResults!["annotation"]),
      extraOptions: jsonEncode(keyValuePairs(argResults!["option"])),
      inputs: argResults!["input"],
      inputFilePaths: argResults!["input-file"],
      maxRuntime: maxRuntime(argResults!),
      name: name,
      noCollector: argResults!["no-collector"],
      noJson: argResults!["no-json"],
      random: argResults!["random"],
      reportFile: argResults!["reportfile"],
      session: newSessionConfig(globalResults!),
    );
    Engine? engine;
    Task? task;
    StreamSubscription<ProcessSignal>? subscription;
    try {
      engine = newEngine();
      task = await engine.startNettestTask(cfg);
      subscription = freeOnSIGINT(task);
      await task.foreachEvent((ev) => printEvent(ev));
    } finally {
      subscription?.cancel();
      task?.free();
      engine?.shutdown();
    }
  }
}

/// Implements the 'runx' CLI command.
class RunxCommand extends Command {
  /// The command's name
  final name = "runx";

  /// The command's description
  final description = "RUNs a single eXperiment.";

  RunxCommand(List<ExperimentMetaInfoEvent> exps) {
    for (final exp in exps) {
      addSubcommand(RunxSubcommand(exp));
    }
  }
}

/// Implements the 'run websites' CLI command.
class WebsitesSubcommand extends Command {
  /// The command's name.
  final name = "websites";

  /// The command's description.
  final description = "Measures whether a list of websites is blocked";

  WebsitesSubcommand() {
    argParser.addMultiOption(
      "annotation",
      abbr: "A",
      help: "Add annotation to measurement (can be repeated).",
      valueHelp: "KEY=VALUE",
      defaultsTo: [],
    );

    argParser.addMultiOption(
      "option",
      help: "Pass extra option to the experiment (can be repeated).",
      abbr: "O",
      valueHelp: "KEY=VALUE",
      defaultsTo: [],
    );

    argParser.addMultiOption(
      "input",
      help: "Add input for the experiment (can be repeated).",
      abbr: "i",
      valueHelp: "INPUT",
      defaultsTo: [],
    );

    argParser.addOption(
      "max-runtime",
      help: "Stop scheduling new measurements after N seconds.",
      valueHelp: "N",
      defaultsTo: null,
    );

    argParser.addFlag(
      "no-collector",
      abbr: "n",
      help: "Do not submit measurements to a collector.",
    );

    argParser.addFlag(
      "no-json",
      abbr: "N",
      help: "Do not save measurements to disk.",
    );

    argParser.addFlag(
      "random",
      help: "Randomize the input list order.",
    );

    argParser.addOption(
      "reportfile",
      help: "Path where to write measurements",
      abbr: "o",
      valueHelp: "PATH",
      defaultsTo: "report.jsonl",
    );
  }

  void run() async {
    final cfg = OONIRunV2MeasureDescriptorConfig(
      maxRuntime: maxRuntime(argResults!),
      noCollector: argResults!["no-collector"],
      noJson: argResults!["no-json"],
      random: argResults!["random"],
      reportFile: argResults!["reportfile"],
      session: newSessionConfig(globalResults!),
      v2Descriptor: OONIRunV2Descriptor(
        name: name,
        description: description,
        author: "Simone Basso <simone@openobservatory.org>",
        nettests: [
          OONIRunV2DescriptorNettest(
            annotations: keyValuePairs(argResults!["annotation"]),
            inputs: argResults!["input"],
            options: jsonEncode(keyValuePairs(argResults!["option"])),
            testName: "web_connectivity",
          ),
        ],
      ),
    );
    Engine? engine;
    Task? task;
    StreamSubscription<ProcessSignal>? subscription;
    try {
      engine = newEngine();
      task = await engine.startOONIRunV2MeasureDescriptorTask(cfg);
      subscription = freeOnSIGINT(task);
      await task.foreachEvent((ev) => printEvent(ev));
    } finally {
      subscription?.cancel();
      task?.free();
      engine?.shutdown();
    }
  }
}

/// Implements the 'run im' CLI command.
class IMSubcommand extends Command {
  /// The command's name.
  final name = "im";

  /// The command's description.
  final description = "Measures whether instant messaging apps are blocked";

  IMSubcommand() {
    argParser.addMultiOption(
      "annotation",
      abbr: "A",
      help: "Add annotation to measurement (can be repeated).",
      valueHelp: "KEY=VALUE",
      defaultsTo: [],
    );

    argParser.addFlag(
      "no-collector",
      abbr: "n",
      help: "Do not submit measurements to a collector.",
    );

    argParser.addFlag(
      "no-json",
      abbr: "N",
      help: "Do not save measurements to disk.",
    );

    argParser.addOption(
      "reportfile",
      help: "Path where to write measurements",
      abbr: "o",
      valueHelp: "PATH",
      defaultsTo: "report.jsonl",
    );
  }

  void run() async {
    final cfg = OONIRunV2MeasureDescriptorConfig(
      maxRuntime: $fixnum.Int64(0),
      noCollector: argResults!["no-collector"],
      noJson: argResults!["no-json"],
      random: false,
      reportFile: argResults!["reportfile"],
      session: newSessionConfig(globalResults!),
      v2Descriptor: OONIRunV2Descriptor(
        name: name,
        description: description,
        author: "Simone Basso <simone@openobservatory.org>",
        nettests: [
          OONIRunV2DescriptorNettest(
            annotations: keyValuePairs(argResults!["annotation"]),
            inputs: [],
            options: "{}",
            testName: "facebook_messenger",
          ),
          OONIRunV2DescriptorNettest(
            annotations: keyValuePairs(argResults!["annotation"]),
            inputs: [],
            options: "{}",
            testName: "whatsapp",
          ),
          OONIRunV2DescriptorNettest(
            annotations: keyValuePairs(argResults!["annotation"]),
            inputs: [],
            options: "{}",
            testName: "telegram",
          ),
          OONIRunV2DescriptorNettest(
            annotations: keyValuePairs(argResults!["annotation"]),
            inputs: [],
            options: "{}",
            testName: "signal",
          ),
        ],
      ),
    );
    Engine? engine;
    Task? task;
    StreamSubscription<ProcessSignal>? subscription;
    try {
      engine = newEngine();
      task = await engine.startOONIRunV2MeasureDescriptorTask(cfg);
      subscription = freeOnSIGINT(task);
      await task.foreachEvent((ev) => printEvent(ev));
    } finally {
      subscription?.cancel();
      task?.free();
      engine?.shutdown();
    }
  }
}

/// Implements the 'run' CLI command.
class RunCommand extends Command {
  /// The command's name.
  final name = "run";

  /// The command's description.
  final description = "Runs a group of experiments.";

  RunCommand() {
    addSubcommand(WebsitesSubcommand());
    addSubcommand(IMSubcommand());
  }
}

void mainWithOptions(List<String> args) async {
  final runner = CommandRunner(
    "ooniengine-tool.exe",
    "Simple command line tool using the OONI Engine.",
  );

  runner.argParser.addFlag("verbose", abbr: "v", help: "Run in verbose mode.");

  runner.argParser.addOption(
    "probe-services",
    help: "Override probe-services URL.",
    valueHelp: "URL",
    defaultsTo: "",
  );

  runner.argParser.addOption(
    "tunnel",
    help: "Use tunnel to communicate with the probe services.",
    allowed: [
      "psiphon",
      "tor",
    ],
    allowedHelp: {
      "psiphon": "Use the bundled Psiphon tunnel core as tunnel.",
      "tor": "Execute tor and use its SOCKS5 port.",
    },
    valueHelp: "TUNNEL",
    defaultsTo: "",
  );

  runner.argParser.addOption(
    "proxy",
    help: "Use the given proxy URL (e.g., socks5://127.0.0.1:9050).",
    valueHelp: "URL",
    defaultsTo: "",
  );

  runner.argParser.addOption(
    "ooni-home",
    help: "Use a specific OONI_HOME directory.",
    valueHelp: "DIR",
    defaultsTo: "",
  );

  runner.argParser.addMultiOption(
    "tor-args",
    help: "Arguments to append to tor's command line (can be repeated).",
    valueHelp: "ARG",
    defaultsTo: [],
  );

  runner.argParser.addOption(
    "tor-binary",
    help: "Absolute path to the tor binary to execute.",
    valueHelp: "PATH",
    defaultsTo: "",
  );

  runner.addCommand(GeoIPCommand());
  runner.addCommand(RunxCommand(await allExperiments()));
  runner.addCommand(RunCommand());

  runner.run(args);
}

void main(List<String> args) {
  mainWithOptions(args);
}
