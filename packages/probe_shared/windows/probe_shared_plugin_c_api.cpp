#include "include/probe_shared/probe_shared_plugin_c_api.h"

#include <flutter/plugin_registrar_windows.h>

#include "probe_shared_plugin.h"

void ProbeSharedPluginCApiRegisterWithRegistrar(
    FlutterDesktopPluginRegistrarRef registrar) {
  probe_shared::ProbeSharedPlugin::RegisterWithRegistrar(
      flutter::PluginRegistrarManager::GetInstance()
          ->GetRegistrar<flutter::PluginRegistrarWindows>(registrar));
}
