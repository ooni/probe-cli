#!/usr/bin/env python3
"""Measure a TLS front end's observable profile, to make the local Signal-Server match production.

For each target (host[@address:port]) it records: TLS versions accepted; each TLS 1.3 and TLS 1.2
cipher suite accepted, and which one the server picks from a full list (server preference);
key-exchange groups accepted in TLS 1.3 (and whether the server asks for another key share);
certificate chain (key type and size, signature algorithm, subjects, issuers); ALPN choices;
session tickets; OCSP stapling; and the server's HTTP/2 SETTINGS and initial WINDOW_UPDATE.

Uses `openssl s_client` (OpenSSL >= 3.5 for ML-KEM groups) and Python's `h2`. About 100 ordinary
handshakes per target, paced. Run inside a container that has both, e.g. via tlsprofile.sh.
Usage: tlsprofile.py TARGET [TARGET...] > profile.json
  TARGET: chat.signal.org  or  chat.signal.org@127.0.0.1:8443 (SNI chat.signal.org, address given)
"""
import json, socket, ssl, subprocess, sys, time

TLS13 = ["TLS_AES_128_GCM_SHA256", "TLS_AES_256_GCM_SHA384", "TLS_CHACHA20_POLY1305_SHA256"]
GROUPS = ["X25519MLKEM768", "x25519", "P-256", "P-384", "P-521", "x448", "ffdhe2048", "ffdhe3072"]
PAUSE = 0.2


def target(t):
    sni, _, addr = t.partition("@")
    return sni, addr or f"{sni}:443"


def s_client(sni, addr, *args, stdin=b""):
    time.sleep(PAUSE)
    r = subprocess.run(["openssl", "s_client", "-connect", addr, "-servername", sni, "-no_ign_eof", *args],
                       input=stdin, capture_output=True, timeout=20)
    return r.stdout.decode("utf-8", "replace") + r.stderr.decode("utf-8", "replace")


def field(out, prefix):
    for line in out.splitlines():
        line = line.strip()
        if line.startswith(prefix):
            return line[len(prefix):].strip()
    return None


def handshake_ok(out):
    c = field(out, "Cipher is") or ""
    return "New, " in out and "(NONE)" not in c and "Cipher is (NONE)" not in out


def tls12_suites():
    r = subprocess.run(["openssl", "ciphers", "-s", "-tls1_2", "-stdname", "ALL:COMPLEMENTOFALL:@SECLEVEL=0"],
                       capture_output=True, text=True)
    out = []
    for line in r.stdout.splitlines():
        parts = line.split(" - ")
        if len(parts) >= 2 and not parts[0].startswith("TLS_AES") and not parts[0].startswith("TLS_CHACHA20_POLY1305_SHA256"):
            out.append((parts[0].strip(), parts[1].split()[0].strip()))
    return out


def profile(t):
    sni, addr = target(t)
    p = {"target": t, "sni": sni, "address": addr}
    p["versions"] = {v: handshake_ok(s_client(sni, addr, f"-{v}")) for v in ("tls1_3", "tls1_2", "tls1_1", "tls1")}
    p["tls13_suites"] = {s: handshake_ok(s_client(sni, addr, "-tls1_3", "-ciphersuites", s)) for s in TLS13}
    p["tls13_choice_from_all"] = field(s_client(sni, addr, "-tls1_3", "-ciphersuites", ":".join(reversed(TLS13))), "New, TLSv1.3, Cipher is")
    suites = tls12_suites()
    accepted = [std for std, osl in suites if handshake_ok(s_client(sni, addr, "-tls1_2", "-cipher", osl + ":@SECLEVEL=0"))]
    p["tls12_suites"] = accepted
    acc_osl = [osl for std, osl in suites if std in accepted]
    if len(acc_osl) > 1:  # server preference: offer them in two opposite orders
        a = field(s_client(sni, addr, "-tls1_2", "-cipher", ":".join(acc_osl) + ":@SECLEVEL=0"), "New, TLSv1.2, Cipher is")
        b = field(s_client(sni, addr, "-tls1_2", "-cipher", ":".join(reversed(acc_osl)) + ":@SECLEVEL=0"), "New, TLSv1.2, Cipher is")
        p["tls12_server_preference"] = a == b
        p["tls12_choice_from_all"] = [a, b]
    groups = {}
    for g in GROUPS:  # offer only this group, with its key share
        out = s_client(sni, addr, "-tls1_3", "-groups", g)
        groups[g] = {"ok": handshake_ok(out), "negotiated": field(out, "Negotiated TLS1.3 group:") or field(out, "Server Temp Key:")}
    p["tls13_groups"] = groups
    out = s_client(sni, addr, "-tls1_3", "-groups", "x25519:P-256:X25519MLKEM768")  # share x25519, also offer others
    p["tls13_hrr_when_x25519_share_first"] = {"negotiated": field(out, "Negotiated TLS1.3 group:"), "hrr": "HelloRetryRequest" in out}
    out = s_client(sni, addr, "-showcerts", "-status", "-alpn", "h2,http/1.1")
    p["alpn"] = {
        "h2,http/1.1": field(out, "ALPN protocol:"),
        "http/1.1": field(s_client(sni, addr, "-alpn", "http/1.1"), "ALPN protocol:"),
        "none": field(s_client(sni, addr), "ALPN protocol:"),
    }
    p["ocsp_stapled"] = "OCSP Response Status: successful" in out
    p["peer_signature"] = [field(out, "Peer signature type:"), field(out, "Peer signing digest:")]
    p["server_temp_key"] = field(out, "Server Temp Key:") or field(out, "Negotiated TLS1.3 group:")
    p["chain"] = chain(out)
    sess = s_client(sni, addr, "-tls1_3", "-sess_out", "/dev/null", stdin=b"")
    p["tls13_session_tickets"] = sess.count("Post-Handshake New Session Ticket arrived")
    p["h2"] = h2_settings(sni, addr)
    return p


def chain(out):
    certs, cur = [], []
    for line in out.splitlines():
        if "BEGIN CERTIFICATE" in line:
            cur = [line]
        elif cur:
            cur.append(line)
            if "END CERTIFICATE" in line:
                certs.append("\n".join(cur)); cur = []
    res = []
    for pem in certs:
        r = subprocess.run(["openssl", "x509", "-noout", "-subject", "-issuer", "-text"], input=pem.encode(), capture_output=True)
        txt = r.stdout.decode()
        info = {"subject": field(txt, "subject="), "issuer": field(txt, "issuer=")}
        for line in txt.splitlines():
            l = line.strip()
            if l.startswith("Public Key Algorithm:"): info["key"] = l.split(":", 1)[1].strip()
            elif "Public-Key:" in l: info["key_bits"] = l.split("(")[1].split()[0] if "(" in l else l
            elif l.startswith("Signature Algorithm:") and "sig" not in info: info["sig"] = l.split(":", 1)[1].strip()
            elif l.startswith("DNS:"): info["san"] = l
        res.append(info)
    return res


def h2_settings(sni, addr):
    import h2.config, h2.connection, h2.events, h2.settings
    host, port = addr.rsplit(":", 1)
    ctx = ssl.create_default_context()
    ctx.check_hostname, ctx.verify_mode = False, ssl.CERT_NONE  # the profile, not trust
    ctx.set_alpn_protocols(["h2"])
    time.sleep(PAUSE)
    with socket.create_connection((host, int(port)), timeout=10) as raw, ctx.wrap_socket(raw, server_hostname=sni) as s:
        if s.selected_alpn_protocol() != "h2":
            return {"alpn": s.selected_alpn_protocol()}
        c = h2.connection.H2Connection(config=h2.config.H2Configuration(client_side=True))
        c.initiate_connection(); s.sendall(c.data_to_send())
        res, deadline = {"settings": {}, "window_update": None}, time.time() + 3
        s.settimeout(1)
        while time.time() < deadline:
            try:
                data = s.recv(65536)
            except (socket.timeout, TimeoutError):
                break
            if not data:
                break
            for ev in c.receive_data(data):
                if isinstance(ev, h2.events.RemoteSettingsChanged):
                    for k, v in ev.changed_settings.items():
                        try:
                            name = h2.settings.SettingCodes(k).name
                        except ValueError:
                            name = str(int(k))
                        res["settings"][name] = v.new_value
                elif isinstance(ev, h2.events.WindowUpdated) and ev.stream_id == 0:
                    res["window_update"] = ev.delta
            s.sendall(c.data_to_send())
        return res


if __name__ == "__main__":
    json.dump([profile(t) for t in sys.argv[1:]], sys.stdout, indent=1)
    print()
