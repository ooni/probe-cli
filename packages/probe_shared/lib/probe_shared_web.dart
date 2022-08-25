// In order to *not* need this ignore, consider extracting the "web" version
// of your plugin as a separate package, instead of inlining it in the same
// package as the core of your plugin.
// ignore: avoid_web_libraries_in_flutter
import 'dart:html' as html show window;

import 'package:flutter_web_plugins/flutter_web_plugins.dart';
import 'package:probe_shared/data/package_info.dart';

import 'probe_shared_platform_interface.dart';

/// A web implementation of the ProbeSharedPlatform of the ProbeShared plugin.
class ProbeSharedWeb extends ProbeSharedPlatform {
  /// Constructs a ProbeSharedWeb
  ProbeSharedWeb();

  static void registerWith(Registrar registrar) {
    ProbeSharedPlatform.instance = ProbeSharedWeb();
  }

  /// Returns a [String] containing the version of the platform.
  @override
  Future<PackageInfoData?> getPlatformVersion() async {
    final version = html.window.navigator.userAgent;
    return PackageInfoData(
      version: version,
      appName: '',
      buildNumber: '',
      buildSignature: '',
      packageName: '',
    );
  }
}
