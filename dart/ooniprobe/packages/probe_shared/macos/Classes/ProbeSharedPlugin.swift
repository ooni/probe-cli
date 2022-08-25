import Cocoa
import FlutterMacOS

public class ProbeSharedPlugin: NSObject, FlutterPlugin {
  public static func register(with registrar: FlutterPluginRegistrar) {
    let channel = FlutterMethodChannel(name: "probe_shared", binaryMessenger: registrar.messenger)
    let instance = ProbeSharedPlugin()
    registrar.addMethodCallDelegate(instance, channel: channel)
  }

  public func handle(_ call: FlutterMethodCall, result: @escaping FlutterResult) {
    switch call.method {
    case "getPlatformVersion":
      result([
        "appName": (Bundle.main.infoDictionary?["CFBundleDisplayName"] as? String) ?? (Bundle.main.infoDictionary?["CFBundleName"] as? String),
        "packageName":Bundle.main.bundleIdentifier,
        "version":Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String,
        "buildNumber":Bundle.main.infoDictionary?["CFBundleVersion"] as? String,
      ])
    default:
      result(FlutterMethodNotImplemented)
    }
  }
}
