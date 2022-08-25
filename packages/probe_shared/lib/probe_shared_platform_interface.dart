import 'package:plugin_platform_interface/plugin_platform_interface.dart';
import 'package:probe_shared/data/package_info.dart';

import 'probe_shared_method_channel.dart';

abstract class ProbeSharedPlatform extends PlatformInterface {
  /// Constructs a ProbeSharedPlatform.
  ProbeSharedPlatform() : super(token: _token);

  static final Object _token = Object();

  static ProbeSharedPlatform _instance = MethodChannelProbeShared();

  /// The default instance of [ProbeSharedPlatform] to use.
  ///
  /// Defaults to [MethodChannelProbeShared].
  static ProbeSharedPlatform get instance => _instance;
  
  /// Platform-specific implementations should set this with their own
  /// platform-specific class that extends [ProbeSharedPlatform] when
  /// they register themselves.
  static set instance(ProbeSharedPlatform instance) {
    PlatformInterface.verifyToken(instance, _token);
    _instance = instance;
  }

  Future<PackageInfoData?> getPlatformVersion() {
    throw UnimplementedError('platformVersion() has not been implemented.');
  }
}
