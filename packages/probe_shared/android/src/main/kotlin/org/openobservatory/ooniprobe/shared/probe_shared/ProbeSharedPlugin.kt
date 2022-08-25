package org.openobservatory.ooniprobe.shared.probe_shared

import android.content.Context
import androidx.annotation.NonNull
import androidx.core.content.pm.PackageInfoCompat.getLongVersionCode

import io.flutter.embedding.engine.plugins.FlutterPlugin
import io.flutter.plugin.common.MethodCall
import io.flutter.plugin.common.MethodChannel
import io.flutter.plugin.common.MethodChannel.MethodCallHandler
import io.flutter.plugin.common.MethodChannel.Result

/** ProbeSharedPlugin */
class ProbeSharedPlugin: FlutterPlugin, MethodCallHandler {
  /// The MethodChannel that will the communication between Flutter and native Android
  ///
  /// This local reference serves to register the plugin with the Flutter Engine and unregister it
  /// when the Flutter Engine is detached from the Activity
  private lateinit var channel : MethodChannel
  private lateinit var applicationContext: Context

  override fun onAttachedToEngine(@NonNull flutterPluginBinding: FlutterPlugin.FlutterPluginBinding) {
    applicationContext = flutterPluginBinding.applicationContext
    channel = MethodChannel(flutterPluginBinding.binaryMessenger, "probe_shared")
    channel.setMethodCallHandler(this)
  }

  override fun onMethodCall(@NonNull call: MethodCall, @NonNull result: Result) {
    if (call.method == "getPlatformVersion") {
      val packageManager = applicationContext.packageManager
      val info = packageManager.getPackageInfo(applicationContext.packageName, 0)


      val infoMap = HashMap<String, String>()
      infoMap.apply {
        put("appName", info.applicationInfo.loadLabel(packageManager).toString())
        put("packageName", applicationContext.packageName)
        put("version", info.versionName)
        put("buildNumber", getLongVersionCode(info).toString())
      }.also { resultingMap ->
        result.success(resultingMap)
      }
    } else {
      result.notImplemented()
    }
  }

  override fun onDetachedFromEngine(@NonNull binding: FlutterPlugin.FlutterPluginBinding) {
    channel.setMethodCallHandler(null)
  }
}
