Pod::Spec.new do |s|
  s.name = "libtor"
  s.version = "@VERSION@"
  s.summary = "tor compiled for OONI Probe iOS"
  s.author = "Mehul Gulati"
  s.homepage = "https://github.com/ooni/probe-cli"
  s.license = { :type => "BSD" }
  s.source = {
    :http => "https://repo1.maven.org/maven2/org/ooni/libtor-ios/@VERSION@/libtor-ios-@VERSION@.zip"
  }
  s.platform = :ios, "9.0"
  s.ios.vendored_frameworks = "libtor.xcframework"
end
