Pod::Spec.new do |s|
  s.name = "libcrypto"
  s.version = "@VERSION@"
  s.summary = "OpenSSL libcrypto compiled for OONI Probe iOS"
  s.author = "Mehul Gulati"
  s.homepage = "https://github.com/ooni/probe-cli"
  s.license = { :type => "Apache" }
  s.source = {
    :http => "https://repo1.maven.org/maven2/org/ooni/libcrypto-ios/@VERSION@/libcrypto-ios-@VERSION@.zip"
  }
  s.platform = :ios, "9.0"
  s.ios.vendored_frameworks = "libcrypto.xcframework"
end
