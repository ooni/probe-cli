Pod::Spec.new do |s|
  s.name = "libz"
  s.version = "@VERSION@"
  s.summary = "zlib compiled for OONI Probe iOS"
  s.author = "Simone Basso"
  s.homepage = "https://github.com/ooni/probe-cli"
  s.license = { :type => "zlib" }
  s.source = {
    :http => "https://github.com/ooni/probe-cli/releases/download/@RELEASE@/libz.xcframework.zip"
  }
  s.platform = :ios, "9.0"
  s.ios.vendored_frameworks = "libz.xcframework"
end
