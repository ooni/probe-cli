#!/usr/bin/env python3
"""Summarize a pasta capture (classic pcap, Ethernet): DNS queries, and for each TCP connection
its server, SNI and ALPN, timing, bytes each way, and its TLS records (time, direction, type,
length), so an app's traffic shape can be read and compared.
Usage: analyze.py capture.pcap [--records]
"""
import ipaddress
import struct
import sys

path = sys.argv[1]
show_records = "--records" in sys.argv
data = open(path, "rb").read()
magic = struct.unpack("<I", data[:4])[0]
e = "<" if magic in (0xA1B2C3D4, 0xA1B23C4D) else ">"
nano = magic in (0xA1B23C4D, 0x4D3CB2A1)
i, packets = 24, []
while i + 16 <= len(data):
    sec, frac, incl, _ = struct.unpack(e + "IIII", data[i:i + 16])
    packets.append((sec + frac / (1e9 if nano else 1e6), data[i + 16:i + 16 + incl]))
    i += 16 + incl
t0 = packets[0][0]


def ip_of(b):
    return str(ipaddress.ip_address(b))


dns, flows = [], {}
for ts, pkt in packets:
    if pkt[12:14] == b"\x08\x00":
        ip = pkt[14:]; n = struct.unpack(">H", ip[2:4])[0]; ip = ip[:n]
        proto, src, dst, l4 = ip[9], ip[12:16], ip[16:20], ip[(ip[0] & 15) * 4:]
    elif pkt[12:14] == b"\x86\xdd":
        ip = pkt[14:]; proto, src, dst, l4 = ip[6], ip[8:24], ip[24:40], ip[40:40 + struct.unpack(">H", ip[4:6])[0]]
    else:
        continue
    if proto == 17 and len(l4) >= 8:
        sp, dp = struct.unpack(">HH", l4[:4])
        if dp == 53 and len(l4) > 20:  # a DNS query: its first question
            q, j = l4[8 + 12:], 0
            labels = []
            while j < len(q) and q[j]:
                labels.append(q[j + 1:j + 1 + q[j]].decode(errors="replace")); j += 1 + q[j]
            qtype = struct.unpack(">H", q[j + 1:j + 3])[0] if j + 3 <= len(q) else 0
            dns.append((ts - t0, ".".join(labels), {1: "A", 28: "AAAA", 65: "HTTPS"}.get(qtype, qtype)))
    if proto != 6 or len(l4) < 20:
        continue
    sp, dp, seq = struct.unpack(">HHI", l4[:8]); flags = l4[13]; payload = l4[(l4[12] >> 4) * 4:]
    client = dp in (443, 80) or (sp not in (443, 80) and dp < sp)
    key = (ip_of(src), sp, ip_of(dst), dp) if client else (ip_of(dst), dp, ip_of(src), sp)
    f = flows.setdefault(key, {"t0": ts - t0, "t": ts - t0, "seg": {"up": {}, "down": {}}, "base": {}, "fin": None})
    f["t"] = ts - t0
    d = "up" if client else "down"
    if flags & 0x02:
        f["base"][d] = (seq + 1) & 0xFFFFFFFF
    if flags & 0x05 and f["fin"] is None:  # FIN or RST
        f["fin"] = (ts - t0, d, "RST" if flags & 0x04 else "FIN")
    if payload:
        base = f["base"].setdefault(d, seq)
        f["seg"][d].setdefault((seq - base) & 0xFFFFFFFF, (ts - t0, payload))


def stream(segs):
    """Reassembled bytes, and for each byte offset where a segment starts, its time."""
    out, times, nxt = b"", [], 0
    for off in sorted(segs):
        ts, seg = segs[off]
        if off <= nxt < off + len(seg):
            times.append((nxt, ts)); out += seg[nxt - off:]; nxt = off + len(seg)
    return out, times


def records(buf, times):
    """TLS records: (time, type, length)."""
    out, i, k = [], 0, 0
    while i + 5 <= len(buf) and buf[i] in (20, 21, 22, 23):
        while k + 1 < len(times) and times[k + 1][0] <= i:
            k += 1
        length = struct.unpack(">H", buf[i + 3:i + 5])[0]
        out.append((times[k][1] if times else 0, {20: "CCS", 21: "alert", 22: "hs", 23: "app"}[buf[i]], length))
        i += 5 + length
    return out


def sni_alpn(buf):
    if len(buf) < 50 or buf[0] != 22 or buf[5] != 1:
        return "", ""
    body = buf[9:]; j = 34
    j += 1 + body[j]; j += 2 + struct.unpack(">H", body[j:j + 2])[0]; j += 1 + body[j]
    end = j + 2 + struct.unpack(">H", body[j:j + 2])[0]; j += 2; sni = alpn = ""
    while j + 4 <= end:
        t, n = struct.unpack(">HH", body[j:j + 4]); x = body[j + 4:j + 4 + n]
        if t == 0:
            sni = x[5:].decode(errors="replace")
        if t == 16:
            k, names = 2, []
            while k < len(x):
                names.append(x[k + 1:k + 1 + x[k]].decode(errors="replace")); k += 1 + x[k]
            alpn = ",".join(names)
        j += 4 + n
    return sni, alpn


print(f"{len(packets)} packets over {packets[-1][0] - t0:.1f} s")
print("\nDNS queries (time, name, type):")
seen = set()
for ts, name, qt in dns:
    if (name, qt) not in seen:
        seen.add((name, qt)); print(f"  {ts:7.2f}  {name} {qt}")
print("\nTCP connections (start, end, server, SNI, ALPN, bytes up/down, close):")
for key, f in sorted(flows.items(), key=lambda kv: kv[1]["t0"]):
    up, up_t = stream(f["seg"]["up"]); down, down_t = stream(f["seg"]["down"])
    sni, alpn = sni_alpn(up)
    close = f"{f['fin'][2]} by {'client' if f['fin'][1] == 'up' else 'server'} at {f['fin'][0]:.2f}" if f["fin"] else "open"
    print(f"  {f['t0']:7.2f} {f['t']:7.2f}  {key[2]}:{key[3]:<5} {sni or '-':28} {alpn or '-':14} {len(up):7d}/{len(down):<8d} {close}")
    if show_records and sni:
        rs = [(t, "C>", ty, n) for t, ty, n in records(up, up_t)] + [(t, "<S", ty, n) for t, ty, n in records(down, down_t)]
        for t, d, ty, n in sorted(rs):
            print(f"            {t:7.2f} {d} {ty:5} {n}")
