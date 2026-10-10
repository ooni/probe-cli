#!/usr/bin/env python3
"""Offline tests for extract_hello.py: known ClientHellos (the committed ones, with fresh random
values) are written into synthetic captures of every layout the extractor claims to handle, and
must come back byte for byte. Covers what real captures rarely produce on demand: reordered,
retransmitted and overlapping segments, flows without their SYN, sequence numbers wrapping past
2^32, Ethernet padding, VLAN tags, every supported link type, pcap byte orders and pcapng.
check_extract.py complements this with real OpenSSL connections.

Run: python3 -m unittest test_extract_hello   (from this directory; standard library only)
"""
import os, random, struct, subprocess, sys, tempfile, unittest

HERE = os.path.dirname(os.path.abspath(__file__))
HELLOS = os.path.join(HERE, "..")
sys.path.insert(0, HERE)
import extract_hello  # noqa: E402

CLIENT4, SERVER4 = bytes([192, 0, 2, 1]), bytes([198, 51, 100, 7])
CLIENT6, SERVER6 = bytes.fromhex("20010db8" + "00" * 11 + "01"), bytes.fromhex("20010db8" + "00" * 11 + "07")


def committed(name):
    with open(os.path.join(HELLOS, name), "rb") as f:
        return f.read()


def with_fresh_randoms(hs, rng):
    """The hello as a client would send it: random, session id and key shares filled in."""
    zeroed, out = extract_hello.normalize(hs), bytearray(hs)
    for i in range(len(hs)):  # the positions normalize() zeroes: changing them changes nothing
        flipped = bytearray(hs)
        flipped[i] ^= 0xFF
        try:
            if extract_hello.normalize(bytes(flipped)) == zeroed:
                out[i] = rng.randrange(256)
        except (IndexError, struct.error):
            pass
    assert sum(a != b for a, b in zip(out, zeroed)) > 32 + 16  # random, session id, key shares
    assert extract_hello.normalize(bytes(out)) == zeroed
    return bytes(out)


def with_sni(hs, sni):
    """The hello with its server name replaced by one of the same length."""
    old = extract_hello.parse(hs)["sni"].encode()
    assert len(old) == len(sni) and hs.count(old) == 1
    return hs.replace(old, sni.encode())


def records(hs, sizes=None):
    """hs wrapped in TLS handshake records, split at the given fragment sizes."""
    sizes, out, i = sizes or [len(hs)], b"", 0
    for n in sizes + [len(hs)]:
        if i >= len(hs):
            break
        out += b"\x16\x03\x01" + struct.pack(">H", len(hs[i:i + n])) + hs[i:i + n]
        i += n
    return out


def tcp(sport, dport, seq, flags, payload=b""):
    return struct.pack(">HHIIBBHHH", sport, dport, seq & 0xFFFFFFFF, 1, 5 << 4, flags, 65535, 0, 0) + payload


def ip(src, dst, l4, v6):
    if v6:
        return struct.pack(">IHBB", 6 << 28, len(l4), 6, 64) + src + dst + l4
    return struct.pack(">BBHHHBBH", 0x45, 0, 20 + len(l4), 0, 0, 64, 6, 0) + src + dst + l4


def link(linktype, pkt, v6, vlan=False):
    """pkt framed for the link type; Ethernet frames are padded to the 60-byte minimum, as on a wire."""
    if linktype == 1:
        eth = b"\x02" * 6 + b"\x04" * 6 + (b"\x81\x00\x00\x07" if vlan else b"") + (b"\x86\xdd" if v6 else b"\x08\x00")
        return (eth + pkt).ljust(60, b"\x00")
    if linktype == 113:
        return struct.pack(">HHH8sH", 4, 1, 6, b"\x02" * 6, 0x86DD if v6 else 0x0800) + pkt
    if linktype == 276:
        return struct.pack(">HHIHBB8s", 0x86DD if v6 else 0x0800, 0, 2, 1, 4, 6, b"\x02" * 6) + pkt
    if linktype == 101:
        return pkt
    if linktype == 0:  # DLT_NULL: address family in the capturing host's byte order
        return struct.pack("<I", 30 if v6 else 2) + pkt
    if linktype == 108:  # DLT_LOOP: network byte order
        return struct.pack(">I", 30 if v6 else 2) + pkt
    raise ValueError(linktype)


def flow(stream, sport, isn, cuts=(), order=None, syn=True, duplicate=None, v6=False, server_reply=True):
    """Client and server TCP packets for one connection carrying `stream`, cut at the given
    offsets and sent in `order` (default: in order). duplicate=(start, end) adds a retransmission
    of that range with different boundaries. Returns (src, dst, l4) tuples."""
    c, s = (CLIENT6, SERVER6) if v6 else (CLIENT4, SERVER4)
    edges = [0, *cuts, len(stream)]
    segs = [(isn + 1 + a, stream[a:b]) for a, b in zip(edges, edges[1:])]
    if duplicate:
        a, b = duplicate
        segs.append((isn + 1 + a, stream[a:b]))
    if order:
        segs = [segs[i] for i in order]
    out = []
    if syn:
        out += [(c, s, tcp(sport, 443, isn, 0x02)), (s, c, tcp(443, sport, 9000, 0x12)),
                (c, s, tcp(sport, 443, isn + 1, 0x10))]  # the final ACK has the hello's sequence number
    out += [(c, s, tcp(sport, 443, q, 0x18, p)) for q, p in segs]
    if server_reply:  # a server flight starting like a hello record: must not be taken for one
        out.append((s, c, tcp(443, sport, 9001, 0x18, records(b"\x02\x00\x00\x04\x03\x03\x00\x00"))))
    return out


def pcap(path, linktype, packets, v6=False, endian="<", nano=False, vlan=False):
    magic = 0xA1B23C4D if nano else 0xA1B2C3D4
    with open(path, "wb") as f:
        f.write(struct.pack(endian + "IHHiIII", magic, 2, 4, 0, 0, 65535, linktype))
        for n, (src, dst, l4) in enumerate(packets):
            frame = link(linktype, ip(src, dst, l4, v6), v6, vlan)
            f.write(struct.pack(endian + "IIII", 1700000000, n, len(frame), len(frame)) + frame)


def pcapng(path, linktype, packets, v6=False):
    def block(t, body):
        body += b"\x00" * (-len(body) % 4)
        return struct.pack("<II", t, 12 + len(body)) + body + struct.pack("<I", 12 + len(body))
    with open(path, "wb") as f:
        f.write(block(0x0A0D0D0A, struct.pack("<IHHq", 0x1A2B3C4D, 1, 0, -1)))
        f.write(block(1, struct.pack("<HHI", linktype, 0, 65535) + struct.pack("<HHB3x", 9, 1, 9) + b"\x00" * 4))
        for n, (src, dst, l4) in enumerate(packets):
            frame = link(linktype, ip(src, dst, l4, v6), v6)
            f.write(block(6, struct.pack("<IIIII", 0, 0, n, len(frame), len(frame)) + frame))


class ExtractHelloTest(unittest.TestCase):
    def setUp(self):
        self.dir = tempfile.TemporaryDirectory()
        self.rng = random.Random(1)
        self.hello = committed("android-okhttp-websocket.bin")  # 1473 B, SNI chat.signal.org
        self.sent = with_fresh_randoms(self.hello, self.rng)

    def tearDown(self):
        self.dir.cleanup()

    def extract(self, path, sni="chat.signal.org"):
        out = os.path.join(self.dir.name, "out.bin")
        r = subprocess.run([sys.executable, "-I", os.path.join(HERE, "extract_hello.py"), path, sni, out],
                           capture_output=True, text=True)
        self.assertEqual(r.returncode, 0, r.stderr)
        with open(out, "rb") as f:
            return f.read()

    def roundtrip(self, packets, linktype=1, writer=pcap, **kw):
        path = os.path.join(self.dir.name, "c.pcap")
        writer(path, linktype, packets, **kw)
        return self.extract(path)

    def test_every_committed_hello_whole(self):
        for name in ("libsignal.bin", "android-okhttp-websocket.bin", "android-okhttp-rest.bin", "desktop-electron.bin"):
            with self.subTest(name):
                hs = committed(name)
                path = os.path.join(self.dir.name, "c.pcap")
                pcap(path, 1, flow(records(with_fresh_randoms(hs, self.rng)), 40000, 1000))
                self.assertEqual(self.extract(path, extract_hello.parse(hs)["sni"]), hs)

    def test_segmented_and_reordered(self):
        pkts = flow(records(self.sent), 40000, 1000, cuts=(100, 600, 1200), order=[2, 0, 3, 1])
        self.assertEqual(self.roundtrip(pkts), self.hello)

    def test_retransmission_with_other_boundaries(self):
        pkts = flow(records(self.sent), 40000, 1000, cuts=(500, 1000), duplicate=(300, 900), order=[0, 3, 1, 2])
        self.assertEqual(self.roundtrip(pkts), self.hello)

    def test_fragmented_into_records_across_segments(self):
        pkts = flow(records(self.sent, [300, 300, 300]), 40000, 1000, cuts=(298, 305, 1000))
        self.assertEqual(self.roundtrip(pkts), self.hello)

    def test_syn_not_captured(self):
        pkts = flow(records(self.sent), 40000, 1000, cuts=(700,), syn=False)
        self.assertEqual(self.roundtrip(pkts), self.hello)

    def test_sequence_numbers_wrap_inside_the_hello(self):
        pkts = flow(records(self.sent), 40000, 2**32 - 600, cuts=(400, 800, 1200))
        self.assertEqual(self.roundtrip(pkts), self.hello)

    def test_ethernet_padding_on_the_handshake_ack(self):
        # the client's bare ACK shares the hello's sequence number and is padded to 60 bytes;
        # the padding is not TCP payload
        pkts = flow(records(self.sent), 40000, 1000)
        self.assertEqual(self.roundtrip(pkts, linktype=1), self.hello)

    def test_vlan(self):
        self.assertEqual(self.roundtrip(flow(records(self.sent), 40000, 1000, cuts=(700,)), vlan=True), self.hello)

    def test_link_types_and_ipv6(self):
        for linktype in (1, 113, 276, 101, 0, 108):
            for v6 in (False, True):
                with self.subTest(linktype=linktype, v6=v6):
                    pkts = flow(records(self.sent), 40000, 1000, cuts=(700,), v6=v6)
                    self.assertEqual(self.roundtrip(pkts, linktype=linktype, v6=v6), self.hello)

    def test_pcap_variants_and_pcapng(self):
        pkts = flow(records(self.sent), 40000, 1000, cuts=(700,))
        for kw in (dict(endian=">"), dict(nano=True), dict(endian=">", nano=True)):
            with self.subTest(**kw):
                self.assertEqual(self.roundtrip(pkts, **kw), self.hello)
        self.assertEqual(self.roundtrip(pkts, writer=pcapng), self.hello)

    def test_picks_the_first_connection_with_the_sni(self):
        other = committed("android-okhttp-rest.bin")  # SNI cdn2.signal.org, same length as chat.signal.org
        decoy = flow(records(with_fresh_randoms(other, self.rng)), 39999, 5000)
        later = flow(records(with_sni(with_fresh_randoms(other, self.rng), "chat.signal.org")), 40001, 7000)
        self.assertEqual(self.roundtrip(decoy + flow(records(self.sent), 40000, 1000, cuts=(700,)) + later), self.hello)

    def test_fails_when_the_sni_is_absent(self):
        path = os.path.join(self.dir.name, "c.pcap")
        pcap(path, 1, flow(records(self.sent), 40000, 1000))
        r = subprocess.run([sys.executable, "-I", os.path.join(HERE, "extract_hello.py"), path, "example.org",
                            os.path.join(self.dir.name, "out.bin")], capture_output=True, text=True)
        self.assertNotEqual(r.returncode, 0)
        self.assertFalse(os.path.exists(os.path.join(self.dir.name, "out.bin")))


if __name__ == "__main__":
    unittest.main()
