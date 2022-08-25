import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:probe_shared/probe_shared_method_channel.dart';

void main() {
  MethodChannelProbeShared platform = MethodChannelProbeShared();
  const MethodChannel channel = MethodChannel('probe_shared');

  TestWidgetsFlutterBinding.ensureInitialized();

  setUp(() {
    channel.setMockMethodCallHandler((MethodCall methodCall) async {
      return {
        'version': '42',
        'packageName': '',
        'buildSignature': '',
        'buildNumber': '',
        'appName': '',
      };
    });
  });

  tearDown(() {
    channel.setMockMethodCallHandler(null);
  });

  test('getPlatformVersion', () async {
    expect((await platform.getPlatformVersion())?.version, '42');
  });
}
