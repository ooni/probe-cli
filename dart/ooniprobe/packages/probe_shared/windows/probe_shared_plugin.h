#ifndef FLUTTER_PLUGIN_PROBE_SHARED_PLUGIN_H_
#define FLUTTER_PLUGIN_PROBE_SHARED_PLUGIN_H_

#include <flutter/method_channel.h>
#include <flutter/plugin_registrar_windows.h>

#include <memory>

namespace probe_shared {

class ProbeSharedPlugin : public flutter::Plugin {
 public:
  static void RegisterWithRegistrar(flutter::PluginRegistrarWindows *registrar);

  ProbeSharedPlugin();

  virtual ~ProbeSharedPlugin();

  // Disallow copy and assign.
  ProbeSharedPlugin(const ProbeSharedPlugin&) = delete;
  ProbeSharedPlugin& operator=(const ProbeSharedPlugin&) = delete;

 private:
  // Called when a method is called on this plugin's channel from Dart.
  void HandleMethodCall(
      const flutter::MethodCall<flutter::EncodableValue> &method_call,
      std::unique_ptr<flutter::MethodResult<flutter::EncodableValue>> result);
};

}  // namespace probe_shared

#endif  // FLUTTER_PLUGIN_PROBE_SHARED_PLUGIN_H_
