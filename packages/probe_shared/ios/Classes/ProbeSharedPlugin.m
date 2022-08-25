#import "ProbeSharedPlugin.h"
#if __has_include(<probe_shared/probe_shared-Swift.h>)
#import <probe_shared/probe_shared-Swift.h>
#else
// Support project import fallback if the generated compatibility header
// is not copied when this plugin is created as a library.
// https://forums.swift.org/t/swift-static-libraries-dont-copy-generated-objective-c-header/19816
#import "probe_shared-Swift.h"
#endif

@implementation ProbeSharedPlugin
+ (void)registerWithRegistrar:(NSObject<FlutterPluginRegistrar>*)registrar {
  FlutterMethodChannel *channel = [FlutterMethodChannel
        methodChannelWithName:@"probe_shared"
              binaryMessenger:[registrar messenger]];
    ProbeSharedPlugin *instance = [[ProbeSharedPlugin alloc] init];
    [registrar addMethodCallDelegate:instance channel:channel];
  }

  - (void)handleMethodCall:(FlutterMethodCall *)call
                    result:(FlutterResult)result {
    if ([call.method isEqualToString:@"getPlatformVersion"]) {
      result(@{
        @"appName" : [[NSBundle mainBundle]
            objectForInfoDictionaryKey:@"CFBundleDisplayName"]
            ?: [[NSBundle mainBundle] objectForInfoDictionaryKey:@"CFBundleName"]
                   ?: [NSNull null],
        @"packageName" : [[NSBundle mainBundle] bundleIdentifier]
            ?: [NSNull null],
        @"version" : [[NSBundle mainBundle]
            objectForInfoDictionaryKey:@"CFBundleShortVersionString"]
            ?: [NSNull null],
        @"buildNumber" : [[NSBundle mainBundle]
            objectForInfoDictionaryKey:@"CFBundleVersion"]
            ?: [NSNull null],
      });
    } else {
      result(FlutterMethodNotImplemented);
    }
  }
@end
