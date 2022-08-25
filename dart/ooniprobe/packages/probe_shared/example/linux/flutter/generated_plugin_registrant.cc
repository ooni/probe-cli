//
//  Generated file. Do not edit.
//

// clang-format off

#include "generated_plugin_registrant.h"

#include <probe_shared/probe_shared_plugin.h>

void fl_register_plugins(FlPluginRegistry* registry) {
  g_autoptr(FlPluginRegistrar) probe_shared_registrar =
      fl_plugin_registry_get_registrar_for_plugin(registry, "ProbeSharedPlugin");
  probe_shared_plugin_register_with_registrar(probe_shared_registrar);
}
