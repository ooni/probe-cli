#!/usr/bin/env python3
"""Generate local test certificates shaped like Signal's production chain (measured 2026-10 with
tlsprofile.sh): two root CAs like Signal's (RSA 4096 and Ed25519), and leaves with an empty subject,
a critical SAN, extendedKeyUsage serverAuth and critical basicConstraints CA:FALSE.

  chat.signal.org:      an Ed25519 leaf and an RSA 2048 leaf, both signed (sha256WithRSA) by the RSA CA;
                        the server picks one by the client's signature algorithms, as production does.
  grpc.chat.signal.org: an Ed25519 leaf signed by the Ed25519 CA.

The CAs are named as test CAs, not as Signal. Writes PEM files into OUT_DIR (keys as PKCS#8).
Usage: certs.py OUT_DIR   (needs the `cryptography` package)
"""
import datetime, os, sys
from cryptography import x509
from cryptography.hazmat.primitives import hashes, serialization
from cryptography.hazmat.primitives.asymmetric import ed25519, rsa
from cryptography.x509.oid import ExtendedKeyUsageOID, NameOID

out = sys.argv[1]
now = datetime.datetime.now(datetime.timezone.utc)
DAYS = 30


def name(cn):
    return x509.Name([x509.NameAttribute(NameOID.ORGANIZATION_NAME, "OONI signal_client test"),
                      x509.NameAttribute(NameOID.COMMON_NAME, cn)])


def write(fn, data):
    with open(os.path.join(out, fn), "wb") as f:
        f.write(data)


def key_pem(k):
    return k.private_bytes(serialization.Encoding.PEM, serialization.PrivateFormat.PKCS8, serialization.NoEncryption())


def ca(cn, key, sign_hash):
    subject = name(cn)
    cert = (x509.CertificateBuilder().subject_name(subject).issuer_name(subject).public_key(key.public_key())
            .serial_number(x509.random_serial_number()).not_valid_before(now - datetime.timedelta(minutes=5))
            .not_valid_after(now + datetime.timedelta(days=DAYS))
            .add_extension(x509.BasicConstraints(ca=True, path_length=None), critical=True)
            .add_extension(x509.KeyUsage(digital_signature=True, key_cert_sign=True, crl_sign=True, content_commitment=False,
                                         key_encipherment=False, data_encipherment=False, key_agreement=False,
                                         encipher_only=False, decipher_only=False), critical=True)
            .sign(key, sign_hash))
    return cert


def leaf(dns, key, ca_cert, ca_key, sign_hash):
    return (x509.CertificateBuilder().subject_name(x509.Name([])).issuer_name(ca_cert.subject)
            .public_key(key.public_key()).serial_number(x509.random_serial_number())
            .not_valid_before(now - datetime.timedelta(minutes=5)).not_valid_after(now + datetime.timedelta(days=DAYS))
            .add_extension(x509.ExtendedKeyUsage([ExtendedKeyUsageOID.SERVER_AUTH]), critical=False)
            .add_extension(x509.BasicConstraints(ca=False, path_length=None), critical=True)
            .add_extension(x509.SubjectAlternativeName([x509.DNSName(dns)]), critical=True)
            .sign(ca_key, sign_hash))


ca_rsa_key = rsa.generate_private_key(public_exponent=65537, key_size=4096)
ca_rsa = ca("signal_client test CA (RSA)", ca_rsa_key, hashes.SHA256())
ca_ed_key = ed25519.Ed25519PrivateKey.generate()
ca_ed = ca("signal_client test CA (Ed25519)", ca_ed_key, None)

chat_ed_key = ed25519.Ed25519PrivateKey.generate()
chat_rsa_key = rsa.generate_private_key(public_exponent=65537, key_size=2048)
grpc_ed_key = ed25519.Ed25519PrivateKey.generate()
pem = lambda c: c.public_bytes(serialization.Encoding.PEM)

write("ca.pem", pem(ca_rsa) + pem(ca_ed))   # trust anchors for the tests
write("chat-ed25519.key", key_pem(chat_ed_key))
write("chat-ed25519.pem", pem(leaf("chat.signal.org", chat_ed_key, ca_rsa, ca_rsa_key, hashes.SHA256())))
write("chat-rsa.key", key_pem(chat_rsa_key))
write("chat-rsa.pem", pem(leaf("chat.signal.org", chat_rsa_key, ca_rsa, ca_rsa_key, hashes.SHA256())))
write("grpc-ed25519.key", key_pem(grpc_ed_key))
write("grpc-ed25519.pem", pem(leaf("grpc.chat.signal.org", grpc_ed_key, ca_ed, ca_ed_key, None)))
print("wrote", ", ".join(sorted(os.listdir(out))))
