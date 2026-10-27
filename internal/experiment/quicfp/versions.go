package quicfp

// This file enumerates the QUIC version groups from Table 9 ("QUIC Parser
// Fingerprints") of Ablove et al. Each group in the paper is collapsed into
// one representative wire-format version number below; the comment on each
// entry lists the full range the paper tested.
//
// For every representative version we also record the "Initial salt" used to
// derive QUIC's Initial packet-protection keys (RFC 9001 §5.2 / RFC 9369
// §3.3). Only three of these are verified against the relevant RFC text:
//
//   - v1        (0x00000001): RFC 9001, salt 38762cf7...
//   - draft-29  (0xff00001d): draft-ietf-quic-tls-29, salt afbfec28...
//   - v2        (0x6b3343cf): RFC 9369,  salt 0dede3de...
//
// The other legacy IETF drafts (22-28) changed their salt across the draft
// series and gQUIC (Q0xx/T0xx) never used TLS 1.3 Initial protection at all
// -- it used Google's own "QUIC Crypto" handshake. For those rows we reuse
// the draft-29 salt purely so the packet is syntactically well-formed; this
// is NOT protocol-accurate and is only good enough to test whether a given
// version *tag* is recognized/negotiated. If you need a byte-accurate replay
// of a specific legacy draft or of real gQUIC framing, replace the salt (and,
// for gQUIC, the whole header/payload framing in quic.go) accordingly.
type quicVersionVector struct {
	// Name is the vector name used in results (VectorResult.Name).
	Name string
	// Group documents the full version range this vector represents,
	// copied from Table 9's row label.
	Group string
	// Version is the 32-bit wire-format version number placed in the
	// QUIC long header.
	Version uint32
	// Salt is the 20-byte Initial salt used to derive Initial secrets
	// for this version (RFC 9001 §5.2). Verified is false when we do
	// not have a citation for the exact value (see comment above).
	Salt     [20]byte
	Verified bool
}

func mustSalt(hexBytes ...byte) [20]byte {
	var out [20]byte
	copy(out[:], hexBytes)
	return out
}

// saltV1 is RFC 9001's Initial salt, used for QUIC version 1.
var saltV1 = mustSalt(0x38, 0x76, 0x2c, 0xf7, 0xf5, 0x59, 0x34, 0xb3, 0x4d, 0x17,
	0x9a, 0xe6, 0xa4, 0xc8, 0x0c, 0xad, 0xcc, 0xbb, 0x7f, 0x0a)

// saltDraft29 is the salt used by draft-ietf-quic-tls-29 (and reused below
// as a best-effort stand-in for other legacy drafts / gQUIC versions).
var saltDraft29 = mustSalt(0xaf, 0xbf, 0xec, 0x28, 0x99, 0x93, 0xd2, 0x4c, 0x9e, 0x97,
	0x86, 0xf1, 0x9c, 0x61, 0x11, 0xe0, 0x43, 0x90, 0xa8, 0x99)

// saltV2 is RFC 9369's Initial salt, used for QUIC version 2.
var saltV2 = mustSalt(0x0d, 0xed, 0xe3, 0xde, 0xf7, 0x00, 0xa6, 0xdb, 0x81, 0x93,
	0x81, 0xbe, 0x6e, 0x26, 0x9d, 0xcb, 0xf9, 0xbd, 0x2e, 0xd9)

// gquicTag builds a 4-byte ASCII gQUIC version tag (e.g. "Q043", "T050") as
// a big-endian uint32, matching how Google QUIC placed the ASCII tag in the
// version field of the long header.
func gquicTag(tag string) uint32 {
	if len(tag) != 4 {
		panic("gquicTag: tag must be exactly 4 ASCII characters")
	}
	return uint32(tag[0])<<24 | uint32(tag[1])<<16 | uint32(tag[2])<<8 | uint32(tag[3])
}

// ietfDraft builds the wire-format version number for an IETF QUIC draft,
// per the draft-quic-transport convention 0xff0000<draft-number>.
func ietfDraft(n uint32) uint32 {
	return 0xff000000 | n
}

// quicVersionVectors returns the seven representative version vectors
// corresponding, in order, to the seven rows of Table 9's "QUIC Version"
// section.
func quicVersionVectors() []quicVersionVector {
	return []quicVersionVector{
		{
			Name:     "quic_version_gquic_q023_q043",
			Group:    "GQUIC: Q023-Q040, Q042-Q043",
			Version:  gquicTag("Q043"),
			Salt:     saltDraft29, // best-effort, see file-level comment
			Verified: false,
		},
		{
			Name:     "quic_version_gquic_q041_q045",
			Group:    "GQUIC: Q041, Q044-Q045",
			Version:  gquicTag("Q044"),
			Salt:     saltDraft29, // best-effort, see file-level comment
			Verified: false,
		},
		{
			Name:     "quic_version_gquic_q046_q049",
			Group:    "GQUIC: Q046-Q049",
			Version:  gquicTag("Q046"),
			Salt:     saltDraft29, // best-effort, see file-level comment
			Verified: false,
		},
		{
			Name:     "quic_version_gquic_t050_t051",
			Group:    "GQUIC: T050-T051",
			Version:  gquicTag("T050"),
			Salt:     saltDraft29, // best-effort, see file-level comment
			Verified: false,
		},
		{
			Name:     "quic_version_ietf_draft22_draft28",
			Group:    "IETF: draft-22-28",
			Version:  ietfDraft(27), // representative mid-range draft
			Salt:     saltDraft29,   // best-effort, see file-level comment
			Verified: false,
		},
		{
			Name:     "quic_version_ietf_draft29_v1",
			Group:    "IETF: draft-29, v1",
			Version:  1, // use the fully-verified v1 salt/version
			Salt:     saltV1,
			Verified: true,
		},
		{
			Name:     "quic_version_ietf_v2",
			Group:    "IETF: v2",
			Version:  0x6b3343cf,
			Salt:     saltV2,
			Verified: true,
		},
	}
}

// gquicVersionSet reports whether version v is one of the gQUIC vectors
// above, for which we do not attempt real Google "QUIC Crypto" framing.
func isGQUICVersion(v uint32) bool {
	b0 := byte(v >> 24)
	return b0 == 'Q' || b0 == 'T'
}

// greasedVersion returns a reserved/unassigned QUIC version number of the
// form 0x?a?a?a?a, as recommended by RFC 9000 §15 for exercising version
// negotiation without colliding with any real, assigned version.
func greasedVersion() uint32 {
	return 0x1a2a3a4a
}
