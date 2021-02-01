Pod::Spec.new do |s|
  s.name = "oonimkall"
  s.version = "@VERSION@"
  s.summary = "OONI Probe Engine for iOS"
  s.author = "Simone Basso"
  s.homepage = "https://github.com/ooni/probe-engine"
  s.license = { :type => "BSD" }
  s.source = {
    :http => "https://dl.bintray.com/ooni/ios/oonimkall-@VERSION@.framework.zip"
  }
  s.platform = :ios, "9.0"
  s.ios.vendored_frameworks = "oonimkall.framework"
end
