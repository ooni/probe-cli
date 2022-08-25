
import 'package:probe_shared/data/package_info.dart';

import 'probe_shared_platform_interface.dart';

class ProbeShared {
  Future<PackageInfoData?> getPlatformVersion() {
    return ProbeSharedPlatform.instance.getPlatformVersion();
  }
}
