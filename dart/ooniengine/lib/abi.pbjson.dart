///
//  Generated code. Do not modify.
//  source: abi.proto
//
// @dart = 2.12
// ignore_for_file: annotate_overrides,camel_case_types,constant_identifier_names,deprecated_member_use_from_same_package,directives_ordering,library_prefixes,non_constant_identifier_names,prefer_final_fields,return_of_invalid_type,unnecessary_const,unnecessary_import,unnecessary_this,unused_import,unused_shown_name

import 'dart:core' as $core;
import 'dart:convert' as $convert;
import 'dart:typed_data' as $typed_data;
@$core.Deprecated('Use logLevelDescriptor instead')
const LogLevel$json = const {
  '1': 'LogLevel',
  '2': const [
    const {'1': 'DEBUG', '2': 0},
    const {'1': 'INFO', '2': 1},
    const {'1': 'WARNING', '2': 2},
  ],
};

/// Descriptor for `LogLevel`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List logLevelDescriptor = $convert.base64Decode('CghMb2dMZXZlbBIJCgVERUJVRxAAEggKBElORk8QARILCgdXQVJOSU5HEAI=');
@$core.Deprecated('Use logEventDescriptor instead')
const LogEvent$json = const {
  '1': 'LogEvent',
  '2': const [
    const {'1': 'level', '3': 1, '4': 1, '5': 14, '6': '.abi.LogLevel', '10': 'level'},
    const {'1': 'message', '3': 2, '4': 1, '5': 9, '10': 'message'},
  ],
};

/// Descriptor for `LogEvent`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List logEventDescriptor = $convert.base64Decode('CghMb2dFdmVudBIjCgVsZXZlbBgBIAEoDjINLmFiaS5Mb2dMZXZlbFIFbGV2ZWwSGAoHbWVzc2FnZRgCIAEoCVIHbWVzc2FnZQ==');
@$core.Deprecated('Use sessionConfigDescriptor instead')
const SessionConfig$json = const {
  '1': 'SessionConfig',
  '2': const [
    const {'1': 'log_level', '3': 1, '4': 1, '5': 14, '6': '.abi.LogLevel', '10': 'logLevel'},
    const {'1': 'probe_services_url', '3': 2, '4': 1, '5': 9, '10': 'probeServicesUrl'},
    const {'1': 'proxy_url', '3': 3, '4': 1, '5': 9, '10': 'proxyUrl'},
    const {'1': 'software_name', '3': 4, '4': 1, '5': 9, '10': 'softwareName'},
    const {'1': 'software_version', '3': 5, '4': 1, '5': 9, '10': 'softwareVersion'},
    const {'1': 'state_dir', '3': 6, '4': 1, '5': 9, '10': 'stateDir'},
    const {'1': 'temp_dir', '3': 7, '4': 1, '5': 9, '10': 'tempDir'},
    const {'1': 'tor_args', '3': 8, '4': 3, '5': 9, '10': 'torArgs'},
    const {'1': 'tor_binary', '3': 9, '4': 1, '5': 9, '10': 'torBinary'},
    const {'1': 'tunnel_dir', '3': 10, '4': 1, '5': 9, '10': 'tunnelDir'},
  ],
};

/// Descriptor for `SessionConfig`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List sessionConfigDescriptor = $convert.base64Decode('Cg1TZXNzaW9uQ29uZmlnEioKCWxvZ19sZXZlbBgBIAEoDjINLmFiaS5Mb2dMZXZlbFIIbG9nTGV2ZWwSLAoScHJvYmVfc2VydmljZXNfdXJsGAIgASgJUhBwcm9iZVNlcnZpY2VzVXJsEhsKCXByb3h5X3VybBgDIAEoCVIIcHJveHlVcmwSIwoNc29mdHdhcmVfbmFtZRgEIAEoCVIMc29mdHdhcmVOYW1lEikKEHNvZnR3YXJlX3ZlcnNpb24YBSABKAlSD3NvZnR3YXJlVmVyc2lvbhIbCglzdGF0ZV9kaXIYBiABKAlSCHN0YXRlRGlyEhkKCHRlbXBfZGlyGAcgASgJUgd0ZW1wRGlyEhkKCHRvcl9hcmdzGAggAygJUgd0b3JBcmdzEh0KCnRvcl9iaW5hcnkYCSABKAlSCXRvckJpbmFyeRIdCgp0dW5uZWxfZGlyGAogASgJUgl0dW5uZWxEaXI=');
@$core.Deprecated('Use geoIPConfigDescriptor instead')
const GeoIPConfig$json = const {
  '1': 'GeoIPConfig',
  '2': const [
    const {'1': 'session', '3': 1, '4': 1, '5': 11, '6': '.abi.SessionConfig', '10': 'session'},
  ],
};

/// Descriptor for `GeoIPConfig`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List geoIPConfigDescriptor = $convert.base64Decode('CgtHZW9JUENvbmZpZxIsCgdzZXNzaW9uGAEgASgLMhIuYWJpLlNlc3Npb25Db25maWdSB3Nlc3Npb24=');
@$core.Deprecated('Use geoIPEventDescriptor instead')
const GeoIPEvent$json = const {
  '1': 'GeoIPEvent',
  '2': const [
    const {'1': 'failure', '3': 1, '4': 1, '5': 9, '10': 'failure'},
    const {'1': 'probe_ip', '3': 2, '4': 1, '5': 9, '10': 'probeIp'},
    const {'1': 'probe_asn', '3': 3, '4': 1, '5': 9, '10': 'probeAsn'},
    const {'1': 'probe_cc', '3': 4, '4': 1, '5': 9, '10': 'probeCc'},
    const {'1': 'probe_network_name', '3': 5, '4': 1, '5': 9, '10': 'probeNetworkName'},
    const {'1': 'resolver_ip', '3': 6, '4': 1, '5': 9, '10': 'resolverIp'},
    const {'1': 'resolver_asn', '3': 7, '4': 1, '5': 9, '10': 'resolverAsn'},
    const {'1': 'resolver_network_name', '3': 8, '4': 1, '5': 9, '10': 'resolverNetworkName'},
  ],
};

/// Descriptor for `GeoIPEvent`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List geoIPEventDescriptor = $convert.base64Decode('CgpHZW9JUEV2ZW50EhgKB2ZhaWx1cmUYASABKAlSB2ZhaWx1cmUSGQoIcHJvYmVfaXAYAiABKAlSB3Byb2JlSXASGwoJcHJvYmVfYXNuGAMgASgJUghwcm9iZUFzbhIZCghwcm9iZV9jYxgEIAEoCVIHcHJvYmVDYxIsChJwcm9iZV9uZXR3b3JrX25hbWUYBSABKAlSEHByb2JlTmV0d29ya05hbWUSHwoLcmVzb2x2ZXJfaXAYBiABKAlSCnJlc29sdmVySXASIQoMcmVzb2x2ZXJfYXNuGAcgASgJUgtyZXNvbHZlckFzbhIyChVyZXNvbHZlcl9uZXR3b3JrX25hbWUYCCABKAlSE3Jlc29sdmVyTmV0d29ya05hbWU=');
@$core.Deprecated('Use experimentMetaInfoConfigDescriptor instead')
const ExperimentMetaInfoConfig$json = const {
  '1': 'ExperimentMetaInfoConfig',
};

/// Descriptor for `ExperimentMetaInfoConfig`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List experimentMetaInfoConfigDescriptor = $convert.base64Decode('ChhFeHBlcmltZW50TWV0YUluZm9Db25maWc=');
@$core.Deprecated('Use experimentMetaInfoEventDescriptor instead')
const ExperimentMetaInfoEvent$json = const {
  '1': 'ExperimentMetaInfoEvent',
  '2': const [
    const {'1': 'name', '3': 1, '4': 1, '5': 9, '10': 'name'},
    const {'1': 'uses_input', '3': 2, '4': 1, '5': 8, '10': 'usesInput'},
  ],
};

/// Descriptor for `ExperimentMetaInfoEvent`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List experimentMetaInfoEventDescriptor = $convert.base64Decode('ChdFeHBlcmltZW50TWV0YUluZm9FdmVudBISCgRuYW1lGAEgASgJUgRuYW1lEh0KCnVzZXNfaW5wdXQYAiABKAhSCXVzZXNJbnB1dA==');
@$core.Deprecated('Use nettestConfigDescriptor instead')
const NettestConfig$json = const {
  '1': 'NettestConfig',
  '2': const [
    const {'1': 'annotations', '3': 1, '4': 3, '5': 11, '6': '.abi.NettestConfig.AnnotationsEntry', '10': 'annotations'},
    const {'1': 'extra_options', '3': 2, '4': 1, '5': 9, '10': 'extraOptions'},
    const {'1': 'inputs', '3': 3, '4': 3, '5': 9, '10': 'inputs'},
    const {'1': 'input_file_paths', '3': 4, '4': 3, '5': 9, '10': 'inputFilePaths'},
    const {'1': 'max_runtime', '3': 5, '4': 1, '5': 3, '10': 'maxRuntime'},
    const {'1': 'name', '3': 6, '4': 1, '5': 9, '10': 'name'},
    const {'1': 'no_collector', '3': 7, '4': 1, '5': 8, '10': 'noCollector'},
    const {'1': 'no_json', '3': 8, '4': 1, '5': 8, '10': 'noJson'},
    const {'1': 'random', '3': 9, '4': 1, '5': 8, '10': 'random'},
    const {'1': 'report_file', '3': 10, '4': 1, '5': 9, '10': 'reportFile'},
    const {'1': 'session', '3': 11, '4': 1, '5': 11, '6': '.abi.SessionConfig', '10': 'session'},
  ],
  '3': const [NettestConfig_AnnotationsEntry$json],
};

@$core.Deprecated('Use nettestConfigDescriptor instead')
const NettestConfig_AnnotationsEntry$json = const {
  '1': 'AnnotationsEntry',
  '2': const [
    const {'1': 'key', '3': 1, '4': 1, '5': 9, '10': 'key'},
    const {'1': 'value', '3': 2, '4': 1, '5': 9, '10': 'value'},
  ],
  '7': const {'7': true},
};

/// Descriptor for `NettestConfig`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List nettestConfigDescriptor = $convert.base64Decode('Cg1OZXR0ZXN0Q29uZmlnEkUKC2Fubm90YXRpb25zGAEgAygLMiMuYWJpLk5ldHRlc3RDb25maWcuQW5ub3RhdGlvbnNFbnRyeVILYW5ub3RhdGlvbnMSIwoNZXh0cmFfb3B0aW9ucxgCIAEoCVIMZXh0cmFPcHRpb25zEhYKBmlucHV0cxgDIAMoCVIGaW5wdXRzEigKEGlucHV0X2ZpbGVfcGF0aHMYBCADKAlSDmlucHV0RmlsZVBhdGhzEh8KC21heF9ydW50aW1lGAUgASgDUgptYXhSdW50aW1lEhIKBG5hbWUYBiABKAlSBG5hbWUSIQoMbm9fY29sbGVjdG9yGAcgASgIUgtub0NvbGxlY3RvchIXCgdub19qc29uGAggASgIUgZub0pzb24SFgoGcmFuZG9tGAkgASgIUgZyYW5kb20SHwoLcmVwb3J0X2ZpbGUYCiABKAlSCnJlcG9ydEZpbGUSLAoHc2Vzc2lvbhgLIAEoCzISLmFiaS5TZXNzaW9uQ29uZmlnUgdzZXNzaW9uGj4KEEFubm90YXRpb25zRW50cnkSEAoDa2V5GAEgASgJUgNrZXkSFAoFdmFsdWUYAiABKAlSBXZhbHVlOgI4AQ==');
@$core.Deprecated('Use progressEventDescriptor instead')
const ProgressEvent$json = const {
  '1': 'ProgressEvent',
  '2': const [
    const {'1': 'percentage', '3': 1, '4': 1, '5': 1, '10': 'percentage'},
    const {'1': 'message', '3': 2, '4': 1, '5': 9, '10': 'message'},
  ],
};

/// Descriptor for `ProgressEvent`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List progressEventDescriptor = $convert.base64Decode('Cg1Qcm9ncmVzc0V2ZW50Eh4KCnBlcmNlbnRhZ2UYASABKAFSCnBlcmNlbnRhZ2USGAoHbWVzc2FnZRgCIAEoCVIHbWVzc2FnZQ==');
@$core.Deprecated('Use dataUsageEventDescriptor instead')
const DataUsageEvent$json = const {
  '1': 'DataUsageEvent',
  '2': const [
    const {'1': 'kibi_bytes_sent', '3': 1, '4': 1, '5': 1, '10': 'kibiBytesSent'},
    const {'1': 'kibi_bytes_received', '3': 2, '4': 1, '5': 1, '10': 'kibiBytesReceived'},
  ],
};

/// Descriptor for `DataUsageEvent`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List dataUsageEventDescriptor = $convert.base64Decode('Cg5EYXRhVXNhZ2VFdmVudBImCg9raWJpX2J5dGVzX3NlbnQYASABKAFSDWtpYmlCeXRlc1NlbnQSLgoTa2liaV9ieXRlc19yZWNlaXZlZBgCIAEoAVIRa2liaUJ5dGVzUmVjZWl2ZWQ=');
@$core.Deprecated('Use submitEventDescriptor instead')
const SubmitEvent$json = const {
  '1': 'SubmitEvent',
  '2': const [
    const {'1': 'not_submitted', '3': 1, '4': 1, '5': 8, '10': 'notSubmitted'},
    const {'1': 'failure', '3': 2, '4': 1, '5': 9, '10': 'failure'},
    const {'1': 'index', '3': 3, '4': 1, '5': 3, '10': 'index'},
    const {'1': 'input', '3': 4, '4': 1, '5': 9, '10': 'input'},
    const {'1': 'report_id', '3': 5, '4': 1, '5': 9, '10': 'reportId'},
    const {'1': 'measurement', '3': 6, '4': 1, '5': 9, '10': 'measurement'},
  ],
};

/// Descriptor for `SubmitEvent`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List submitEventDescriptor = $convert.base64Decode('CgtTdWJtaXRFdmVudBIjCg1ub3Rfc3VibWl0dGVkGAEgASgIUgxub3RTdWJtaXR0ZWQSGAoHZmFpbHVyZRgCIAEoCVIHZmFpbHVyZRIUCgVpbmRleBgDIAEoA1IFaW5kZXgSFAoFaW5wdXQYBCABKAlSBWlucHV0EhsKCXJlcG9ydF9pZBgFIAEoCVIIcmVwb3J0SWQSIAoLbWVhc3VyZW1lbnQYBiABKAlSC21lYXN1cmVtZW50');
@$core.Deprecated('Use oONIRunV2DescriptorNettestDescriptor instead')
const OONIRunV2DescriptorNettest$json = const {
  '1': 'OONIRunV2DescriptorNettest',
  '2': const [
    const {'1': 'annotations', '3': 1, '4': 3, '5': 11, '6': '.abi.OONIRunV2DescriptorNettest.AnnotationsEntry', '10': 'annotations'},
    const {'1': 'inputs', '3': 2, '4': 3, '5': 9, '10': 'inputs'},
    const {'1': 'options', '3': 3, '4': 1, '5': 9, '10': 'options'},
    const {'1': 'test_name', '3': 4, '4': 1, '5': 9, '10': 'testName'},
  ],
  '3': const [OONIRunV2DescriptorNettest_AnnotationsEntry$json],
};

@$core.Deprecated('Use oONIRunV2DescriptorNettestDescriptor instead')
const OONIRunV2DescriptorNettest_AnnotationsEntry$json = const {
  '1': 'AnnotationsEntry',
  '2': const [
    const {'1': 'key', '3': 1, '4': 1, '5': 9, '10': 'key'},
    const {'1': 'value', '3': 2, '4': 1, '5': 9, '10': 'value'},
  ],
  '7': const {'7': true},
};

/// Descriptor for `OONIRunV2DescriptorNettest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List oONIRunV2DescriptorNettestDescriptor = $convert.base64Decode('ChpPT05JUnVuVjJEZXNjcmlwdG9yTmV0dGVzdBJSCgthbm5vdGF0aW9ucxgBIAMoCzIwLmFiaS5PT05JUnVuVjJEZXNjcmlwdG9yTmV0dGVzdC5Bbm5vdGF0aW9uc0VudHJ5Ugthbm5vdGF0aW9ucxIWCgZpbnB1dHMYAiADKAlSBmlucHV0cxIYCgdvcHRpb25zGAMgASgJUgdvcHRpb25zEhsKCXRlc3RfbmFtZRgEIAEoCVIIdGVzdE5hbWUaPgoQQW5ub3RhdGlvbnNFbnRyeRIQCgNrZXkYASABKAlSA2tleRIUCgV2YWx1ZRgCIAEoCVIFdmFsdWU6AjgB');
@$core.Deprecated('Use oONIRunV2DescriptorDescriptor instead')
const OONIRunV2Descriptor$json = const {
  '1': 'OONIRunV2Descriptor',
  '2': const [
    const {'1': 'name', '3': 1, '4': 1, '5': 9, '10': 'name'},
    const {'1': 'description', '3': 2, '4': 1, '5': 9, '10': 'description'},
    const {'1': 'author', '3': 3, '4': 1, '5': 9, '10': 'author'},
    const {'1': 'nettests', '3': 4, '4': 3, '5': 11, '6': '.abi.OONIRunV2DescriptorNettest', '10': 'nettests'},
  ],
};

/// Descriptor for `OONIRunV2Descriptor`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List oONIRunV2DescriptorDescriptor = $convert.base64Decode('ChNPT05JUnVuVjJEZXNjcmlwdG9yEhIKBG5hbWUYASABKAlSBG5hbWUSIAoLZGVzY3JpcHRpb24YAiABKAlSC2Rlc2NyaXB0aW9uEhYKBmF1dGhvchgDIAEoCVIGYXV0aG9yEjsKCG5ldHRlc3RzGAQgAygLMh8uYWJpLk9PTklSdW5WMkRlc2NyaXB0b3JOZXR0ZXN0UghuZXR0ZXN0cw==');
@$core.Deprecated('Use oONIRunV2MeasureDescriptorConfigDescriptor instead')
const OONIRunV2MeasureDescriptorConfig$json = const {
  '1': 'OONIRunV2MeasureDescriptorConfig',
  '2': const [
    const {'1': 'max_runtime', '3': 1, '4': 1, '5': 3, '10': 'maxRuntime'},
    const {'1': 'no_collector', '3': 2, '4': 1, '5': 8, '10': 'noCollector'},
    const {'1': 'no_json', '3': 3, '4': 1, '5': 8, '10': 'noJson'},
    const {'1': 'random', '3': 4, '4': 1, '5': 8, '10': 'random'},
    const {'1': 'report_file', '3': 5, '4': 1, '5': 9, '10': 'reportFile'},
    const {'1': 'session', '3': 6, '4': 1, '5': 11, '6': '.abi.SessionConfig', '10': 'session'},
    const {'1': 'v2_descriptor', '3': 7, '4': 1, '5': 11, '6': '.abi.OONIRunV2Descriptor', '10': 'v2Descriptor'},
  ],
};

/// Descriptor for `OONIRunV2MeasureDescriptorConfig`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List oONIRunV2MeasureDescriptorConfigDescriptor = $convert.base64Decode('CiBPT05JUnVuVjJNZWFzdXJlRGVzY3JpcHRvckNvbmZpZxIfCgttYXhfcnVudGltZRgBIAEoA1IKbWF4UnVudGltZRIhCgxub19jb2xsZWN0b3IYAiABKAhSC25vQ29sbGVjdG9yEhcKB25vX2pzb24YAyABKAhSBm5vSnNvbhIWCgZyYW5kb20YBCABKAhSBnJhbmRvbRIfCgtyZXBvcnRfZmlsZRgFIAEoCVIKcmVwb3J0RmlsZRIsCgdzZXNzaW9uGAYgASgLMhIuYWJpLlNlc3Npb25Db25maWdSB3Nlc3Npb24SPQoNdjJfZGVzY3JpcHRvchgHIAEoCzIYLmFiaS5PT05JUnVuVjJEZXNjcmlwdG9yUgx2MkRlc2NyaXB0b3I=');
