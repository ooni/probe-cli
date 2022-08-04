import 'dart:async';
import 'dart:io';

import "package:args/args.dart";
import "package:args/command_runner.dart";
import 'package:ooniengine/sugar.dart';
import 'package:path/path.dart' as filepath;

import "package:ooniengine/abi.dart";
import "package:ooniengine/engine.dart";
import "package:ooniengine/tasks.dart";

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
String logLevel(ArgResults args) {
  if (args["verbose"]) {
    return "DEBUG";
  }
  return "INFO";
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
    probeServicesURL: args["probe-services"],
    proxyURL: proxyURL(args),
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
StreamSubscription<ProcessSignal> stopTaskOnSIGINT(BaseTask task) {
  return ProcessSignal.sigint.watch().listen((event) {
    task.stop();
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
    final engine = newEngine();
    final task = GeoIPTask(engine, cfg);
    final subscription = stopTaskOnSIGINT(task);
    final r = await findFirst<GeoIPEventValue>(task);
    subscription.cancel();
    task.stop();
    engine.shutdown();
    printEvent(r);
  }
}

/// Queries the OONI engine to get all the available experiments.
Future<List<MetaInfoExperimentEventValue>> allExperiments() async {
  final engine = newEngine();
  final cfg = MetaInfoExperimentConfig();
  final task = MetaInfoExperimentTask(engine, cfg);
  final res = await collect<MetaInfoExperimentEventValue>(task);
  task.stop();
  engine.shutdown();
  return res;
}

/// Returns the maxRuntime value to configure.
int maxRuntime(ArgResults args) {
  final v = args["max-runtime"] as String?;
  if (v == null) {
    return 0;
  }
  return int.parse(v);
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

  RunxSubcommand(MetaInfoExperimentEventValue exp)
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
    final engine = newEngine();
    final cfg = NettestConfig(
      annotations: keyValuePairs(argResults!["annotation"]),
      extraOptions: keyValuePairs(argResults!["option"]),
      inputs: argResults!["input"],
      inputFilePaths: argResults!["input-file"],
      maxRuntime: maxRuntime(argResults!),
      name: name,
      noCollector: argResults!["no-collector"],
      noJSON: argResults!["no-json"],
      random: argResults!["random"],
      reportFile: argResults!["reportfile"],
      session: newSessionConfig(globalResults!),
    );
    final task = NettestTask(engine, cfg);
    final subscription = stopTaskOnSIGINT(task);
    await foreachEvent(task, printEvent);
    subscription.cancel();
    task.stop();
    engine.shutdown();
  }
}

/// Implements the 'runx' CLI command.
class RunxCommand extends Command {
  /// The command's name
  final name = "runx";

  /// The command's description
  final description = "RUNs a single eXperiment.";

  RunxCommand(List<MetaInfoExperimentEventValue> exps) {
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
    final engine = newEngine();
    final cfg = OONIRunV2MeasureDescriptorConfig(
      maxRuntime: maxRuntime(argResults!),
      noCollector: argResults!["no-collector"],
      noJSON: argResults!["no-json"],
      random: argResults!["random"],
      reportFile: argResults!["reportfile"],
      session: newSessionConfig(globalResults!),
      descriptor: OONIRunV2Descriptor(
        name: name,
        description: description,
        author: "Simone Basso <simone@openobservatory.org>",
        nettests: [
          OONIRunV2Nettest(
            annotations: keyValuePairs(argResults!["annotation"]),
            inputs: argResults!["input"],
            options: keyValuePairs(argResults!["option"]),
            testName: "web_connectivity",
          ),
        ],
      ),
    );
    final task = OONIRunV2MeasureDescriptorTask(engine, cfg);
    final subscription = stopTaskOnSIGINT(task);
    await foreachEvent(task, printEvent);
    subscription.cancel();
    task.stop();
    engine.shutdown();
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
    final engine = newEngine();
    final cfg = OONIRunV2MeasureDescriptorConfig(
      maxRuntime: 0,
      noCollector: argResults!["no-collector"],
      noJSON: argResults!["no-json"],
      random: false,
      reportFile: argResults!["reportfile"],
      session: newSessionConfig(globalResults!),
      descriptor: OONIRunV2Descriptor(
        name: name,
        description: description,
        author: "Simone Basso <simone@openobservatory.org>",
        nettests: [
          OONIRunV2Nettest(
            annotations: keyValuePairs(argResults!["annotation"]),
            inputs: [],
            options: {},
            testName: "facebook_messenger",
          ),
          OONIRunV2Nettest(
            annotations: keyValuePairs(argResults!["annotation"]),
            inputs: [],
            options: {},
            testName: "whatsapp",
          ),
          OONIRunV2Nettest(
            annotations: keyValuePairs(argResults!["annotation"]),
            inputs: [],
            options: {},
            testName: "telegram",
          ),
          OONIRunV2Nettest(
            annotations: keyValuePairs(argResults!["annotation"]),
            inputs: [],
            options: {},
            testName: "signal",
          ),
        ],
      ),
    );
    final task = OONIRunV2MeasureDescriptorTask(engine, cfg);
    final subscription = stopTaskOnSIGINT(task);
    await foreachEvent(task, printEvent);
    subscription.cancel();
    task.stop();
    engine.shutdown();
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
  //mainMetaInfo();
}
