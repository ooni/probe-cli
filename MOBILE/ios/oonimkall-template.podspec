Pod::Spec.new do |s|
  s.name = "oonimkall"
  s.version = "@VERSION@"
  s.summary = "OONI Probe Library for iOS"
  s.author = "Mehul Gulati"
  s.homepage = "https://github.com/ooni/probe-cli"
  s.license = { :type => "GPL" }
  s.source = {
    :http => "https://repo1.maven.org/maven2/org/ooni/oonimkall-ios/@VERSION@/oonimkall-ios-@VERSION@.zip"
  }
  s.platform = :ios, "9.0"
  s.ios.vendored_frameworks = "oonimkall.xcframework"
end
