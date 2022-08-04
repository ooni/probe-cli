// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'abi.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

LogEventValue _$LogEventValueFromJson(Map<String, dynamic> json) =>
    LogEventValue(
      level: json['level'] as String,
      message: json['message'] as String,
    );

Map<String, dynamic> _$LogEventValueToJson(LogEventValue instance) =>
    <String, dynamic>{
      'level': instance.level,
      'message': instance.message,
    };

DataUsageEventValue _$DataUsageEventValueFromJson(Map<String, dynamic> json) =>
    DataUsageEventValue(
      kibiBytesSent: (json['kibi_bytes_sent'] as num).toDouble(),
      kibiBytesReceived: (json['kibi_bytes_received'] as num).toDouble(),
    );

Map<String, dynamic> _$DataUsageEventValueToJson(
        DataUsageEventValue instance) =>
    <String, dynamic>{
      'kibi_bytes_sent': instance.kibiBytesSent,
      'kibi_bytes_received': instance.kibiBytesReceived,
    };

GeoIPEventValue _$GeoIPEventValueFromJson(Map<String, dynamic> json) =>
    GeoIPEventValue(
      failure: json['failure'] as String,
      probeIP: json['probe_ip'] as String,
      probeASN: json['probe_asn'] as String,
      probeCC: json['probe_cc'] as String,
      probeNetworkName: json['probe_network_name'] as String,
      resolverIP: json['resolver_ip'] as String,
      resolverASN: json['resolver_asn'] as String,
      resolverNetworkName: json['resolver_network_name'] as String,
    );

Map<String, dynamic> _$GeoIPEventValueToJson(GeoIPEventValue instance) =>
    <String, dynamic>{
      'failure': instance.failure,
      'probe_ip': instance.probeIP,
      'probe_asn': instance.probeASN,
      'probe_cc': instance.probeCC,
      'probe_network_name': instance.probeNetworkName,
      'resolver_ip': instance.resolverIP,
      'resolver_asn': instance.resolverASN,
      'resolver_network_name': instance.resolverNetworkName,
    };

GeoIPConfig _$GeoIPConfigFromJson(Map<String, dynamic> json) => GeoIPConfig(
      session: SessionConfig.fromJson(json['session'] as Map<String, dynamic>),
    );

Map<String, dynamic> _$GeoIPConfigToJson(GeoIPConfig instance) =>
    <String, dynamic>{
      'session': instance.session,
    };

MetaInfoExperimentConfig _$MetaInfoExperimentConfigFromJson(
        Map<String, dynamic> json) =>
    MetaInfoExperimentConfig();

Map<String, dynamic> _$MetaInfoExperimentConfigToJson(
        MetaInfoExperimentConfig instance) =>
    <String, dynamic>{};

MetaInfoExperimentEventValue _$MetaInfoExperimentEventValueFromJson(
        Map<String, dynamic> json) =>
    MetaInfoExperimentEventValue(
      name: json['name'] as String,
      usesInput: json['uses_input'] as bool,
    );

Map<String, dynamic> _$MetaInfoExperimentEventValueToJson(
        MetaInfoExperimentEventValue instance) =>
    <String, dynamic>{
      'name': instance.name,
      'uses_input': instance.usesInput,
    };

BaseEvent _$BaseEventFromJson(Map<String, dynamic> json) => BaseEvent();

Map<String, dynamic> _$BaseEventToJson(BaseEvent instance) =>
    <String, dynamic>{};

BaseConfig _$BaseConfigFromJson(Map<String, dynamic> json) => BaseConfig();

Map<String, dynamic> _$BaseConfigToJson(BaseConfig instance) =>
    <String, dynamic>{};

NettestConfig _$NettestConfigFromJson(Map<String, dynamic> json) =>
    NettestConfig(
      annotations: Map<String, String>.from(json['annotations'] as Map),
      extraOptions: (json['extra_options'] as Map<String, dynamic>).map(
        (k, e) => MapEntry(k, e as Object),
      ),
      inputs:
          (json['inputs'] as List<dynamic>).map((e) => e as String).toList(),
      inputFilePaths: (json['input_file_paths'] as List<dynamic>)
          .map((e) => e as String)
          .toList(),
      maxRuntime: json['max_runtime'] as int,
      name: json['name'] as String,
      noCollector: json['no_collector'] as bool,
      noJSON: json['no_json'] as bool,
      random: json['random'] as bool,
      reportFile: json['report_file'] as String,
      session: SessionConfig.fromJson(json['session'] as Map<String, dynamic>),
    );

Map<String, dynamic> _$NettestConfigToJson(NettestConfig instance) =>
    <String, dynamic>{
      'annotations': instance.annotations,
      'extra_options': instance.extraOptions,
      'inputs': instance.inputs,
      'input_file_paths': instance.inputFilePaths,
      'max_runtime': instance.maxRuntime,
      'name': instance.name,
      'no_collector': instance.noCollector,
      'no_json': instance.noJSON,
      'random': instance.random,
      'report_file': instance.reportFile,
      'session': instance.session,
    };

OONIRunV2Nettest _$OONIRunV2NettestFromJson(Map<String, dynamic> json) =>
    OONIRunV2Nettest(
      annotations: Map<String, String>.from(json['annotations'] as Map),
      inputs:
          (json['inputs'] as List<dynamic>).map((e) => e as String).toList(),
      options: (json['options'] as Map<String, dynamic>).map(
        (k, e) => MapEntry(k, e as Object),
      ),
      testName: json['test_name'] as String,
    );

Map<String, dynamic> _$OONIRunV2NettestToJson(OONIRunV2Nettest instance) =>
    <String, dynamic>{
      'annotations': instance.annotations,
      'inputs': instance.inputs,
      'options': instance.options,
      'test_name': instance.testName,
    };

OONIRunV2Descriptor _$OONIRunV2DescriptorFromJson(Map<String, dynamic> json) =>
    OONIRunV2Descriptor(
      name: json['name'] as String,
      description: json['description'] as String,
      author: json['author'] as String,
      nettests: (json['nettests'] as List<dynamic>)
          .map((e) => OONIRunV2Nettest.fromJson(e as Map<String, dynamic>))
          .toList(),
    );

Map<String, dynamic> _$OONIRunV2DescriptorToJson(
        OONIRunV2Descriptor instance) =>
    <String, dynamic>{
      'name': instance.name,
      'description': instance.description,
      'author': instance.author,
      'nettests': instance.nettests,
    };

OONIRunV2MeasureDescriptorConfig _$OONIRunV2MeasureDescriptorConfigFromJson(
        Map<String, dynamic> json) =>
    OONIRunV2MeasureDescriptorConfig(
      maxRuntime: json['max_runtime'] as int,
      noCollector: json['no_collector'] as bool,
      noJSON: json['no_json'] as bool,
      random: json['random'] as bool,
      reportFile: json['report_file'] as String,
      session: SessionConfig.fromJson(json['session'] as Map<String, dynamic>),
      descriptor: OONIRunV2Descriptor.fromJson(
          json['descriptor'] as Map<String, dynamic>),
    );

Map<String, dynamic> _$OONIRunV2MeasureDescriptorConfigToJson(
        OONIRunV2MeasureDescriptorConfig instance) =>
    <String, dynamic>{
      'max_runtime': instance.maxRuntime,
      'no_collector': instance.noCollector,
      'no_json': instance.noJSON,
      'random': instance.random,
      'report_file': instance.reportFile,
      'session': instance.session,
      'descriptor': instance.descriptor,
    };

ProgressEventValue _$ProgressEventValueFromJson(Map<String, dynamic> json) =>
    ProgressEventValue(
      percentage: (json['percentage'] as num).toDouble(),
      message: json['message'] as String,
    );

Map<String, dynamic> _$ProgressEventValueToJson(ProgressEventValue instance) =>
    <String, dynamic>{
      'percentage': instance.percentage,
      'message': instance.message,
    };

SessionConfig _$SessionConfigFromJson(Map<String, dynamic> json) =>
    SessionConfig(
      logLevel: json['log_level'] as String,
      probeServicesURL: json['probe_services_url'] as String,
      proxyURL: json['proxy_url'] as String,
      softwareName: json['software_name'] as String,
      softwareVersion: json['software_version'] as String,
      stateDir: json['state_dir'] as String,
      tempDir: json['temp_dir'] as String,
      torArgs:
          (json['tor_args'] as List<dynamic>).map((e) => e as String).toList(),
      torBinary: json['tor_binary'] as String,
      tunnelDir: json['tunnel_dir'] as String,
    );

Map<String, dynamic> _$SessionConfigToJson(SessionConfig instance) =>
    <String, dynamic>{
      'log_level': instance.logLevel,
      'probe_services_url': instance.probeServicesURL,
      'proxy_url': instance.proxyURL,
      'software_name': instance.softwareName,
      'software_version': instance.softwareVersion,
      'state_dir': instance.stateDir,
      'temp_dir': instance.tempDir,
      'tor_args': instance.torArgs,
      'tor_binary': instance.torBinary,
      'tunnel_dir': instance.tunnelDir,
    };

SubmitEventValue _$SubmitEventValueFromJson(Map<String, dynamic> json) =>
    SubmitEventValue(
      failure: json['failure'] as String,
      index: json['index'] as int,
      input: json['input'] as String,
      reportID: json['report_id'] as String,
      measurement: json['measurement'] as String,
    );

Map<String, dynamic> _$SubmitEventValueToJson(SubmitEventValue instance) =>
    <String, dynamic>{
      'failure': instance.failure,
      'index': instance.index,
      'input': instance.input,
      'report_id': instance.reportID,
      'measurement': instance.measurement,
    };
