Pod::Spec.new do |s|
  s.name = "libcrypto"
  s.version = "@VERSION@"
  s.summary = "OpenSSL libcrypto compiled for OONI Probe iOS"
  s.author = "Simone Basso"
  s.homepage = "https://github.com/ooni/probe-cli"
  s.license = { :type => "Apache" }
  s.source = {
    :http => "https://github.com/ooni/probe-cli/releases/download/@RELEASE@/libcrypto.xcframework.zip"
  }
  s.platform = :ios, "9.0"
  s.ios.vendored_frameworks = "libcrypto.xcframework"
end
