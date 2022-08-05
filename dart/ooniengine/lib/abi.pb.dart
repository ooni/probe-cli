///
//  Generated code. Do not modify.
//  source: abi.proto
//
// @dart = 2.12
// ignore_for_file: annotate_overrides,camel_case_types,constant_identifier_names,directives_ordering,library_prefixes,non_constant_identifier_names,prefer_final_fields,return_of_invalid_type,unnecessary_const,unnecessary_import,unnecessary_this,unused_import,unused_shown_name

import 'dart:core' as $core;

import 'package:fixnum/fixnum.dart' as $fixnum;
import 'package:protobuf/protobuf.dart' as $pb;

import 'abi.pbenum.dart';

export 'abi.pbenum.dart';

class LogEvent extends $pb.GeneratedMessage {
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(const $core.bool.fromEnvironment('protobuf.omit_message_names') ? '' : 'LogEvent', package: const $pb.PackageName(const $core.bool.fromEnvironment('protobuf.omit_message_names') ? '' : 'abi'), createEmptyInstance: create)
    ..e<LogLevel>(1, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'level', $pb.PbFieldType.OE, defaultOrMaker: LogLevel.DEBUG, valueOf: LogLevel.valueOf, enumValues: LogLevel.values)
    ..aOS(2, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'message')
    ..hasRequiredFields = false
  ;

  LogEvent._() : super();
  factory LogEvent({
    LogLevel? level,
    $core.String? message,
  }) {
    final _result = create();
    if (level != null) {
      _result.level = level;
    }
    if (message != null) {
      _result.message = message;
    }
    return _result;
  }
  factory LogEvent.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory LogEvent.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  LogEvent clone() => LogEvent()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  LogEvent copyWith(void Function(LogEvent) updates) => super.copyWith((message) => updates(message as LogEvent)) as LogEvent; // ignore: deprecated_member_use
  $pb.BuilderInfo get info_ => _i;
  @$core.pragma('dart2js:noInline')
  static LogEvent create() => LogEvent._();
  LogEvent createEmptyInstance() => create();
  static $pb.PbList<LogEvent> createRepeated() => $pb.PbList<LogEvent>();
  @$core.pragma('dart2js:noInline')
  static LogEvent getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<LogEvent>(create);
  static LogEvent? _defaultInstance;

  @$pb.TagNumber(1)
  LogLevel get level => $_getN(0);
  @$pb.TagNumber(1)
  set level(LogLevel v) { setField(1, v); }
  @$pb.TagNumber(1)
  $core.bool hasLevel() => $_has(0);
  @$pb.TagNumber(1)
  void clearLevel() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get message => $_getSZ(1);
  @$pb.TagNumber(2)
  set message($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasMessage() => $_has(1);
  @$pb.TagNumber(2)
  void clearMessage() => clearField(2);
}

class SessionConfig extends $pb.GeneratedMessage {
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(const $core.bool.fromEnvironment('protobuf.omit_message_names') ? '' : 'SessionConfig', package: const $pb.PackageName(const $core.bool.fromEnvironment('protobuf.omit_message_names') ? '' : 'abi'), createEmptyInstance: create)
    ..e<LogLevel>(1, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'logLevel', $pb.PbFieldType.OE, defaultOrMaker: LogLevel.DEBUG, valueOf: LogLevel.valueOf, enumValues: LogLevel.values)
    ..aOS(2, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'probeServicesUrl')
    ..aOS(3, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'proxyUrl')
    ..aOS(4, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'softwareName')
    ..aOS(5, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'softwareVersion')
    ..aOS(6, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'stateDir')
    ..aOS(7, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'tempDir')
    ..pPS(8, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'torArgs')
    ..aOS(9, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'torBinary')
    ..aOS(10, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'tunnelDir')
    ..hasRequiredFields = false
  ;

  SessionConfig._() : super();
  factory SessionConfig({
    LogLevel? logLevel,
    $core.String? probeServicesUrl,
    $core.String? proxyUrl,
    $core.String? softwareName,
    $core.String? softwareVersion,
    $core.String? stateDir,
    $core.String? tempDir,
    $core.Iterable<$core.String>? torArgs,
    $core.String? torBinary,
    $core.String? tunnelDir,
  }) {
    final _result = create();
    if (logLevel != null) {
      _result.logLevel = logLevel;
    }
    if (probeServicesUrl != null) {
      _result.probeServicesUrl = probeServicesUrl;
    }
    if (proxyUrl != null) {
      _result.proxyUrl = proxyUrl;
    }
    if (softwareName != null) {
      _result.softwareName = softwareName;
    }
    if (softwareVersion != null) {
      _result.softwareVersion = softwareVersion;
    }
    if (stateDir != null) {
      _result.stateDir = stateDir;
    }
    if (tempDir != null) {
      _result.tempDir = tempDir;
    }
    if (torArgs != null) {
      _result.torArgs.addAll(torArgs);
    }
    if (torBinary != null) {
      _result.torBinary = torBinary;
    }
    if (tunnelDir != null) {
      _result.tunnelDir = tunnelDir;
    }
    return _result;
  }
  factory SessionConfig.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory SessionConfig.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  SessionConfig clone() => SessionConfig()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  SessionConfig copyWith(void Function(SessionConfig) updates) => super.copyWith((message) => updates(message as SessionConfig)) as SessionConfig; // ignore: deprecated_member_use
  $pb.BuilderInfo get info_ => _i;
  @$core.pragma('dart2js:noInline')
  static SessionConfig create() => SessionConfig._();
  SessionConfig createEmptyInstance() => create();
  static $pb.PbList<SessionConfig> createRepeated() => $pb.PbList<SessionConfig>();
  @$core.pragma('dart2js:noInline')
  static SessionConfig getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<SessionConfig>(create);
  static SessionConfig? _defaultInstance;

  @$pb.TagNumber(1)
  LogLevel get logLevel => $_getN(0);
  @$pb.TagNumber(1)
  set logLevel(LogLevel v) { setField(1, v); }
  @$pb.TagNumber(1)
  $core.bool hasLogLevel() => $_has(0);
  @$pb.TagNumber(1)
  void clearLogLevel() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get probeServicesUrl => $_getSZ(1);
  @$pb.TagNumber(2)
  set probeServicesUrl($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasProbeServicesUrl() => $_has(1);
  @$pb.TagNumber(2)
  void clearProbeServicesUrl() => clearField(2);

  @$pb.TagNumber(3)
  $core.String get proxyUrl => $_getSZ(2);
  @$pb.TagNumber(3)
  set proxyUrl($core.String v) { $_setString(2, v); }
  @$pb.TagNumber(3)
  $core.bool hasProxyUrl() => $_has(2);
  @$pb.TagNumber(3)
  void clearProxyUrl() => clearField(3);

  @$pb.TagNumber(4)
  $core.String get softwareName => $_getSZ(3);
  @$pb.TagNumber(4)
  set softwareName($core.String v) { $_setString(3, v); }
  @$pb.TagNumber(4)
  $core.bool hasSoftwareName() => $_has(3);
  @$pb.TagNumber(4)
  void clearSoftwareName() => clearField(4);

  @$pb.TagNumber(5)
  $core.String get softwareVersion => $_getSZ(4);
  @$pb.TagNumber(5)
  set softwareVersion($core.String v) { $_setString(4, v); }
  @$pb.TagNumber(5)
  $core.bool hasSoftwareVersion() => $_has(4);
  @$pb.TagNumber(5)
  void clearSoftwareVersion() => clearField(5);

  @$pb.TagNumber(6)
  $core.String get stateDir => $_getSZ(5);
  @$pb.TagNumber(6)
  set stateDir($core.String v) { $_setString(5, v); }
  @$pb.TagNumber(6)
  $core.bool hasStateDir() => $_has(5);
  @$pb.TagNumber(6)
  void clearStateDir() => clearField(6);

  @$pb.TagNumber(7)
  $core.String get tempDir => $_getSZ(6);
  @$pb.TagNumber(7)
  set tempDir($core.String v) { $_setString(6, v); }
  @$pb.TagNumber(7)
  $core.bool hasTempDir() => $_has(6);
  @$pb.TagNumber(7)
  void clearTempDir() => clearField(7);

  @$pb.TagNumber(8)
  $core.List<$core.String> get torArgs => $_getList(7);

  @$pb.TagNumber(9)
  $core.String get torBinary => $_getSZ(8);
  @$pb.TagNumber(9)
  set torBinary($core.String v) { $_setString(8, v); }
  @$pb.TagNumber(9)
  $core.bool hasTorBinary() => $_has(8);
  @$pb.TagNumber(9)
  void clearTorBinary() => clearField(9);

  @$pb.TagNumber(10)
  $core.String get tunnelDir => $_getSZ(9);
  @$pb.TagNumber(10)
  set tunnelDir($core.String v) { $_setString(9, v); }
  @$pb.TagNumber(10)
  $core.bool hasTunnelDir() => $_has(9);
  @$pb.TagNumber(10)
  void clearTunnelDir() => clearField(10);
}

class GeoIPConfig extends $pb.GeneratedMessage {
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(const $core.bool.fromEnvironment('protobuf.omit_message_names') ? '' : 'GeoIPConfig', package: const $pb.PackageName(const $core.bool.fromEnvironment('protobuf.omit_message_names') ? '' : 'abi'), createEmptyInstance: create)
    ..aOM<SessionConfig>(1, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'session', subBuilder: SessionConfig.create)
    ..hasRequiredFields = false
  ;

  GeoIPConfig._() : super();
  factory GeoIPConfig({
    SessionConfig? session,
  }) {
    final _result = create();
    if (session != null) {
      _result.session = session;
    }
    return _result;
  }
  factory GeoIPConfig.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory GeoIPConfig.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  GeoIPConfig clone() => GeoIPConfig()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  GeoIPConfig copyWith(void Function(GeoIPConfig) updates) => super.copyWith((message) => updates(message as GeoIPConfig)) as GeoIPConfig; // ignore: deprecated_member_use
  $pb.BuilderInfo get info_ => _i;
  @$core.pragma('dart2js:noInline')
  static GeoIPConfig create() => GeoIPConfig._();
  GeoIPConfig createEmptyInstance() => create();
  static $pb.PbList<GeoIPConfig> createRepeated() => $pb.PbList<GeoIPConfig>();
  @$core.pragma('dart2js:noInline')
  static GeoIPConfig getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<GeoIPConfig>(create);
  static GeoIPConfig? _defaultInstance;

  @$pb.TagNumber(1)
  SessionConfig get session => $_getN(0);
  @$pb.TagNumber(1)
  set session(SessionConfig v) { setField(1, v); }
  @$pb.TagNumber(1)
  $core.bool hasSession() => $_has(0);
  @$pb.TagNumber(1)
  void clearSession() => clearField(1);
  @$pb.TagNumber(1)
  SessionConfig ensureSession() => $_ensure(0);
}

class GeoIPEvent extends $pb.GeneratedMessage {
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(const $core.bool.fromEnvironment('protobuf.omit_message_names') ? '' : 'GeoIPEvent', package: const $pb.PackageName(const $core.bool.fromEnvironment('protobuf.omit_message_names') ? '' : 'abi'), createEmptyInstance: create)
    ..aOS(1, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'failure')
    ..aOS(2, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'probeIp')
    ..aOS(3, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'probeAsn')
    ..aOS(4, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'probeCc')
    ..aOS(5, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'probeNetworkName')
    ..aOS(6, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'resolverIp')
    ..aOS(7, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'resolverAsn')
    ..aOS(8, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'resolverNetworkName')
    ..hasRequiredFields = false
  ;

  GeoIPEvent._() : super();
  factory GeoIPEvent({
    $core.String? failure,
    $core.String? probeIp,
    $core.String? probeAsn,
    $core.String? probeCc,
    $core.String? probeNetworkName,
    $core.String? resolverIp,
    $core.String? resolverAsn,
    $core.String? resolverNetworkName,
  }) {
    final _result = create();
    if (failure != null) {
      _result.failure = failure;
    }
    if (probeIp != null) {
      _result.probeIp = probeIp;
    }
    if (probeAsn != null) {
      _result.probeAsn = probeAsn;
    }
    if (probeCc != null) {
      _result.probeCc = probeCc;
    }
    if (probeNetworkName != null) {
      _result.probeNetworkName = probeNetworkName;
    }
    if (resolverIp != null) {
      _result.resolverIp = resolverIp;
    }
    if (resolverAsn != null) {
      _result.resolverAsn = resolverAsn;
    }
    if (resolverNetworkName != null) {
      _result.resolverNetworkName = resolverNetworkName;
    }
    return _result;
  }
  factory GeoIPEvent.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory GeoIPEvent.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  GeoIPEvent clone() => GeoIPEvent()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  GeoIPEvent copyWith(void Function(GeoIPEvent) updates) => super.copyWith((message) => updates(message as GeoIPEvent)) as GeoIPEvent; // ignore: deprecated_member_use
  $pb.BuilderInfo get info_ => _i;
  @$core.pragma('dart2js:noInline')
  static GeoIPEvent create() => GeoIPEvent._();
  GeoIPEvent createEmptyInstance() => create();
  static $pb.PbList<GeoIPEvent> createRepeated() => $pb.PbList<GeoIPEvent>();
  @$core.pragma('dart2js:noInline')
  static GeoIPEvent getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<GeoIPEvent>(create);
  static GeoIPEvent? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get failure => $_getSZ(0);
  @$pb.TagNumber(1)
  set failure($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasFailure() => $_has(0);
  @$pb.TagNumber(1)
  void clearFailure() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get probeIp => $_getSZ(1);
  @$pb.TagNumber(2)
  set probeIp($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasProbeIp() => $_has(1);
  @$pb.TagNumber(2)
  void clearProbeIp() => clearField(2);

  @$pb.TagNumber(3)
  $core.String get probeAsn => $_getSZ(2);
  @$pb.TagNumber(3)
  set probeAsn($core.String v) { $_setString(2, v); }
  @$pb.TagNumber(3)
  $core.bool hasProbeAsn() => $_has(2);
  @$pb.TagNumber(3)
  void clearProbeAsn() => clearField(3);

  @$pb.TagNumber(4)
  $core.String get probeCc => $_getSZ(3);
  @$pb.TagNumber(4)
  set probeCc($core.String v) { $_setString(3, v); }
  @$pb.TagNumber(4)
  $core.bool hasProbeCc() => $_has(3);
  @$pb.TagNumber(4)
  void clearProbeCc() => clearField(4);

  @$pb.TagNumber(5)
  $core.String get probeNetworkName => $_getSZ(4);
  @$pb.TagNumber(5)
  set probeNetworkName($core.String v) { $_setString(4, v); }
  @$pb.TagNumber(5)
  $core.bool hasProbeNetworkName() => $_has(4);
  @$pb.TagNumber(5)
  void clearProbeNetworkName() => clearField(5);

  @$pb.TagNumber(6)
  $core.String get resolverIp => $_getSZ(5);
  @$pb.TagNumber(6)
  set resolverIp($core.String v) { $_setString(5, v); }
  @$pb.TagNumber(6)
  $core.bool hasResolverIp() => $_has(5);
  @$pb.TagNumber(6)
  void clearResolverIp() => clearField(6);

  @$pb.TagNumber(7)
  $core.String get resolverAsn => $_getSZ(6);
  @$pb.TagNumber(7)
  set resolverAsn($core.String v) { $_setString(6, v); }
  @$pb.TagNumber(7)
  $core.bool hasResolverAsn() => $_has(6);
  @$pb.TagNumber(7)
  void clearResolverAsn() => clearField(7);

  @$pb.TagNumber(8)
  $core.String get resolverNetworkName => $_getSZ(7);
  @$pb.TagNumber(8)
  set resolverNetworkName($core.String v) { $_setString(7, v); }
  @$pb.TagNumber(8)
  $core.bool hasResolverNetworkName() => $_has(7);
  @$pb.TagNumber(8)
  void clearResolverNetworkName() => clearField(8);
}

class ExperimentMetaInfoRequest extends $pb.GeneratedMessage {
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(const $core.bool.fromEnvironment('protobuf.omit_message_names') ? '' : 'ExperimentMetaInfoRequest', package: const $pb.PackageName(const $core.bool.fromEnvironment('protobuf.omit_message_names') ? '' : 'abi'), createEmptyInstance: create)
    ..hasRequiredFields = false
  ;

  ExperimentMetaInfoRequest._() : super();
  factory ExperimentMetaInfoRequest() => create();
  factory ExperimentMetaInfoRequest.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory ExperimentMetaInfoRequest.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  ExperimentMetaInfoRequest clone() => ExperimentMetaInfoRequest()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  ExperimentMetaInfoRequest copyWith(void Function(ExperimentMetaInfoRequest) updates) => super.copyWith((message) => updates(message as ExperimentMetaInfoRequest)) as ExperimentMetaInfoRequest; // ignore: deprecated_member_use
  $pb.BuilderInfo get info_ => _i;
  @$core.pragma('dart2js:noInline')
  static ExperimentMetaInfoRequest create() => ExperimentMetaInfoRequest._();
  ExperimentMetaInfoRequest createEmptyInstance() => create();
  static $pb.PbList<ExperimentMetaInfoRequest> createRepeated() => $pb.PbList<ExperimentMetaInfoRequest>();
  @$core.pragma('dart2js:noInline')
  static ExperimentMetaInfoRequest getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<ExperimentMetaInfoRequest>(create);
  static ExperimentMetaInfoRequest? _defaultInstance;
}

class ExperimentMetaInfoEntry extends $pb.GeneratedMessage {
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(const $core.bool.fromEnvironment('protobuf.omit_message_names') ? '' : 'ExperimentMetaInfoEntry', package: const $pb.PackageName(const $core.bool.fromEnvironment('protobuf.omit_message_names') ? '' : 'abi'), createEmptyInstance: create)
    ..aOS(1, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'name')
    ..aOB(2, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'usesInput')
    ..hasRequiredFields = false
  ;

  ExperimentMetaInfoEntry._() : super();
  factory ExperimentMetaInfoEntry({
    $core.String? name,
    $core.bool? usesInput,
  }) {
    final _result = create();
    if (name != null) {
      _result.name = name;
    }
    if (usesInput != null) {
      _result.usesInput = usesInput;
    }
    return _result;
  }
  factory ExperimentMetaInfoEntry.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory ExperimentMetaInfoEntry.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  ExperimentMetaInfoEntry clone() => ExperimentMetaInfoEntry()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  ExperimentMetaInfoEntry copyWith(void Function(ExperimentMetaInfoEntry) updates) => super.copyWith((message) => updates(message as ExperimentMetaInfoEntry)) as ExperimentMetaInfoEntry; // ignore: deprecated_member_use
  $pb.BuilderInfo get info_ => _i;
  @$core.pragma('dart2js:noInline')
  static ExperimentMetaInfoEntry create() => ExperimentMetaInfoEntry._();
  ExperimentMetaInfoEntry createEmptyInstance() => create();
  static $pb.PbList<ExperimentMetaInfoEntry> createRepeated() => $pb.PbList<ExperimentMetaInfoEntry>();
  @$core.pragma('dart2js:noInline')
  static ExperimentMetaInfoEntry getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<ExperimentMetaInfoEntry>(create);
  static ExperimentMetaInfoEntry? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get name => $_getSZ(0);
  @$pb.TagNumber(1)
  set name($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasName() => $_has(0);
  @$pb.TagNumber(1)
  void clearName() => clearField(1);

  @$pb.TagNumber(2)
  $core.bool get usesInput => $_getBF(1);
  @$pb.TagNumber(2)
  set usesInput($core.bool v) { $_setBool(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasUsesInput() => $_has(1);
  @$pb.TagNumber(2)
  void clearUsesInput() => clearField(2);
}

class ExperimentMetaInfoResponse extends $pb.GeneratedMessage {
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(const $core.bool.fromEnvironment('protobuf.omit_message_names') ? '' : 'ExperimentMetaInfoResponse', package: const $pb.PackageName(const $core.bool.fromEnvironment('protobuf.omit_message_names') ? '' : 'abi'), createEmptyInstance: create)
    ..pc<ExperimentMetaInfoEntry>(1, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'entry', $pb.PbFieldType.PM, subBuilder: ExperimentMetaInfoEntry.create)
    ..hasRequiredFields = false
  ;

  ExperimentMetaInfoResponse._() : super();
  factory ExperimentMetaInfoResponse({
    $core.Iterable<ExperimentMetaInfoEntry>? entry,
  }) {
    final _result = create();
    if (entry != null) {
      _result.entry.addAll(entry);
    }
    return _result;
  }
  factory ExperimentMetaInfoResponse.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory ExperimentMetaInfoResponse.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  ExperimentMetaInfoResponse clone() => ExperimentMetaInfoResponse()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  ExperimentMetaInfoResponse copyWith(void Function(ExperimentMetaInfoResponse) updates) => super.copyWith((message) => updates(message as ExperimentMetaInfoResponse)) as ExperimentMetaInfoResponse; // ignore: deprecated_member_use
  $pb.BuilderInfo get info_ => _i;
  @$core.pragma('dart2js:noInline')
  static ExperimentMetaInfoResponse create() => ExperimentMetaInfoResponse._();
  ExperimentMetaInfoResponse createEmptyInstance() => create();
  static $pb.PbList<ExperimentMetaInfoResponse> createRepeated() => $pb.PbList<ExperimentMetaInfoResponse>();
  @$core.pragma('dart2js:noInline')
  static ExperimentMetaInfoResponse getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<ExperimentMetaInfoResponse>(create);
  static ExperimentMetaInfoResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.List<ExperimentMetaInfoEntry> get entry => $_getList(0);
}

class NettestConfig extends $pb.GeneratedMessage {
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(const $core.bool.fromEnvironment('protobuf.omit_message_names') ? '' : 'NettestConfig', package: const $pb.PackageName(const $core.bool.fromEnvironment('protobuf.omit_message_names') ? '' : 'abi'), createEmptyInstance: create)
    ..m<$core.String, $core.String>(1, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'annotations', entryClassName: 'NettestConfig.AnnotationsEntry', keyFieldType: $pb.PbFieldType.OS, valueFieldType: $pb.PbFieldType.OS, packageName: const $pb.PackageName('abi'))
    ..aOS(2, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'extraOptions')
    ..pPS(3, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'inputs')
    ..pPS(4, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'inputFilePaths')
    ..aInt64(5, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'maxRuntime')
    ..aOS(6, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'name')
    ..aOB(7, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'noCollector')
    ..aOB(8, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'noJson')
    ..aOB(9, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'random')
    ..aOS(10, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'reportFile')
    ..aOM<SessionConfig>(11, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'session', subBuilder: SessionConfig.create)
    ..hasRequiredFields = false
  ;

  NettestConfig._() : super();
  factory NettestConfig({
    $core.Map<$core.String, $core.String>? annotations,
    $core.String? extraOptions,
    $core.Iterable<$core.String>? inputs,
    $core.Iterable<$core.String>? inputFilePaths,
    $fixnum.Int64? maxRuntime,
    $core.String? name,
    $core.bool? noCollector,
    $core.bool? noJson,
    $core.bool? random,
    $core.String? reportFile,
    SessionConfig? session,
  }) {
    final _result = create();
    if (annotations != null) {
      _result.annotations.addAll(annotations);
    }
    if (extraOptions != null) {
      _result.extraOptions = extraOptions;
    }
    if (inputs != null) {
      _result.inputs.addAll(inputs);
    }
    if (inputFilePaths != null) {
      _result.inputFilePaths.addAll(inputFilePaths);
    }
    if (maxRuntime != null) {
      _result.maxRuntime = maxRuntime;
    }
    if (name != null) {
      _result.name = name;
    }
    if (noCollector != null) {
      _result.noCollector = noCollector;
    }
    if (noJson != null) {
      _result.noJson = noJson;
    }
    if (random != null) {
      _result.random = random;
    }
    if (reportFile != null) {
      _result.reportFile = reportFile;
    }
    if (session != null) {
      _result.session = session;
    }
    return _result;
  }
  factory NettestConfig.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory NettestConfig.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  NettestConfig clone() => NettestConfig()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  NettestConfig copyWith(void Function(NettestConfig) updates) => super.copyWith((message) => updates(message as NettestConfig)) as NettestConfig; // ignore: deprecated_member_use
  $pb.BuilderInfo get info_ => _i;
  @$core.pragma('dart2js:noInline')
  static NettestConfig create() => NettestConfig._();
  NettestConfig createEmptyInstance() => create();
  static $pb.PbList<NettestConfig> createRepeated() => $pb.PbList<NettestConfig>();
  @$core.pragma('dart2js:noInline')
  static NettestConfig getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<NettestConfig>(create);
  static NettestConfig? _defaultInstance;

  @$pb.TagNumber(1)
  $core.Map<$core.String, $core.String> get annotations => $_getMap(0);

  @$pb.TagNumber(2)
  $core.String get extraOptions => $_getSZ(1);
  @$pb.TagNumber(2)
  set extraOptions($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasExtraOptions() => $_has(1);
  @$pb.TagNumber(2)
  void clearExtraOptions() => clearField(2);

  @$pb.TagNumber(3)
  $core.List<$core.String> get inputs => $_getList(2);

  @$pb.TagNumber(4)
  $core.List<$core.String> get inputFilePaths => $_getList(3);

  @$pb.TagNumber(5)
  $fixnum.Int64 get maxRuntime => $_getI64(4);
  @$pb.TagNumber(5)
  set maxRuntime($fixnum.Int64 v) { $_setInt64(4, v); }
  @$pb.TagNumber(5)
  $core.bool hasMaxRuntime() => $_has(4);
  @$pb.TagNumber(5)
  void clearMaxRuntime() => clearField(5);

  @$pb.TagNumber(6)
  $core.String get name => $_getSZ(5);
  @$pb.TagNumber(6)
  set name($core.String v) { $_setString(5, v); }
  @$pb.TagNumber(6)
  $core.bool hasName() => $_has(5);
  @$pb.TagNumber(6)
  void clearName() => clearField(6);

  @$pb.TagNumber(7)
  $core.bool get noCollector => $_getBF(6);
  @$pb.TagNumber(7)
  set noCollector($core.bool v) { $_setBool(6, v); }
  @$pb.TagNumber(7)
  $core.bool hasNoCollector() => $_has(6);
  @$pb.TagNumber(7)
  void clearNoCollector() => clearField(7);

  @$pb.TagNumber(8)
  $core.bool get noJson => $_getBF(7);
  @$pb.TagNumber(8)
  set noJson($core.bool v) { $_setBool(7, v); }
  @$pb.TagNumber(8)
  $core.bool hasNoJson() => $_has(7);
  @$pb.TagNumber(8)
  void clearNoJson() => clearField(8);

  @$pb.TagNumber(9)
  $core.bool get random => $_getBF(8);
  @$pb.TagNumber(9)
  set random($core.bool v) { $_setBool(8, v); }
  @$pb.TagNumber(9)
  $core.bool hasRandom() => $_has(8);
  @$pb.TagNumber(9)
  void clearRandom() => clearField(9);

  @$pb.TagNumber(10)
  $core.String get reportFile => $_getSZ(9);
  @$pb.TagNumber(10)
  set reportFile($core.String v) { $_setString(9, v); }
  @$pb.TagNumber(10)
  $core.bool hasReportFile() => $_has(9);
  @$pb.TagNumber(10)
  void clearReportFile() => clearField(10);

  @$pb.TagNumber(11)
  SessionConfig get session => $_getN(10);
  @$pb.TagNumber(11)
  set session(SessionConfig v) { setField(11, v); }
  @$pb.TagNumber(11)
  $core.bool hasSession() => $_has(10);
  @$pb.TagNumber(11)
  void clearSession() => clearField(11);
  @$pb.TagNumber(11)
  SessionConfig ensureSession() => $_ensure(10);
}

class ProgressEvent extends $pb.GeneratedMessage {
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(const $core.bool.fromEnvironment('protobuf.omit_message_names') ? '' : 'ProgressEvent', package: const $pb.PackageName(const $core.bool.fromEnvironment('protobuf.omit_message_names') ? '' : 'abi'), createEmptyInstance: create)
    ..a<$core.double>(1, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'percentage', $pb.PbFieldType.OD)
    ..aOS(2, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'message')
    ..hasRequiredFields = false
  ;

  ProgressEvent._() : super();
  factory ProgressEvent({
    $core.double? percentage,
    $core.String? message,
  }) {
    final _result = create();
    if (percentage != null) {
      _result.percentage = percentage;
    }
    if (message != null) {
      _result.message = message;
    }
    return _result;
  }
  factory ProgressEvent.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory ProgressEvent.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  ProgressEvent clone() => ProgressEvent()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  ProgressEvent copyWith(void Function(ProgressEvent) updates) => super.copyWith((message) => updates(message as ProgressEvent)) as ProgressEvent; // ignore: deprecated_member_use
  $pb.BuilderInfo get info_ => _i;
  @$core.pragma('dart2js:noInline')
  static ProgressEvent create() => ProgressEvent._();
  ProgressEvent createEmptyInstance() => create();
  static $pb.PbList<ProgressEvent> createRepeated() => $pb.PbList<ProgressEvent>();
  @$core.pragma('dart2js:noInline')
  static ProgressEvent getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<ProgressEvent>(create);
  static ProgressEvent? _defaultInstance;

  @$pb.TagNumber(1)
  $core.double get percentage => $_getN(0);
  @$pb.TagNumber(1)
  set percentage($core.double v) { $_setDouble(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasPercentage() => $_has(0);
  @$pb.TagNumber(1)
  void clearPercentage() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get message => $_getSZ(1);
  @$pb.TagNumber(2)
  set message($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasMessage() => $_has(1);
  @$pb.TagNumber(2)
  void clearMessage() => clearField(2);
}

class DataUsageEvent extends $pb.GeneratedMessage {
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(const $core.bool.fromEnvironment('protobuf.omit_message_names') ? '' : 'DataUsageEvent', package: const $pb.PackageName(const $core.bool.fromEnvironment('protobuf.omit_message_names') ? '' : 'abi'), createEmptyInstance: create)
    ..a<$core.double>(1, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'kibiBytesSent', $pb.PbFieldType.OD)
    ..a<$core.double>(2, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'kibiBytesReceived', $pb.PbFieldType.OD)
    ..hasRequiredFields = false
  ;

  DataUsageEvent._() : super();
  factory DataUsageEvent({
    $core.double? kibiBytesSent,
    $core.double? kibiBytesReceived,
  }) {
    final _result = create();
    if (kibiBytesSent != null) {
      _result.kibiBytesSent = kibiBytesSent;
    }
    if (kibiBytesReceived != null) {
      _result.kibiBytesReceived = kibiBytesReceived;
    }
    return _result;
  }
  factory DataUsageEvent.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory DataUsageEvent.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  DataUsageEvent clone() => DataUsageEvent()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  DataUsageEvent copyWith(void Function(DataUsageEvent) updates) => super.copyWith((message) => updates(message as DataUsageEvent)) as DataUsageEvent; // ignore: deprecated_member_use
  $pb.BuilderInfo get info_ => _i;
  @$core.pragma('dart2js:noInline')
  static DataUsageEvent create() => DataUsageEvent._();
  DataUsageEvent createEmptyInstance() => create();
  static $pb.PbList<DataUsageEvent> createRepeated() => $pb.PbList<DataUsageEvent>();
  @$core.pragma('dart2js:noInline')
  static DataUsageEvent getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<DataUsageEvent>(create);
  static DataUsageEvent? _defaultInstance;

  @$pb.TagNumber(1)
  $core.double get kibiBytesSent => $_getN(0);
  @$pb.TagNumber(1)
  set kibiBytesSent($core.double v) { $_setDouble(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasKibiBytesSent() => $_has(0);
  @$pb.TagNumber(1)
  void clearKibiBytesSent() => clearField(1);

  @$pb.TagNumber(2)
  $core.double get kibiBytesReceived => $_getN(1);
  @$pb.TagNumber(2)
  set kibiBytesReceived($core.double v) { $_setDouble(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasKibiBytesReceived() => $_has(1);
  @$pb.TagNumber(2)
  void clearKibiBytesReceived() => clearField(2);
}

class SubmitEvent extends $pb.GeneratedMessage {
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(const $core.bool.fromEnvironment('protobuf.omit_message_names') ? '' : 'SubmitEvent', package: const $pb.PackageName(const $core.bool.fromEnvironment('protobuf.omit_message_names') ? '' : 'abi'), createEmptyInstance: create)
    ..aOB(1, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'notSubmitted')
    ..aOS(2, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'failure')
    ..aInt64(3, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'index')
    ..aOS(4, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'input')
    ..aOS(5, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'reportId')
    ..aOS(6, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'measurement')
    ..hasRequiredFields = false
  ;

  SubmitEvent._() : super();
  factory SubmitEvent({
    $core.bool? notSubmitted,
    $core.String? failure,
    $fixnum.Int64? index,
    $core.String? input,
    $core.String? reportId,
    $core.String? measurement,
  }) {
    final _result = create();
    if (notSubmitted != null) {
      _result.notSubmitted = notSubmitted;
    }
    if (failure != null) {
      _result.failure = failure;
    }
    if (index != null) {
      _result.index = index;
    }
    if (input != null) {
      _result.input = input;
    }
    if (reportId != null) {
      _result.reportId = reportId;
    }
    if (measurement != null) {
      _result.measurement = measurement;
    }
    return _result;
  }
  factory SubmitEvent.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory SubmitEvent.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  SubmitEvent clone() => SubmitEvent()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  SubmitEvent copyWith(void Function(SubmitEvent) updates) => super.copyWith((message) => updates(message as SubmitEvent)) as SubmitEvent; // ignore: deprecated_member_use
  $pb.BuilderInfo get info_ => _i;
  @$core.pragma('dart2js:noInline')
  static SubmitEvent create() => SubmitEvent._();
  SubmitEvent createEmptyInstance() => create();
  static $pb.PbList<SubmitEvent> createRepeated() => $pb.PbList<SubmitEvent>();
  @$core.pragma('dart2js:noInline')
  static SubmitEvent getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<SubmitEvent>(create);
  static SubmitEvent? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get notSubmitted => $_getBF(0);
  @$pb.TagNumber(1)
  set notSubmitted($core.bool v) { $_setBool(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasNotSubmitted() => $_has(0);
  @$pb.TagNumber(1)
  void clearNotSubmitted() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get failure => $_getSZ(1);
  @$pb.TagNumber(2)
  set failure($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasFailure() => $_has(1);
  @$pb.TagNumber(2)
  void clearFailure() => clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get index => $_getI64(2);
  @$pb.TagNumber(3)
  set index($fixnum.Int64 v) { $_setInt64(2, v); }
  @$pb.TagNumber(3)
  $core.bool hasIndex() => $_has(2);
  @$pb.TagNumber(3)
  void clearIndex() => clearField(3);

  @$pb.TagNumber(4)
  $core.String get input => $_getSZ(3);
  @$pb.TagNumber(4)
  set input($core.String v) { $_setString(3, v); }
  @$pb.TagNumber(4)
  $core.bool hasInput() => $_has(3);
  @$pb.TagNumber(4)
  void clearInput() => clearField(4);

  @$pb.TagNumber(5)
  $core.String get reportId => $_getSZ(4);
  @$pb.TagNumber(5)
  set reportId($core.String v) { $_setString(4, v); }
  @$pb.TagNumber(5)
  $core.bool hasReportId() => $_has(4);
  @$pb.TagNumber(5)
  void clearReportId() => clearField(5);

  @$pb.TagNumber(6)
  $core.String get measurement => $_getSZ(5);
  @$pb.TagNumber(6)
  set measurement($core.String v) { $_setString(5, v); }
  @$pb.TagNumber(6)
  $core.bool hasMeasurement() => $_has(5);
  @$pb.TagNumber(6)
  void clearMeasurement() => clearField(6);
}

class OONIRunV2DescriptorNettest extends $pb.GeneratedMessage {
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(const $core.bool.fromEnvironment('protobuf.omit_message_names') ? '' : 'OONIRunV2DescriptorNettest', package: const $pb.PackageName(const $core.bool.fromEnvironment('protobuf.omit_message_names') ? '' : 'abi'), createEmptyInstance: create)
    ..m<$core.String, $core.String>(1, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'annotations', entryClassName: 'OONIRunV2DescriptorNettest.AnnotationsEntry', keyFieldType: $pb.PbFieldType.OS, valueFieldType: $pb.PbFieldType.OS, packageName: const $pb.PackageName('abi'))
    ..pPS(2, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'inputs')
    ..aOS(3, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'options')
    ..aOS(4, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'testName')
    ..hasRequiredFields = false
  ;

  OONIRunV2DescriptorNettest._() : super();
  factory OONIRunV2DescriptorNettest({
    $core.Map<$core.String, $core.String>? annotations,
    $core.Iterable<$core.String>? inputs,
    $core.String? options,
    $core.String? testName,
  }) {
    final _result = create();
    if (annotations != null) {
      _result.annotations.addAll(annotations);
    }
    if (inputs != null) {
      _result.inputs.addAll(inputs);
    }
    if (options != null) {
      _result.options = options;
    }
    if (testName != null) {
      _result.testName = testName;
    }
    return _result;
  }
  factory OONIRunV2DescriptorNettest.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory OONIRunV2DescriptorNettest.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  OONIRunV2DescriptorNettest clone() => OONIRunV2DescriptorNettest()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  OONIRunV2DescriptorNettest copyWith(void Function(OONIRunV2DescriptorNettest) updates) => super.copyWith((message) => updates(message as OONIRunV2DescriptorNettest)) as OONIRunV2DescriptorNettest; // ignore: deprecated_member_use
  $pb.BuilderInfo get info_ => _i;
  @$core.pragma('dart2js:noInline')
  static OONIRunV2DescriptorNettest create() => OONIRunV2DescriptorNettest._();
  OONIRunV2DescriptorNettest createEmptyInstance() => create();
  static $pb.PbList<OONIRunV2DescriptorNettest> createRepeated() => $pb.PbList<OONIRunV2DescriptorNettest>();
  @$core.pragma('dart2js:noInline')
  static OONIRunV2DescriptorNettest getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<OONIRunV2DescriptorNettest>(create);
  static OONIRunV2DescriptorNettest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.Map<$core.String, $core.String> get annotations => $_getMap(0);

  @$pb.TagNumber(2)
  $core.List<$core.String> get inputs => $_getList(1);

  @$pb.TagNumber(3)
  $core.String get options => $_getSZ(2);
  @$pb.TagNumber(3)
  set options($core.String v) { $_setString(2, v); }
  @$pb.TagNumber(3)
  $core.bool hasOptions() => $_has(2);
  @$pb.TagNumber(3)
  void clearOptions() => clearField(3);

  @$pb.TagNumber(4)
  $core.String get testName => $_getSZ(3);
  @$pb.TagNumber(4)
  set testName($core.String v) { $_setString(3, v); }
  @$pb.TagNumber(4)
  $core.bool hasTestName() => $_has(3);
  @$pb.TagNumber(4)
  void clearTestName() => clearField(4);
}

class OONIRunV2Descriptor extends $pb.GeneratedMessage {
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(const $core.bool.fromEnvironment('protobuf.omit_message_names') ? '' : 'OONIRunV2Descriptor', package: const $pb.PackageName(const $core.bool.fromEnvironment('protobuf.omit_message_names') ? '' : 'abi'), createEmptyInstance: create)
    ..aOS(1, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'name')
    ..aOS(2, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'description')
    ..aOS(3, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'author')
    ..pc<OONIRunV2DescriptorNettest>(4, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'nettests', $pb.PbFieldType.PM, subBuilder: OONIRunV2DescriptorNettest.create)
    ..hasRequiredFields = false
  ;

  OONIRunV2Descriptor._() : super();
  factory OONIRunV2Descriptor({
    $core.String? name,
    $core.String? description,
    $core.String? author,
    $core.Iterable<OONIRunV2DescriptorNettest>? nettests,
  }) {
    final _result = create();
    if (name != null) {
      _result.name = name;
    }
    if (description != null) {
      _result.description = description;
    }
    if (author != null) {
      _result.author = author;
    }
    if (nettests != null) {
      _result.nettests.addAll(nettests);
    }
    return _result;
  }
  factory OONIRunV2Descriptor.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory OONIRunV2Descriptor.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  OONIRunV2Descriptor clone() => OONIRunV2Descriptor()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  OONIRunV2Descriptor copyWith(void Function(OONIRunV2Descriptor) updates) => super.copyWith((message) => updates(message as OONIRunV2Descriptor)) as OONIRunV2Descriptor; // ignore: deprecated_member_use
  $pb.BuilderInfo get info_ => _i;
  @$core.pragma('dart2js:noInline')
  static OONIRunV2Descriptor create() => OONIRunV2Descriptor._();
  OONIRunV2Descriptor createEmptyInstance() => create();
  static $pb.PbList<OONIRunV2Descriptor> createRepeated() => $pb.PbList<OONIRunV2Descriptor>();
  @$core.pragma('dart2js:noInline')
  static OONIRunV2Descriptor getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<OONIRunV2Descriptor>(create);
  static OONIRunV2Descriptor? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get name => $_getSZ(0);
  @$pb.TagNumber(1)
  set name($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasName() => $_has(0);
  @$pb.TagNumber(1)
  void clearName() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get description => $_getSZ(1);
  @$pb.TagNumber(2)
  set description($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasDescription() => $_has(1);
  @$pb.TagNumber(2)
  void clearDescription() => clearField(2);

  @$pb.TagNumber(3)
  $core.String get author => $_getSZ(2);
  @$pb.TagNumber(3)
  set author($core.String v) { $_setString(2, v); }
  @$pb.TagNumber(3)
  $core.bool hasAuthor() => $_has(2);
  @$pb.TagNumber(3)
  void clearAuthor() => clearField(3);

  @$pb.TagNumber(4)
  $core.List<OONIRunV2DescriptorNettest> get nettests => $_getList(3);
}

class OONIRunV2MeasureDescriptorConfig extends $pb.GeneratedMessage {
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(const $core.bool.fromEnvironment('protobuf.omit_message_names') ? '' : 'OONIRunV2MeasureDescriptorConfig', package: const $pb.PackageName(const $core.bool.fromEnvironment('protobuf.omit_message_names') ? '' : 'abi'), createEmptyInstance: create)
    ..aInt64(1, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'maxRuntime')
    ..aOB(2, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'noCollector')
    ..aOB(3, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'noJson')
    ..aOB(4, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'random')
    ..aOS(5, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'reportFile')
    ..aOM<SessionConfig>(6, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'session', subBuilder: SessionConfig.create)
    ..aOM<OONIRunV2Descriptor>(7, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'v2Descriptor', subBuilder: OONIRunV2Descriptor.create)
    ..hasRequiredFields = false
  ;

  OONIRunV2MeasureDescriptorConfig._() : super();
  factory OONIRunV2MeasureDescriptorConfig({
    $fixnum.Int64? maxRuntime,
    $core.bool? noCollector,
    $core.bool? noJson,
    $core.bool? random,
    $core.String? reportFile,
    SessionConfig? session,
    OONIRunV2Descriptor? v2Descriptor,
  }) {
    final _result = create();
    if (maxRuntime != null) {
      _result.maxRuntime = maxRuntime;
    }
    if (noCollector != null) {
      _result.noCollector = noCollector;
    }
    if (noJson != null) {
      _result.noJson = noJson;
    }
    if (random != null) {
      _result.random = random;
    }
    if (reportFile != null) {
      _result.reportFile = reportFile;
    }
    if (session != null) {
      _result.session = session;
    }
    if (v2Descriptor != null) {
      _result.v2Descriptor = v2Descriptor;
    }
    return _result;
  }
  factory OONIRunV2MeasureDescriptorConfig.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory OONIRunV2MeasureDescriptorConfig.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  OONIRunV2MeasureDescriptorConfig clone() => OONIRunV2MeasureDescriptorConfig()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  OONIRunV2MeasureDescriptorConfig copyWith(void Function(OONIRunV2MeasureDescriptorConfig) updates) => super.copyWith((message) => updates(message as OONIRunV2MeasureDescriptorConfig)) as OONIRunV2MeasureDescriptorConfig; // ignore: deprecated_member_use
  $pb.BuilderInfo get info_ => _i;
  @$core.pragma('dart2js:noInline')
  static OONIRunV2MeasureDescriptorConfig create() => OONIRunV2MeasureDescriptorConfig._();
  OONIRunV2MeasureDescriptorConfig createEmptyInstance() => create();
  static $pb.PbList<OONIRunV2MeasureDescriptorConfig> createRepeated() => $pb.PbList<OONIRunV2MeasureDescriptorConfig>();
  @$core.pragma('dart2js:noInline')
  static OONIRunV2MeasureDescriptorConfig getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<OONIRunV2MeasureDescriptorConfig>(create);
  static OONIRunV2MeasureDescriptorConfig? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get maxRuntime => $_getI64(0);
  @$pb.TagNumber(1)
  set maxRuntime($fixnum.Int64 v) { $_setInt64(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasMaxRuntime() => $_has(0);
  @$pb.TagNumber(1)
  void clearMaxRuntime() => clearField(1);

  @$pb.TagNumber(2)
  $core.bool get noCollector => $_getBF(1);
  @$pb.TagNumber(2)
  set noCollector($core.bool v) { $_setBool(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasNoCollector() => $_has(1);
  @$pb.TagNumber(2)
  void clearNoCollector() => clearField(2);

  @$pb.TagNumber(3)
  $core.bool get noJson => $_getBF(2);
  @$pb.TagNumber(3)
  set noJson($core.bool v) { $_setBool(2, v); }
  @$pb.TagNumber(3)
  $core.bool hasNoJson() => $_has(2);
  @$pb.TagNumber(3)
  void clearNoJson() => clearField(3);

  @$pb.TagNumber(4)
  $core.bool get random => $_getBF(3);
  @$pb.TagNumber(4)
  set random($core.bool v) { $_setBool(3, v); }
  @$pb.TagNumber(4)
  $core.bool hasRandom() => $_has(3);
  @$pb.TagNumber(4)
  void clearRandom() => clearField(4);

  @$pb.TagNumber(5)
  $core.String get reportFile => $_getSZ(4);
  @$pb.TagNumber(5)
  set reportFile($core.String v) { $_setString(4, v); }
  @$pb.TagNumber(5)
  $core.bool hasReportFile() => $_has(4);
  @$pb.TagNumber(5)
  void clearReportFile() => clearField(5);

  @$pb.TagNumber(6)
  SessionConfig get session => $_getN(5);
  @$pb.TagNumber(6)
  set session(SessionConfig v) { setField(6, v); }
  @$pb.TagNumber(6)
  $core.bool hasSession() => $_has(5);
  @$pb.TagNumber(6)
  void clearSession() => clearField(6);
  @$pb.TagNumber(6)
  SessionConfig ensureSession() => $_ensure(5);

  @$pb.TagNumber(7)
  OONIRunV2Descriptor get v2Descriptor => $_getN(6);
  @$pb.TagNumber(7)
  set v2Descriptor(OONIRunV2Descriptor v) { setField(7, v); }
  @$pb.TagNumber(7)
  $core.bool hasV2Descriptor() => $_has(6);
  @$pb.TagNumber(7)
  void clearV2Descriptor() => clearField(7);
  @$pb.TagNumber(7)
  OONIRunV2Descriptor ensureV2Descriptor() => $_ensure(6);
}

