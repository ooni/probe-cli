#!/usr/bin/env python3
"""Check the hello extractor against ground truth: real OpenSSL connections, captured with pasta.

For each case, `openssl s_client -msg` connects from a rootless container whose traffic pasta
captures (--network pasta:--pcap). OpenSSL prints the exact ClientHello it sent; the extractor
reads the capture. The two must agree byte for byte (after zeroing the random values, as
extract_hello.py does), and the JA4 fields that follow from the case's options (TLS version, SNI,
ALPN) must hold. Cases cover IPv4 and IPv6, hellos split over two and three TCP segments, a hello
fragmented over four TLS records, a HelloRetryRequest (the second hello must not win), and decoy
connections in the same capture. Each case also asserts that it exercised what it is there for
(segment, record and hello counts), so a change in pasta or OpenSSL cannot make it pass vacuously.

With --fingerprint PATH it also checks the android-capture harness's fingerprint.py, which keeps
raw bytes (so these must match exactly), on TCP and on QUIC (decrypting the Initial packets).

Needs podman with pasta and network access. Not part of the Go tests: run it after changing the
extractor.  Usage: check_extract.py [--fingerprint PATH/fingerprint.py] [--keep DIR]
"""
import argparse, json, os, re, shutil, struct, subprocess, sys, tempfile

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import extract_hello  # noqa: E402

IMAGE = "docker.io/library/python:3.12-slim"  # OpenSSL >= 3.5: ML-KEM key shares and a QUIC client
TOOLS_IMAGE = "localhost/capture-tools"       # fingerprint.py's QUIC path needs `cryptography`
A, B = "www.cloudflare.com", "www.google.com"


# Each case: the connections to make (host, extra s_client options), the index of the one under
# test, the JA4 fields its options imply (transport+version, SNI flag, ALPN), pasta options, and
# what it must exercise: minimum TCP segments and TLS records carrying the hello, ClientHellos sent.
def case(conns, idx, ja4, pasta=("--mtu", "1280"), segments=1, records=1, hellos=1):
    return dict(conns=conns, idx=idx, ja4=ja4, pasta=pasta, segments=segments, records=records, hellos=hellos)


CASES = {  # MTU 1280: an ML-KEM hello (~1.5 KB) spans two TCP segments
    "ipv6-mlkem-2-segments": case([(A, ["-6"])], 0, ("t13", "d", "00"), segments=2),
    "ipv4-mlkem-2-segments": case([(A, ["-4"])], 0, ("t13", "d", "00"), segments=2),
    "ipv4-mtu576-3-segments": case([(A, ["-4"])], 0, ("t13", "d", "00"), pasta=("--mtu", "576", "-4"), segments=3),
    "tls12-only": case([(A, ["-4", "-tls1_2"])], 0, ("t12", "d", "00")),
    "alpn-h2-http11": case([(A, ["-4", "-alpn", "h2,http/1.1"])], 0, ("t13", "d", "h2"), segments=2),
    "x25519-small": case([(A, ["-4", "-groups", "x25519"])], 0, ("t13", "d", "00")),
    "4-records-over-3-segments": case([(A, ["-4", "-max_send_frag", "512"])], 0, ("t13", "d", "00"),
                                      pasta=("--mtu", "576", "-4"), segments=3, records=4),
    "hello-retry-request": case([(A, ["-4", "-groups", "ffdhe2048:x25519"])], 0, ("t13", "d", "00"), hellos=2),
    "decoy-other-sni-first": case([(B, ["-4"]), (A, ["-4", "-alpn", "http/1.1"])], 1, ("t13", "d", "h1"), segments=2),
    "same-sni-twice-first-wins": case([(A, ["-4", "-alpn", "h2"]), (A, ["-4"])], 0, ("t13", "d", "h2"), segments=2),
}
FINGERPRINT_ONLY = {  # extract_hello.py selects by SNI and reads TCP only
    "no-sni": case([(A, ["-4", "-noservername"])], 0, ("t13", "i", "00")),
    "quic-h3": case([(A, ["-quic", "-alpn", "h3"])], 0, ("q13", "d", "h3"), segments=0, records=0),
}


def openssl_hellos(msg):
    """The ClientHello messages OpenSSL's -msg output says it sent, in order."""
    out, lines = [], msg.splitlines()
    for i, line in enumerate(lines):
        m = re.match(r">>> .*Handshake \[length ([0-9a-f]+)\], ClientHello", line)
        if not m:
            continue
        hexes = []
        for h in lines[i + 1:]:
            if not h.startswith(" "):
                break
            hexes.append(h.strip())
        b = bytes.fromhex(" ".join(hexes))
        assert len(b) == int(m.group(1), 16), f"-msg length {m.group(1)} but {len(b)} bytes"
        out.append(b)
    return out


def capture(work, c):
    """Run the case's connections one after another in one container; return (pcap, [-msg output])."""
    pcap = os.path.join(work, "capture.pcap")
    script = "; ".join(
        f"openssl s_client -msg -connect {h}:443 {'' if '-noservername' in o else '-servername ' + h} "
        f"{' '.join(o)} </dev/null >/w/msg{i} 2>&1" for i, (h, o) in enumerate(c["conns"]))
    net = ",".join(("pasta:--pcap", pcap) + tuple(c["pasta"]))
    subprocess.run(["podman", "run", "--rm", "--network", net, "-v", f"{work}:/w:Z", IMAGE, "sh", "-c", script],
                   check=True, capture_output=True, timeout=120)
    return pcap, [open(os.path.join(work, f"msg{i}")).read() for i in range(len(c["conns"]))]


def hello_layout(pcap, idx, hello_len):
    """(TCP data segments, TLS records) that carry the hello of the idx-th TCP connection to port
    443, connections ordered by their SYNs. Independent of the extractor's reassembly: the
    connections here have no loss or reordering, so segments are taken in capture order."""
    flows = {}
    for src, _dst, l4 in extract_hello.packets(pcap):
        sp, dp = struct.unpack(">HH", l4[:4])
        if dp != 443:
            continue
        if l4[13] & 0x02:
            flows.setdefault((src, sp), [])
        elif (src, sp) in flows and len(l4) > (l4[12] >> 4) * 4:
            flows[(src, sp)].append(l4[(l4[12] >> 4) * 4:])
    segs = list(flows.values())[idx]
    data, nseg, i, records, got = b"", 0, 0, 0, 0
    for s in segs:
        data += s; nseg += 1
        while i + 5 <= len(data) and data[i] == 0x16 and got < hello_len:
            n = struct.unpack(">H", data[i + 3:i + 5])[0]
            if i + 5 + n > len(data):
                break
            records += 1; got += n; i += 5 + n
        if got >= hello_len:
            break
    return nseg, records


def ja4_fields(ja4):
    return (ja4[:3], ja4[3], ja4[8:10])


def check_case(c, work, fingerprint, tcp):
    pcap, msgs = capture(work, c)
    idx, (host, opts) = c["idx"], c["conns"][c["idx"]]
    sent = openssl_hellos(msgs[idx])
    if not sent:
        return [f"OpenSSL printed no ClientHello: {msgs[idx][-300:]!r}"], []
    truth, problems = sent[0], []
    notes = [f"{len(truth)} B hello"]
    if len(sent) < c["hellos"]:
        problems.append(f"OpenSSL sent {len(sent)} ClientHello(s), the case needs {c['hellos']}")
    elif c["hellos"] > 1:
        notes.append(f"{len(sent)} hellos sent")
    if tcp:
        nseg, nrec = hello_layout(pcap, idx, len(truth))
        notes.append(f"{nseg} segment(s), {nrec} record(s)")
        if nseg < c["segments"] or nrec < c["records"]:
            problems.append(f"the hello took {nseg} segment(s) and {nrec} record(s); the case needs "
                            f"{c['segments']} and {c['records']}")
    if tcp and "-noservername" not in opts:
        out = os.path.join(work, "extracted.bin")
        r = subprocess.run([sys.executable, "-I", os.path.join(HERE, "extract_hello.py"), pcap, host, out],
                           capture_output=True, text=True)
        if r.returncode != 0:
            problems.append(f"extract_hello.py failed: {r.stderr.strip()[-200:]}")
        else:
            got = open(out, "rb").read()
            if got != extract_hello.normalize(truth):
                problems.append(f"extract_hello.py bytes differ from OpenSSL's ({len(got)} B vs {len(truth)} B)")
            m = re.search(r"JA4 ([^\s,]+)", r.stdout)
            if not m or ja4_fields(m.group(1)) != c["ja4"]:
                problems.append(f"extract_hello.py JA4 {m.group(1) if m else None}, want fields {c['ja4']}")
            notes.append(f"extract_hello {m.group(1) if m else '-'}")
    if fingerprint:
        fp = os.path.dirname(os.path.abspath(fingerprint))
        r = subprocess.run(["podman", "run", "--rm", "-v", f"{work}:/w:Z", "-v", f"{fp}:/h:ro,Z", TOOLS_IMAGE,
                            "python3", "-I", f"/h/{os.path.basename(fingerprint)}", "/w/capture.pcap", "--json",
                            "--raw-dir", "/w/raw"], capture_output=True, text=True)
        if r.returncode != 0:
            problems.append(f"fingerprint.py failed: {r.stderr.strip()[-200:]}")
            return problems, notes
        transport = "tcp" if tcp else "quic"
        rows = [(n, x) for n, x in enumerate(json.loads(r.stdout)["connections"]) if x["transport"] == transport]
        if len(rows) != len(c["conns"]):
            problems.append(f"fingerprint.py found {len(rows)} {transport} handshakes, want {len(c['conns'])}")
            return problems, notes
        n, row = rows[idx]  # rows are in time order, as the connections were made
        files = [f for f in os.listdir(os.path.join(work, "raw")) if f.startswith(f"{n:03d}-")]
        raw = open(os.path.join(work, "raw", files[0]), "rb").read() if files else b""
        raw = raw[:4 + int.from_bytes(raw[1:4], "big")] if len(raw) >= 4 else raw  # TLS 1.2: more follows
        if raw != truth:
            problems.append(f"fingerprint.py raw bytes differ from OpenSSL's ({len(raw)} B vs {len(truth)} B)")
        if ja4_fields(row["ja4"]) != c["ja4"]:
            problems.append(f"fingerprint.py JA4 {row['ja4']}, want fields {c['ja4']}")
        if row["sni"] != ("" if "-noservername" in opts else host):
            problems.append(f"fingerprint.py SNI {row['sni']!r}")
        notes.append(f"fingerprint {row['ja4']}")
    return problems, notes


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--fingerprint", help="also check this fingerprint.py (TCP and QUIC)")
    ap.add_argument("--keep", help="keep each case's capture and -msg output under this directory")
    a = ap.parse_args()
    cases = [(n, c, True) for n, c in CASES.items()]
    if a.fingerprint:
        cases += [(n, c, c["segments"] > 0) for n, c in FINGERPRINT_ONLY.items()]
    failed = 0
    for name, c, tcp in cases:
        work = os.path.join(a.keep, name) if a.keep else tempfile.mkdtemp(prefix="check-extract-")
        os.makedirs(work, exist_ok=True)
        try:
            problems, notes = check_case(c, work, a.fingerprint, tcp)
        except Exception as e:  # a case that cannot run is a failure, not a pass
            problems, notes = [f"{type(e).__name__}: {e}"], []
        finally:
            if not a.keep:
                shutil.rmtree(work, ignore_errors=True)
        failed += bool(problems)
        print(f"{'FAIL' if problems else 'ok  '} {name:27} {'; '.join(notes)}")
        for p in problems:
            print(f"     {p}")
    print(f"{len(cases) - failed}/{len(cases)} cases agree with OpenSSL")
    sys.exit(1 if failed else 0)


if __name__ == "__main__":
    main()
