// Signal-Android's pre-account provisioning socket, rebuilt on the JVM from the app's own code paths
// and library versions, to reproduce its network behaviour outside Android:
//   - TLS provider: Conscrypt 2.6.2 inserted at position 2 (ApplicationContext.initializeSecurityProvider;
//     the app bundles org.conscrypt:conscrypt-android:2.6.2, here the openjdk-uber build of the same version)
//   - SSLContext "TLS", trust = Signal's RSA root only (whisper.store via BlacklistingTrustManager)
//   - TLS 1.3 + 1.2 only (Tls12SocketFactory), OkHttp 5.3.2 with ConnectionSpec.RESTRICTED_TLS,
//     30 s connect/read timeouts, no retries (SignalUrlExtensions.buildOkHttpClient)
//   - wss://chat.signal.org/v1/websocket/provisioning/ (ProvisioningSocket.connect), then the
//     server's PUT /v1/address is acked with 200 "OK" and a GET /v1/keepalive is sent
// Usage (see android.sh): java -cp "lib/*" SignalAndroidHello.java [--get URL] [--ua UA] [--pad N]
import java.io.*;
import java.net.*;
import java.security.*;
import java.security.cert.*;
import java.util.*;
import java.util.concurrent.*;
import javax.net.ssl.*;
import okhttp3.*;
import okio.ByteString;

public class SignalAndroidHello {
  static final long T0 = System.nanoTime();
  static String ms() { return String.format("%7d ms", (System.nanoTime() - T0) / 1_000_000); }

  // --- Tls12SocketFactory: enable only TLSv1.3 / TLSv1.2 ---
  static final class Tls12SocketFactory extends SSLSocketFactory {
    private final SSLSocketFactory d;
    Tls12SocketFactory(SSLSocketFactory d) { this.d = d; }
    private Socket fix(Socket s) {
      if (s instanceof SSLSocket ss) {
        List<String> en = new ArrayList<>();
        for (String p : new String[]{"TLSv1.3", "TLSv1.2"}) if (Arrays.asList(ss.getSupportedProtocols()).contains(p)) en.add(p);
        if (!en.isEmpty()) ss.setEnabledProtocols(en.toArray(new String[0]));
      }
      return s;
    }
    public String[] getDefaultCipherSuites() { return d.getDefaultCipherSuites(); }
    public String[] getSupportedCipherSuites() { return d.getSupportedCipherSuites(); }
    public Socket createSocket(Socket s, String h, int p, boolean a) throws IOException { return fix(d.createSocket(s, h, p, a)); }
    public Socket createSocket(String h, int p) throws IOException { return fix(d.createSocket(h, p)); }
    public Socket createSocket(String h, int p, InetAddress l, int lp) throws IOException { return fix(d.createSocket(h, p, l, lp)); }
    public Socket createSocket(InetAddress h, int p) throws IOException { return fix(d.createSocket(h, p)); }
    public Socket createSocket(InetAddress h, int p, InetAddress l, int lp) throws IOException { return fix(d.createSocket(h, p, l, lp)); }
  }

  // --- minimal proto2 encode/decode for WebSocketMessage ---
  static byte[] varint(long v) { ByteArrayOutputStream o = new ByteArrayOutputStream(); while (true) { int b = (int) (v & 0x7F); v >>>= 7; if (v == 0) { o.write(b); break; } o.write(b | 0x80); } return o.toByteArray(); }
  static byte[] field(int num, byte[] payload) { return cat(varint(num << 3 | 2), varint(payload.length), payload); }
  static byte[] fieldV(int num, long v) { return cat(varint(num << 3), varint(v)); }
  static byte[] cat(byte[]... xs) { ByteArrayOutputStream o = new ByteArrayOutputStream(); for (byte[] x : xs) o.writeBytes(x); return o.toByteArray(); }
  static Map<Integer, List<Object>> decode(byte[] b) {
    Map<Integer, List<Object>> m = new HashMap<>(); int[] i = {0};
    while (i[0] < b.length) {
      long key = readVarint(b, i); int num = (int) (key >>> 3), wire = (int) (key & 7);
      Object v;
      if (wire == 0) v = readVarint(b, i);
      else if (wire == 2) { int ln = (int) readVarint(b, i); v = Arrays.copyOfRange(b, i[0], i[0] + ln); i[0] += ln; }
      else throw new IllegalStateException("wire " + wire);
      m.computeIfAbsent(num, k -> new ArrayList<>()).add(v);
    }
    return m;
  }
  static long readVarint(byte[] b, int[] i) { long n = 0; int s = 0; while (true) { int x = b[i[0]++] & 0xFF; n |= (long) (x & 0x7F) << s; s += 7; if ((x & 0x80) == 0) return n; } }

  public static void main(String[] args) throws Exception {
    String ua = "Signal-Android/8.29.3 Android/34"; int pad = 0;
    for (int k = 0; k < args.length; k++) { if (args[k].equals("--ua")) ua = args[++k]; if (args[k].equals("--pad")) pad = Integer.parseInt(args[++k]); }

    int pos = Security.insertProviderAt(org.conscrypt.Conscrypt.newProvider(), 2);
    X509Certificate root = (X509Certificate) CertificateFactory.getInstance("X.509").generateCertificate(new FileInputStream(System.getenv().getOrDefault("SIGNAL_CA", "signal.cer")));
    KeyStore ks = KeyStore.getInstance(KeyStore.getDefaultType()); ks.load(null, null); ks.setCertificateEntry("signal", root);
    TrustManagerFactory tmf = TrustManagerFactory.getInstance(TrustManagerFactory.getDefaultAlgorithm()); tmf.init(ks);
    SSLContext ctx = SSLContext.getInstance("TLS"); ctx.init(null, tmf.getTrustManagers(), null);
    System.out.println(ms() + " Conscrypt at position " + pos + "; SSLContext provider: " + ctx.getProvider().getName() + " " + ctx.getProvider().getVersionStr());

    final String agent = ua;
    OkHttpClient client = new OkHttpClient.Builder()
        .sslSocketFactory(new Tls12SocketFactory(ctx.getSocketFactory()), (X509TrustManager) tmf.getTrustManagers()[0])
        .connectionSpecs(List.of(ConnectionSpec.RESTRICTED_TLS))
        .retryOnConnectionFailure(false).readTimeout(30, TimeUnit.SECONDS).connectTimeout(30, TimeUnit.SECONDS)
        .addInterceptor(c -> c.proceed(c.request().newBuilder().header("User-Agent", agent).build()))  // StandardUserAgentInterceptor
        .build();

    for (int k = 0; k < args.length; k++) if (args[k].equals("--get")) {  // plain HTTPS GET (REST/CDN clients use the same builder)
      try (Response r = client.newCall(new Request.Builder().url(args[k + 1]).build()).execute()) {
        System.out.println(ms() + " GET " + args[k + 1] + " -> " + r.code() + " " + r.protocol() + " " + r.handshake().tlsVersion());
      } catch (IOException e) { System.out.println(ms() + " GET " + args[k + 1] + " failed: " + e); }
      client.connectionPool().evictAll(); return;
    }
    CountDownLatch done = new CountDownLatch(1); final int padding = pad; final long[] kid = {0};
    client.newWebSocket(new Request.Builder().url("wss://chat.signal.org/v1/websocket/provisioning/").build(), new WebSocketListener() {
      @Override public void onOpen(WebSocket ws, Response r) {
        System.out.println(ms() + " open: " + r.code() + " " + r.protocol() + " x-signal-timestamp=" + (r.header("x-signal-timestamp") != null)
            + " tls=" + r.handshake().tlsVersion() + " " + r.handshake().cipherSuite());
      }
      @Override public void onMessage(WebSocket ws, ByteString bytes) {
        Map<Integer, List<Object>> m = decode(bytes.toByteArray());
        if (m.containsKey(2)) {  // REQUEST from the server
          Map<Integer, List<Object>> req = decode((byte[]) m.get(2).get(0));
          String verb = new String((byte[]) req.get(1).get(0)), path = new String((byte[]) req.get(2).get(0));
          long id = (Long) req.get(4).get(0);
          int alen = req.containsKey(3) ? new String((byte[]) decode((byte[]) req.get(3).get(0)).get(1).get(0)).length() : 0;
          System.out.println(ms() + " server pushed " + verb + " " + path + " (address " + alen + " chars)");
          byte[] resp = cat(fieldV(1, id), fieldV(2, 200), field(3, "OK".getBytes()));
          ws.send(ByteString.of(cat(fieldV(1, 2), field(3, resp))));
          kid[0] = System.currentTimeMillis();
          byte[] rq = cat(field(1, "GET".getBytes()), field(2, "/v1/keepalive".getBytes()), fieldV(4, kid[0]));
          if (padding > 0) rq = cat(rq, field(3, new byte[padding]));
          ws.send(ByteString.of(cat(fieldV(1, 1), field(2, rq))));
        } else if (m.containsKey(3)) {  // RESPONSE to our keepalive
          Map<Integer, List<Object>> r = decode((byte[]) m.get(3).get(0));
          List<String> hs = new ArrayList<>(); for (Object h : r.getOrDefault(5, List.of())) hs.add(new String((byte[]) h).split(":")[0]);
          System.out.println(ms() + " keepalive response: status=" + r.get(2).get(0) + " id_match=" + (((Long) r.get(1).get(0)) == kid[0]) + " headers=" + hs);
          ws.close(1000, null); done.countDown();
        }
      }
      @Override public void onFailure(WebSocket ws, Throwable t, Response r) { System.out.println(ms() + " failure: " + t); done.countDown(); }
    });
    done.await(40, TimeUnit.SECONDS);
    client.dispatcher().executorService().shutdown(); client.connectionPool().evictAll();
    System.out.println(ms() + " done");
  }
}
