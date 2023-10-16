Pod::Spec.new do |s|
  s.name = "libtor"
  s.version = "@VERSION@"
  s.summary = "tor compiled for OONI Probe iOS"
  s.author = "Simone Basso"
  s.homepage = "https://github.com/ooni/probe-cli"
  s.license = { :type => "BSD" }
  s.source = {
    :http => "https://github.com/ooni/probe-cli/releases/download/@RELEASE@/libtor.xcframework.zip"
  }
  s.platform = :ios, "9.0"
  s.ios.vendored_frameworks = "libtor.xcframework"
end
