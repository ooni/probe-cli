#!/usr/bin/env python3
"""Extract the first TLS ClientHello sent to SNI from a pcap and write it as a handshake message
(starting with type 1, no record header), the format hellos.go embeds. Prints its JA4 and JA3.

The per-connection random values (client random, legacy session id, key share public keys) are
zeroed, keeping their lengths, so a refresh changes the file only when the handshake changes. uTLS
generates fresh values for each of them anyway.

Usage: extract_hello.py PCAP SNI OUT.bin
       extract_hello.py --normalize FILE.bin...   (zero the random values of existing files)
Standalone (Python 3 standard library only). Link types: Ethernet (pasta's --pcap), Linux cooked v1/v2, raw IP.
"""
import hashlib, ipaddress, struct, sys

GREASE = lambda v: (v & 0x0F0F) == 0x0A0A and (v >> 8) == (v & 0xFF)


def packets(path):
    d = open(path, "rb").read()
    e = {b"\xd4\xc3\xb2\xa1": "<", b"\x4d\x3c\xb2\xa1": "<", b"\xa1\xb2\xc3\xd4": ">", b"\xa1\xb2\x3c\x4d": ">"}.get(d[:4])
    if not e:
        sys.exit("not a pcap file (pcapng is not supported)")
    off = {1: 14, 113: 16, 276: 20, 101: 0, 228: 0, 229: 0}[struct.unpack(e + "I", d[20:24])[0]]
    i = 24
    while i + 16 <= len(d):
        incl = struct.unpack(e + "I", d[i + 8:i + 12])[0]
        pkt = d[i + 16:i + 16 + incl]; i += 16 + incl
        ip = pkt[off:]
        if not ip: continue
        if ip[0] >> 4 == 4:
            proto, src, dst, l4 = ip[9], ip[12:16], ip[16:20], ip[(ip[0] & 15) * 4:]
        elif ip[0] >> 4 == 6:
            proto, src, dst, l4 = ip[6], ip[8:24], ip[24:40], ip[40:]
        else:
            continue
        if proto == 6 and len(l4) >= 20:
            yield src, dst, l4


def client_streams(path):
    flows = {}
    for src, dst, l4 in packets(path):
        sp, dp, seq = struct.unpack(">HHI", l4[:8]); flags = l4[13]; payload = l4[(l4[12] >> 4) * 4:]
        key = (src, sp, dst, dp)
        if flags & 0x02 and not flags & 0x10:
            flows[key] = {"next": seq + 1, "segs": {}}
        elif key in flows and payload:
            flows[key]["segs"][seq] = payload
    for f in flows.values():
        data, nxt = b"", f["next"]
        for s in sorted(f["segs"]):
            seg = f["segs"][s]
            if s <= nxt < s + len(seg):
                data += seg[nxt - s:]; nxt = s + len(seg)
        yield data


def client_hello(stream):
    if len(stream) < 9 or stream[0] != 0x16:
        return None
    hs, i = b"", 0
    while i + 5 <= len(stream) and stream[i] == 0x16:
        n = struct.unpack(">H", stream[i + 3:i + 5])[0]; hs += stream[i + 5:i + 5 + n]; i += 5 + n
        if len(hs) >= 4 and len(hs) >= 4 + int.from_bytes(hs[1:4], "big"):
            break
    if len(hs) < 4 or hs[0] != 1:
        return None
    return hs[:4 + int.from_bytes(hs[1:4], "big")]


def parse(hs):
    i = 4; legacy = struct.unpack(">H", hs[i:i + 2])[0]; i += 34; i += 1 + hs[i]
    n = struct.unpack(">H", hs[i:i + 2])[0]; ciphers = list(struct.unpack(f">{n // 2}H", hs[i + 2:i + 2 + n])); i += 2 + n
    i += 1 + hs[i]
    end = i + 2 + struct.unpack(">H", hs[i:i + 2])[0]; i += 2
    ch = dict(legacy=legacy, ciphers=ciphers, exts=[], sni="", alpn=[], groups=[], sigs=[], pf=[], versions=[])
    while i + 4 <= end:
        t, l = struct.unpack(">HH", hs[i:i + 4]); v = hs[i + 4:i + 4 + l]; i += 4 + l; ch["exts"].append(t)
        if t == 0: ch["sni"] = v[5:].decode(errors="replace")
        elif t == 16:
            j = 2
            while j < len(v): ch["alpn"].append(v[j + 1:j + 1 + v[j]].decode()); j += 1 + v[j]
        elif t == 10: ch["groups"] = list(struct.unpack(f">{(l - 2) // 2}H", v[2:]))
        elif t == 13: ch["sigs"] = list(struct.unpack(f">{(l - 2) // 2}H", v[2:]))
        elif t == 11: ch["pf"] = list(v[1:])
        elif t == 43: ch["versions"] = list(struct.unpack(f">{v[0] // 2}H", v[1:1 + v[0]]))
    return ch


def normalize(hs):
    """Zero the client random, the legacy session id and the key share values."""
    b = bytearray(hs)
    b[6:38] = bytes(32)
    i = 38; sid = b[i]; b[i + 1:i + 1 + sid] = bytes(sid); i += 1 + sid
    n = struct.unpack(">H", b[i:i + 2])[0]; i += 2 + n
    i += 1 + b[i]
    end = i + 2 + struct.unpack(">H", b[i:i + 2])[0]; i += 2
    while i + 4 <= end:
        t, l = struct.unpack(">HH", b[i:i + 4])
        if t == 51:  # key_share: zero each key_exchange
            j = i + 4 + 2
            while j + 4 <= i + 4 + l:
                kl = struct.unpack(">H", b[j + 2:j + 4])[0]
                if not GREASE(struct.unpack(">H", b[j:j + 2])[0]):
                    b[j + 4:j + 4 + kl] = bytes(kl)
                j += 4 + kl
        i += 4 + l
    return bytes(b)


def ja3(ch):
    f = lambda xs: "-".join(str(x) for x in xs if not GREASE(x))
    s = f"{ch['legacy']},{f(ch['ciphers'])},{f(ch['exts'])},{f(ch['groups'])},{'-'.join(map(str, ch['pf']))}"
    return hashlib.md5(s.encode()).hexdigest()


def ja4(ch):
    vers = [v for v in ch["versions"] if not GREASE(v)] or [ch["legacy"]]
    ver = {0x0304: "13", 0x0303: "12", 0x0302: "11", 0x0301: "10"}.get(max(vers), "00")
    ciphers = [c for c in ch["ciphers"] if not GREASE(c)]; exts = [e for e in ch["exts"] if not GREASE(e)]
    alpn = ch["alpn"][0] if ch["alpn"] else ""
    try:
        ipaddress.ip_address(ch["sni"]); d = "i"
    except ValueError:
        d = "d" if ch["sni"] else "i"
    h12 = lambda s: hashlib.sha256(s.encode()).hexdigest()[:12] if s else "000000000000"
    sigs = ",".join(f"{s:04x}" for s in ch["sigs"] if not GREASE(s))
    rest = ",".join(f"{e:04x}" for e in sorted(x for x in exts if x not in (0, 16))) + ("_" + sigs if sigs else "")
    return (f"t{ver}{d}{len(ciphers):02d}{len(exts):02d}{(alpn[0] + alpn[-1]) if alpn else '00'}_"
            f"{h12(','.join(f'{c:04x}' for c in sorted(ciphers)))}_{h12(rest)}")


def main():
    if len(sys.argv) >= 3 and sys.argv[1] == "--normalize":
        for f in sys.argv[2:]:
            data = normalize(open(f, "rb").read())
            open(f, "wb").write(data)
            print(f"{f}: normalized")
        return
    if len(sys.argv) != 4:
        sys.exit(__doc__)
    pcap, sni, out = sys.argv[1:]
    for stream in client_streams(pcap):
        hs = client_hello(stream)
        if hs and parse(hs)["sni"] == sni:
            hs = normalize(hs)
            open(out, "wb").write(hs)
            ch = parse(hs)
            print(f"{out}: {len(hs)} bytes, SNI {sni}, ALPN {','.join(ch['alpn']) or '(none)'}, JA4 {ja4(ch)}, JA3 {ja3(ch)}")
            return
    sys.exit(f"no ClientHello for {sni} in {pcap}")


if __name__ == "__main__":
    main()
