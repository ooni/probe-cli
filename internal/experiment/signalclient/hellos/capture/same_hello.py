#!/usr/bin/env python3
"""Check that ClientHello files (as extract_hello.py writes them) are the same hello: identical but
for the SNI and the padding extension's length, which follows the SNI's length. Exits 1 otherwise.
Usage: same_hello.py A.bin B.bin...
"""
import struct, sys


def shape(raw):
    if raw[:1] != b"\x01":
        raise ValueError("not a ClientHello handshake message")
    body, i = raw[4:], 2 + 32
    i += 1 + body[i]                                                  # legacy session id
    i += 2 + struct.unpack(">H", body[i:i + 2])[0]                    # cipher suites
    i += 1 + body[i]                                                  # compression methods
    head, end, i, exts = body[:i], i + 2 + struct.unpack(">H", body[i:i + 2])[0], i + 2, []
    while i < end:
        t, n = struct.unpack(">HH", body[i:i + 4])
        exts.append((t, b"" if t in (0, 21) else body[i + 4:i + 4 + n]))  # 0: SNI, 21: padding
        i += 4 + n
    return head, exts, len(raw) if 21 in [t for t, _ in exts] else None


files = sys.argv[1:]
shapes = [shape(open(f, "rb").read()) for f in files]
bad = [f for f, s in zip(files, shapes) if s != shapes[0]]
for f in bad:
    print(f"same_hello.py: {f} differs from {files[0]} (beyond the SNI and padding)", file=sys.stderr)
sys.exit(1 if bad or not files else 0)
