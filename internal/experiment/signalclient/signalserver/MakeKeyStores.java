// Assemble the PKCS#12 keystores Signal-Server's connectors load, from the PEM files certs.py writes:
// chat.p12 holds the Ed25519 and the RSA entries for chat.signal.org (Jetty picks by the client's
// signature algorithms, as production does); grpc.p12 holds the Ed25519 entry for grpc.chat.signal.org.
// Only the leaf is stored: production sends no intermediates. Done in Java because the password must be
// Signal-Server's test secret ("unset"), shorter than keytool's minimum, and openssl exports one key per file.
// Usage: java MakeKeyStores.java DIR PASSWORD
import java.io.FileOutputStream;
import java.nio.file.Files;
import java.nio.file.Path;
import java.security.KeyFactory;
import java.security.KeyStore;
import java.security.PrivateKey;
import java.security.cert.Certificate;
import java.security.cert.CertificateFactory;
import java.security.spec.PKCS8EncodedKeySpec;
import java.util.Base64;

public class MakeKeyStores {
  static PrivateKey key(Path p, String alg) throws Exception {
    String pem = Files.readString(p).replaceAll("-----[A-Z ]+-----", "").replaceAll("\\s", "");
    return KeyFactory.getInstance(alg).generatePrivate(new PKCS8EncodedKeySpec(Base64.getDecoder().decode(pem)));
  }

  static Certificate cert(Path p) throws Exception {
    try (var in = Files.newInputStream(p)) {
      return CertificateFactory.getInstance("X.509").generateCertificate(in);
    }
  }

  public static void main(String[] args) throws Exception {
    Path dir = Path.of(args[0]);
    char[] pw = args[1].toCharArray();
    KeyStore chat = KeyStore.getInstance("PKCS12");
    chat.load(null, pw);
    chat.setKeyEntry("ed25519", key(dir.resolve("chat-ed25519.key"), "Ed25519"), pw, new Certificate[] {cert(dir.resolve("chat-ed25519.pem"))});
    chat.setKeyEntry("rsa", key(dir.resolve("chat-rsa.key"), "RSA"), pw, new Certificate[] {cert(dir.resolve("chat-rsa.pem"))});
    try (var out = new FileOutputStream(dir.resolve("chat.p12").toFile())) { chat.store(out, pw); }
    KeyStore grpc = KeyStore.getInstance("PKCS12");
    grpc.load(null, pw);
    grpc.setKeyEntry("ed25519", key(dir.resolve("grpc-ed25519.key"), "Ed25519"), pw, new Certificate[] {cert(dir.resolve("grpc-ed25519.pem"))});
    try (var out = new FileOutputStream(dir.resolve("grpc.p12").toFile())) { grpc.store(out, pw); }
    System.out.println("wrote chat.p12 (ed25519, rsa) and grpc.p12 (ed25519)");
  }
}
