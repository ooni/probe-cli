#!/usr/bin/env python3
"""TLS server on :443 that completes handshakes only for the SNIs it serves, and answers any
other SNI (or none) with a handshake_failure alert, as CDN edges do (e.g. CloudFront)."""
import os, socket, ssl, subprocess, tempfile, threading

SERVED = {"target.test", "control.test"}

d = tempfile.mkdtemp()
cert, key = os.path.join(d, "cert.pem"), os.path.join(d, "key.pem")
subprocess.run(["openssl", "req", "-x509", "-newkey", "ec", "-pkeyopt", "ec_paramgen_curve:P-256",
                "-nodes", "-days", "1", "-subj", "/CN=target.test", "-keyout", key, "-out", cert,
                "-addext", "subjectAltName=DNS:target.test,DNS:control.test,IP:10.0.4.2"],
               check=True, capture_output=True)
ctx = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
ctx.load_cert_chain(cert, key)
ctx.set_alpn_protocols(["h2", "http/1.1"])


def sni_callback(sslobj, name, _ctx):
    if name not in SERVED:
        return ssl.ALERT_DESCRIPTION_HANDSHAKE_FAILURE
    return None


ctx.sni_callback = sni_callback


def serve(conn):
    try:
        with ctx.wrap_socket(conn, server_side=True) as tls:
            tls.recv(1)
    except (OSError, ssl.SSLError):
        pass


srv = socket.create_server(("0.0.0.0", 443), reuse_port=True)
while True:
    conn, _ = srv.accept()
    conn.settimeout(30)
    threading.Thread(target=serve, args=(conn,), daemon=True).start()
