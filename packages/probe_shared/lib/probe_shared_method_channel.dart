import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:probe_shared/data/package_info.dart';

import 'probe_shared_platform_interface.dart';

/// An implementation of [ProbeSharedPlatform] that uses method channels.
class MethodChannelProbeShared extends ProbeSharedPlatform {
  /// The method channel used to interact with the native platform.
  @visibleForTesting
  final methodChannel = const MethodChannel('probe_shared');

  @override
  Future<PackageInfoData?> getPlatformVersion() async {
    final map = await methodChannel.invokeMapMethod<String,dynamic>('getPlatformVersion');
    return PackageInfoData(
      appName: map!['appName'] ?? '',
      packageName: map['packageName'] ?? '',
      version: map['version'] ?? '',
      buildNumber: map['buildNumber'] ?? '',
      buildSignature: map['buildSignature'] ?? '',
    );
  }
}
