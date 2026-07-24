Pod::Spec.new do |s|
  s.name = "libssl"
  s.version = "@VERSION@"
  s.summary = "OpenSSL libssl compiled for OONI Probe iOS"
  s.author = "Mehul Gulati"
  s.homepage = "https://github.com/ooni/probe-cli"
  s.license = { :type => "Apache" }
  s.source = {
    :http => "https://repo1.maven.org/maven2/org/ooni/libssl-ios/@VERSION@/libssl-ios-@VERSION@.zip"
  }
  s.platform = :ios, "9.0"
  s.ios.vendored_frameworks = "libssl.xcframework"
end
