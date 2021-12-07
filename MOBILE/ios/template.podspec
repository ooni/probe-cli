Pod::Spec.new do |s|
  s.name = "oonimkall"
  s.version = "@VERSION@"
  s.summary = "OONI Probe Library for iOS"
  s.author = "Simone Basso"
  s.homepage = "https://github.com/ooni/probe-cli"
  s.license = { :type => "BSD" }
  s.source = {
    :http => "https://github.com/ooni/probe-cli/releases/download/@RELEASE@/oonimkall.xcframework.zip"
  }
  s.platform = :ios, "9.0"
  s.ios.vendored_frameworks = "oonimkall.framework"
end
