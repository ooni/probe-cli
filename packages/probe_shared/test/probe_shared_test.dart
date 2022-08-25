import 'package:flutter_test/flutter_test.dart';
import 'package:plugin_platform_interface/plugin_platform_interface.dart';
import 'package:probe_shared/data/package_info.dart';
import 'package:probe_shared/probe_shared.dart';
import 'package:probe_shared/probe_shared_method_channel.dart';
import 'package:probe_shared/probe_shared_platform_interface.dart';

class MockProbeSharedPlatform
    with MockPlatformInterfaceMixin
    implements ProbeSharedPlatform {
  @override
  Future<PackageInfoData?> getPlatformVersion() => Future.value(
        PackageInfoData(
          version: '42',
          packageName: '',
          buildSignature: '',
          buildNumber: '',
          appName: '',
        ),
      );
}

void main() {
  final ProbeSharedPlatform initialPlatform = ProbeSharedPlatform.instance;

  test('$MethodChannelProbeShared is the default instance', () {
    expect(initialPlatform, isInstanceOf<MethodChannelProbeShared>());
  });

  test('getPlatformVersion', () async {
    ProbeShared probeSharedPlugin = ProbeShared();
    MockProbeSharedPlatform fakePlatform = MockProbeSharedPlatform();
    ProbeSharedPlatform.instance = fakePlatform;

    expect((await probeSharedPlugin.getPlatformVersion())?.version, '42');
  });
}
