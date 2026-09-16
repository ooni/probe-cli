Pod::Spec.new do |s|
  s.name = "libz"
  s.version = "@VERSION@"
  s.summary = "zlib compiled for OONI Probe iOS"
  s.author = "Mehul Gulati"
  s.homepage = "https://github.com/ooni/probe-cli"
  s.license = { :type => "zlib" }
  s.source = {
    :http => "https://repo1.maven.org/maven2/org/ooni/libz-ios/@VERSION@/libz-ios-@VERSION@.zip"
  }
  s.platform = :ios, "9.0"
  s.ios.vendored_frameworks = "libz.xcframework"
end
