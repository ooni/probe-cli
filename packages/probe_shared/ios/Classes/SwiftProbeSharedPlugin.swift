import Flutter
import UIKit

public class SwiftProbeSharedPlugin: NSObject, FlutterPlugin {
    public static func register(with registrar: FlutterPluginRegistrar) {
        let channel = FlutterMethodChannel(name: "probe_shared", binaryMessenger: registrar.messenger())
        let instance = SwiftProbeSharedPlugin()
        registrar.addMethodCallDelegate(instance, channel: channel)
    }

    public func handle(_ call: FlutterMethodCall, result: @escaping FlutterResult) {
        switch call.method {
        case "getPlatformVersion":
        break
        default:
            result(FlutterMethodNotImplemented)
        }
    }
}
