// AUTO GENERATED FILE, DO NOT EDIT.

import 'package:json_annotation/json_annotation.dart';

part 'abi.g.dart';

//
// Auto-generated ABI.
//

/// ABI version number.
const ABIVersion = "202208042045";

/// Name of the Log event
const LogEventName = "Log";

/// Name of the DataUsage event
const DataUsageEventName = "DataUsage";

/// Name of the GeoIP event
const GeoIPEventName = "GeoIP";

/// Debug log level.
const LogLevelDebug = "DEBUG";

/// Info log level.
const LogLevelInfo = "INFO";

/// Warning log level.
const LogLevelWarning = "WARNING";

/// Name of the MetaInfoExperiment event
const MetaInfoExperimentEventName = "MetaInfoExperiment";

/// Name of the Progress event
const ProgressEventName = "Progress";

/// The error string inside SubmitEvent when the user disabled submission.
const SubmissionDisabledError = "submission_disabled_error";

/// Name of the Submit event
const SubmitEventName = "Submit";

/// A log message.
@JsonSerializable()
class LogEventValue extends BaseEvent {
  /// Log level.
  @JsonKey(name: "level")
  final String level;

  /// Log message.
  @JsonKey(name: "message")
  final String message;

  /// Default constructor.
  LogEventValue({
    required this.level,
    required this.message,
  });

  /// Factory to construct from JSON.
  factory LogEventValue.fromJson(Map<String, dynamic> json) =>
      _$LogEventValueFromJson(json);

  /// Serialize to JSON.
  Map<String, dynamic> toJson() => _$LogEventValueToJson(this);
}

/// Information about the amount of data consumed by an experiment.
@JsonSerializable()
class DataUsageEventValue extends BaseEvent {
  /// KiB sent by this experiment.
  @JsonKey(name: "kibi_bytes_sent")
  final double kibiBytesSent;

  /// KiB received by this experiment.
  @JsonKey(name: "kibi_bytes_received")
  final double kibiBytesReceived;

  /// Default constructor.
  DataUsageEventValue({
    required this.kibiBytesSent,
    required this.kibiBytesReceived,
  });

  /// Factory to construct from JSON.
  factory DataUsageEventValue.fromJson(Map<String, dynamic> json) =>
      _$DataUsageEventValueFromJson(json);

  /// Serialize to JSON.
  Map<String, dynamic> toJson() => _$DataUsageEventValueToJson(this);
}

/// Probe geolocation information.
@JsonSerializable()
class GeoIPEventValue extends BaseEvent {
  /// Failure that occurred or empty string (on success)
  @JsonKey(name: "failure")
  final String failure;

  /// The probe's IP address.
  @JsonKey(name: "probe_ip")
  final String probeIP;

  /// ASN derived from the probe's IP.
  @JsonKey(name: "probe_asn")
  final String probeASN;

  /// Country code derived from the probe's IP.
  @JsonKey(name: "probe_cc")
  final String probeCC;

  /// Network name of the probe's ASN.
  @JsonKey(name: "probe_network_name")
  final String probeNetworkName;

  /// IPv4 address used by getaddrinfo.
  @JsonKey(name: "resolver_ip")
  final String resolverIP;

  /// ASN derived from the resolver's IP.
  @JsonKey(name: "resolver_asn")
  final String resolverASN;

  /// Network name of resolver's ASN.
  @JsonKey(name: "resolver_network_name")
  final String resolverNetworkName;

  /// Default constructor.
  GeoIPEventValue({
    required this.failure,
    required this.probeIP,
    required this.probeASN,
    required this.probeCC,
    required this.probeNetworkName,
    required this.resolverIP,
    required this.resolverASN,
    required this.resolverNetworkName,
  });

  /// Factory to construct from JSON.
  factory GeoIPEventValue.fromJson(Map<String, dynamic> json) =>
      _$GeoIPEventValueFromJson(json);

  /// Serialize to JSON.
  Map<String, dynamic> toJson() => _$GeoIPEventValueToJson(this);
}

/// Contains config for the GeoIP task.
@JsonSerializable()
class GeoIPConfig extends BaseConfig {
  /// Config for creating a session.
  @JsonKey(name: "session")
  final SessionConfig session;

  /// Default constructor.
  GeoIPConfig({
    required this.session,
  });

  /// Factory to construct from JSON.
  factory GeoIPConfig.fromJson(Map<String, dynamic> json) =>
      _$GeoIPConfigFromJson(json);

  /// Serialize to JSON.
  Map<String, dynamic> toJson() => _$GeoIPConfigToJson(this);
}

/// Config for the meta-info-experiment task.
@JsonSerializable()
class MetaInfoExperimentConfig extends BaseConfig {
  /// Default constructor.
  MetaInfoExperimentConfig();

  /// Factory to construct from JSON.
  factory MetaInfoExperimentConfig.fromJson(Map<String, dynamic> json) =>
      _$MetaInfoExperimentConfigFromJson(json);

  /// Serialize to JSON.
  Map<String, dynamic> toJson() => _$MetaInfoExperimentConfigToJson(this);
}

/// Contains meta-info about an experiment
@JsonSerializable()
class MetaInfoExperimentEventValue extends BaseEvent {
  /// The experiment name
  @JsonKey(name: "name")
  final String name;

  /// Whether this experiment could use input.
  ///
  /// If this field is false, it does not make sense to generate
  /// command line options for passing input to the experiment.
  @JsonKey(name: "uses_input")
  final bool usesInput;

  /// Default constructor.
  MetaInfoExperimentEventValue({
    required this.name,
    required this.usesInput,
  });

  /// Factory to construct from JSON.
  factory MetaInfoExperimentEventValue.fromJson(Map<String, dynamic> json) =>
      _$MetaInfoExperimentEventValueFromJson(json);

  /// Serialize to JSON.
  Map<String, dynamic> toJson() => _$MetaInfoExperimentEventValueToJson(this);
}

/// Base class for all events.
@JsonSerializable()
class BaseEvent {
  /// Default constructor.
  BaseEvent();

  /// Factory to construct from JSON.
  factory BaseEvent.fromJson(Map<String, dynamic> json) =>
      _$BaseEventFromJson(json);

  /// Serialize to JSON.
  Map<String, dynamic> toJson() => _$BaseEventToJson(this);
}

/// Base class for all configs.
@JsonSerializable()
class BaseConfig {
  /// Default constructor.
  BaseConfig();

  /// Factory to construct from JSON.
  factory BaseConfig.fromJson(Map<String, dynamic> json) =>
      _$BaseConfigFromJson(json);

  /// Serialize to JSON.
  Map<String, dynamic> toJson() => _$BaseConfigToJson(this);
}

/// Config for running a nettest.
@JsonSerializable()
class NettestConfig extends BaseConfig {
  /// OPTIONAL annotations for the nettest.
  @JsonKey(name: "annotations")
  final Map<String, String> annotations;

  /// OPTIONAL extra options for the nettest.
  @JsonKey(name: "extra_options")
  final Map<String, Object> extraOptions;

  /// OPTIONAL inputs for the nettest.
  @JsonKey(name: "inputs")
  final List<String> inputs;

  /// An OPTIONAL list of files from which to read inputs for the nettest.
  @JsonKey(name: "input_file_paths")
  final List<String> inputFilePaths;

  /// The OPTIONAL nettest maximum runtime in seconds.
  ///
  /// This setting only applies to nettests that require input, such
  /// as Web Connectivity.
  @JsonKey(name: "max_runtime")
  final int maxRuntime;

  /// The MANDATORY name of the nettest to execute.
  @JsonKey(name: "name")
  final String name;

  /// This setting allows to OPTIONALLY disable submitting measurements.
  ///
  /// The default is that we submit every measurement we perform.
  @JsonKey(name: "no_collector")
  final bool noCollector;

  /// This setting allows to OPTIONALLY disable saving measurements to disk.
  ///
  /// The default is to save using the file name indicated by ReportFile.
  @JsonKey(name: "no_json")
  final bool noJSON;

  /// OPTIONALLY tells the engine to randomly shuffle the input list.
  @JsonKey(name: "random")
  final bool random;

  /// The OPTIONAL name of the file where to save measurements.
  ///
  /// If this field is empty, we will use 'report.jsonl' as the file name.
  @JsonKey(name: "report_file")
  final String reportFile;

  /// Config for creating a session.
  @JsonKey(name: "session")
  final SessionConfig session;

  /// Default constructor.
  NettestConfig({
    required this.annotations,
    required this.extraOptions,
    required this.inputs,
    required this.inputFilePaths,
    required this.maxRuntime,
    required this.name,
    required this.noCollector,
    required this.noJSON,
    required this.random,
    required this.reportFile,
    required this.session,
  });

  /// Factory to construct from JSON.
  factory NettestConfig.fromJson(Map<String, dynamic> json) =>
      _$NettestConfigFromJson(json);

  /// Serialize to JSON.
  Map<String, dynamic> toJson() => _$NettestConfigToJson(this);
}

/// OONI Run v2 nettest descriptor.
@JsonSerializable()
class OONIRunV2Nettest {
  /// OPTIONAL annotations for the nettest.
  @JsonKey(name: "annotations")
  final Map<String, String> annotations;

  /// OPTIONAL inputs for the nettest.
  @JsonKey(name: "inputs")
  final List<String> inputs;

  /// OPTIONAL extra options for the nettest.
  @JsonKey(name: "options")
  final Map<String, Object> options;

  /// The MANDATORY name of the nettest to execute.
  @JsonKey(name: "test_name")
  final String testName;

  /// Default constructor.
  OONIRunV2Nettest({
    required this.annotations,
    required this.inputs,
    required this.options,
    required this.testName,
  });

  /// Factory to construct from JSON.
  factory OONIRunV2Nettest.fromJson(Map<String, dynamic> json) =>
      _$OONIRunV2NettestFromJson(json);

  /// Serialize to JSON.
  Map<String, dynamic> toJson() => _$OONIRunV2NettestToJson(this);
}

/// OONI Run v2 descriptor.
@JsonSerializable()
class OONIRunV2Descriptor {
  /// Name of this OONI Run v2 descriptor.
  @JsonKey(name: "name")
  final String name;

  /// Description for this OONI Run v2 descriptor.
  @JsonKey(name: "description")
  final String description;

  /// Author of this OONI Run v2 descriptor.
  @JsonKey(name: "author")
  final String author;

  @JsonKey(name: "nettests")
  final List<OONIRunV2Nettest> nettests;

  /// Default constructor.
  OONIRunV2Descriptor({
    required this.name,
    required this.description,
    required this.author,
    required this.nettests,
  });

  /// Factory to construct from JSON.
  factory OONIRunV2Descriptor.fromJson(Map<String, dynamic> json) =>
      _$OONIRunV2DescriptorFromJson(json);

  /// Serialize to JSON.
  Map<String, dynamic> toJson() => _$OONIRunV2DescriptorToJson(this);
}

/// Configures the OONI Run v2 task measuring an already available descriptor.
@JsonSerializable()
class OONIRunV2MeasureDescriptorConfig extends BaseConfig {
  /// The OPTIONAL nettest maximum runtime in seconds.
  ///
  /// This setting only applies to nettests that require input, such
  /// as Web Connectivity.
  @JsonKey(name: "max_runtime")
  final int maxRuntime;

  /// This setting allows to OPTIONALLY disable submitting measurements.
  ///
  /// The default is that we submit every measurement we perform.
  @JsonKey(name: "no_collector")
  final bool noCollector;

  /// This setting allows to OPTIONALLY disable saving measurements to disk.
  ///
  /// The default is to save using the file name indicated by ReportFile.
  @JsonKey(name: "no_json")
  final bool noJSON;

  /// OPTIONALLY tells the engine to randomly shuffle the input list.
  @JsonKey(name: "random")
  final bool random;

  /// The OPTIONAL name of the file where to save measurements.
  ///
  /// If this field is empty, we will use 'report.jsonl' as the file name.
  @JsonKey(name: "report_file")
  final String reportFile;

  /// Config for creating a session.
  @JsonKey(name: "session")
  final SessionConfig session;

  /// Descriptor for OONI Run v2
  @JsonKey(name: "descriptor")
  final OONIRunV2Descriptor descriptor;

  /// Default constructor.
  OONIRunV2MeasureDescriptorConfig({
    required this.maxRuntime,
    required this.noCollector,
    required this.noJSON,
    required this.random,
    required this.reportFile,
    required this.session,
    required this.descriptor,
  });

  /// Factory to construct from JSON.
  factory OONIRunV2MeasureDescriptorConfig.fromJson(
          Map<String, dynamic> json) =>
      _$OONIRunV2MeasureDescriptorConfigFromJson(json);

  /// Serialize to JSON.
  Map<String, dynamic> toJson() =>
      _$OONIRunV2MeasureDescriptorConfigToJson(this);
}

/// Provides information about nettests' progress.
@JsonSerializable()
class ProgressEventValue extends BaseEvent {
  /// Number between 0 and 1 indicating the current progress.
  @JsonKey(name: "percentage")
  final double percentage;

  /// Message associated with the current progress indication.
  @JsonKey(name: "message")
  final String message;

  /// Default constructor.
  ProgressEventValue({
    required this.percentage,
    required this.message,
  });

  /// Factory to construct from JSON.
  factory ProgressEventValue.fromJson(Map<String, dynamic> json) =>
      _$ProgressEventValueFromJson(json);

  /// Serialize to JSON.
  Map<String, dynamic> toJson() => _$ProgressEventValueToJson(this);
}

/// Config for creating a new session
@JsonSerializable()
class SessionConfig {
  /// The verbosity level for the sessions's logger.
  ///
  /// It must be one of LogLevelDebug and LogLevelInfo or an empty string. In case
  /// it's an empty string, the code will assume LogLevelInfo.
  @JsonKey(name: "log_level")
  final String logLevel;

  /// The OPTIONAL probe-services URL.
  ///
  /// Leaving this field empty means we're going to use the default URL
  /// for communicating with the OONI backend. You may want to change
  /// this value for testing purposes or to use another backend.
  @JsonKey(name: "probe_services_url")
  final String probeServicesURL;

  /// The OPTIONAL proxy URL.
  ///
  /// Leaving this field empty means we're not using a proxy. You can
  /// use the following proxies:
  ///
  /// 1. socks5://<host>:<port> to use a SOCKS5 proxy;
  ///
  /// 2. tor:/// to launch the tor executable and use its SOCKS5 port;
  ///
  /// 3. psiphon:/// to use the built-in Psiphon client as a proxy.
  ///
  /// On mobile devices, we will use a version of tor that we link as a library
  /// as opposed to using the tor executable. On desktop, you must have
  /// installed the tor executable somewhere in your PATH.
  @JsonKey(name: "proxy_url")
  final String proxyURL;

  /// The MANDATORY name of the tool using this library.
  ///
  /// You MUST specify this field or the session won't be started.
  @JsonKey(name: "software_name")
  final String softwareName;

  /// The MANDATORY version of the tool using this library.
  ///
  /// You MUST specify this field or the session won't be started.
  @JsonKey(name: "software_version")
  final String softwareVersion;

  /// The MANDATORY directory where to store the engine state.
  ///
  /// You MUST specify this field or the session won't be started.
  ///
  /// You MUST create this directory in advance.
  @JsonKey(name: "state_dir")
  final String stateDir;

  /// The MANDATORY directory where to store temporary files.
  ///
  /// You MUST specify this field or the session won't be started.
  ///
  /// You MUST create this directory in advance.
  ///
  /// The session will create a temporary directory _inside_ this directory
  /// and will remove the inner directory when it is finished running.
  @JsonKey(name: "temp_dir")
  final String tempDir;

  /// TorArgs contains OPTIONAL arguments to pass to tor.
  @JsonKey(name: "tor_args")
  final List<String> torArgs;

  /// The OPTIONAL path to the tor binary.
  ///
  /// You can use this field to execute a version of tor that has
  /// not been installed inside your PATH.
  @JsonKey(name: "tor_binary")
  final String torBinary;

  /// The MANDATORY directory where to store persistent tunnel state.
  ///
  /// You MUST specify this field or the session won't be started.
  ///
  /// You MUST create this directory in advance.
  ///
  /// Both psiphon and tor will store information inside this directory when
  /// they're used as a circumention mechanism, i.e., using ProxyURL.
  @JsonKey(name: "tunnel_dir")
  final String tunnelDir;

  /// Default constructor.
  SessionConfig({
    required this.logLevel,
    required this.probeServicesURL,
    required this.proxyURL,
    required this.softwareName,
    required this.softwareVersion,
    required this.stateDir,
    required this.tempDir,
    required this.torArgs,
    required this.torBinary,
    required this.tunnelDir,
  });

  /// Factory to construct from JSON.
  factory SessionConfig.fromJson(Map<String, dynamic> json) =>
      _$SessionConfigFromJson(json);

  /// Serialize to JSON.
  Map<String, dynamic> toJson() => _$SessionConfigToJson(this);
}

/// Contains the results of a measurement submission.
@JsonSerializable()
class SubmitEventValue extends BaseEvent {
  /// Failure that occurred or empty string (on success)
  @JsonKey(name: "failure")
  final String failure;

  /// Index of this measurement relative to the current experiment.
  @JsonKey(name: "index")
  final int index;

  /// The measurement's input.
  @JsonKey(name: "input")
  final String input;

  /// The measurement's report ID.
  @JsonKey(name: "report_id")
  final String reportID;

  /// UTF-8 string containing serialized JSON measurement.
  @JsonKey(name: "measurement")
  final String measurement;

  /// Default constructor.
  SubmitEventValue({
    required this.failure,
    required this.index,
    required this.input,
    required this.reportID,
    required this.measurement,
  });

  /// Factory to construct from JSON.
  factory SubmitEventValue.fromJson(Map<String, dynamic> json) =>
      _$SubmitEventValueFromJson(json);

  /// Serialize to JSON.
  Map<String, dynamic> toJson() => _$SubmitEventValueToJson(this);
}
