package quicfp

import "fmt"

// Category labels used in VectorResult.Category, one per section of Table 9.
const (
	CategoryQUICVersion = "quic_version"
	CategoryQUICFeature = "quic_feature"
	CategoryTLSLength   = "tls_length_field"
)

// TestVector describes one probe to send: a human-readable name/category
// (mirroring one cell of Table 9) and a Build function that produces the
// raw UDP datagram(s) to transmit for a single repetition.
type TestVector struct {
	Name        string
	Category    string
	Field       string // set for CategoryTLSLength vectors, e.g. "message_length"
	Permutation string // set for CategoryTLSLength vectors
	QUICGroup   string // set for CategoryQUICVersion vectors, copied from Table 9's row label
	QUICVersion uint32 // the version placed in the long header, when applicable

	// Build returns one or more raw UDP datagrams representing this
	// vector's probe. Most vectors return exactly one datagram; the
	// "multi_packet_initial" feature vector returns two, which must be
	// sent back-to-back on the same 4-tuple.
	Build func() ([][]byte, connectionID, connectionID, error)
}

// tlsLengthFieldSpec associates a Table 9 field name with the tlsField it
// corresponds to on a built ClientHello, and which permutations the paper
// applied to it (Section 6.4: every field gets zero/half/double/max; only
// Message Length and Extensions Length additionally get
// exclude_last_extension).
type tlsLengthFieldSpec struct {
	FieldName    string
	Select       func(ch *clientHello) tlsField
	Permutations []string
}

func tlsLengthFieldSpecs() []tlsLengthFieldSpec {
	basic := []string{PermZero, PermHalf, PermDouble, PermMax}
	withExclude := append(append([]string{}, basic...), PermExcludeLastExtension)
	return []tlsLengthFieldSpec{
		{
			FieldName:    "message_length",
			Select:       func(ch *clientHello) tlsField { return ch.MessageLength },
			Permutations: withExclude,
		},
		{
			FieldName: "session_id_length",
			Select:    func(ch *clientHello) tlsField { return ch.SessionIDLength },
			// The paper tests only one value for this field ("Too
			// Long"), not the usual zero/half/double/max set: the
			// field's correct value is 0 (empty session ID), so
			// "half"/"double" of that would be meaningless.
			Permutations: []string{PermTooLong},
		},
		{
			FieldName:    "cipher_suite_length",
			Select:       func(ch *clientHello) tlsField { return ch.CipherSuiteLength },
			Permutations: basic,
		},
		{
			FieldName:    "compression_methods_length",
			Select:       func(ch *clientHello) tlsField { return ch.CompressionMethodsLength },
			Permutations: basic,
		},
		{
			FieldName:    "extensions_length",
			Select:       func(ch *clientHello) tlsField { return ch.ExtensionsLength },
			Permutations: withExclude,
		},
		{
			FieldName:    "sni_extension_length",
			Select:       func(ch *clientHello) tlsField { return ch.SNIExtensionLength },
			Permutations: basic,
		},
		{
			FieldName:    "sni_list_length",
			Select:       func(ch *clientHello) tlsField { return ch.SNIListLength },
			Permutations: basic,
		},
		{
			FieldName:    "sni_name_length",
			Select:       func(ch *clientHello) tlsField { return ch.SNINameLength },
			Permutations: basic,
		},
	}
}

// buildVectors returns every test vector implementing Table 9, addressed at
// SNI sni. Vectors are built lazily (via the closures in TestVector.Build)
// so that each repetition gets a fresh ClientHello (fresh random Random/
// key_share) and fresh connection IDs, exactly like a real handshake
// attempt would.
func buildVectors(sni string) ([]*TestVector, error) {
	var vectors []*TestVector

	// -- Section 1: QUIC Version ----------------------------------------
	for _, qv := range quicVersionVectors() {
		qv := qv // capture
		vectors = append(vectors, &TestVector{
			Name:        qv.Name,
			Category:    CategoryQUICVersion,
			QUICGroup:   qv.Group,
			QUICVersion: qv.Version,
			Build: func() ([][]byte, connectionID, connectionID, error) {
				ch, err := buildClientHello(sni)
				if err != nil {
					return nil, nil, nil, err
				}
				raw, dst, src, err := buildStandardPacket(qv.Version, qv.Salt, ch.Raw)
				if err != nil {
					return nil, nil, nil, err
				}
				return [][]byte{raw}, dst, src, nil
			},
		})
	}

	// -- Section 2: QUIC Features -----------------------------------------
	vectors = append(vectors,
		&TestVector{
			Name:     "quic_feature_fragmentation",
			Category: CategoryQUICFeature,
			Build: func() ([][]byte, connectionID, connectionID, error) {
				ch, err := buildClientHello(sni)
				if err != nil {
					return nil, nil, nil, err
				}
				raw, dst, src, err := buildFragmentedPacket(1 /* v1 */, saltV1, ch.Raw)
				if err != nil {
					return nil, nil, nil, err
				}
				return [][]byte{raw}, dst, src, nil
			},
		},
		&TestVector{
			// Not a row of Table 9/12: see buildOversizedPacket.
			Name:     "quic_feature_oversized_datagram",
			Category: CategoryQUICFeature,
			Build: func() ([][]byte, connectionID, connectionID, error) {
				ch, err := buildClientHello(sni)
				if err != nil {
					return nil, nil, nil, err
				}
				raw, dst, src, err := buildOversizedPacket(1, saltV1, ch.Raw)
				if err != nil {
					return nil, nil, nil, err
				}
				return [][]byte{raw}, dst, src, nil
			},
		},
		&TestVector{
			Name:     "quic_feature_below_1200_bytes",
			Category: CategoryQUICFeature,
			Build: func() ([][]byte, connectionID, connectionID, error) {
				ch, err := buildClientHello(sni)
				if err != nil {
					return nil, nil, nil, err
				}
				raw, dst, src, err := buildBelow1200Packet(1, saltV1, ch.Raw)
				if err != nil {
					return nil, nil, nil, err
				}
				return [][]byte{raw}, dst, src, nil
			},
		},
		&TestVector{
			Name:     "quic_feature_forced_negotiation",
			Category: CategoryQUICFeature,
			Build: func() ([][]byte, connectionID, connectionID, error) {
				ch, err := buildClientHello(sni)
				if err != nil {
					return nil, nil, nil, err
				}
				raw, dst, src, err := buildForcedNegotiationPacket(ch.Raw)
				if err != nil {
					return nil, nil, nil, err
				}
				return [][]byte{raw}, dst, src, nil
			},
		},
		&TestVector{
			Name:     "quic_feature_multi_packet_initial",
			Category: CategoryQUICFeature,
			Build: func() ([][]byte, connectionID, connectionID, error) {
				ch, err := buildClientHello(sni)
				if err != nil {
					return nil, nil, nil, err
				}
				return buildMultiPacketInitial(1, saltV1, ch.Raw)
			},
		},
	)

	// -- Section 3: TLS Length Fields --------------------------------------
	for _, spec := range tlsLengthFieldSpecs() {
		spec := spec // capture
		for _, perm := range spec.Permutations {
			perm := perm // capture
			vectors = append(vectors, &TestVector{
				Name:        fmt.Sprintf("tls_length_%s_%s", spec.FieldName, perm),
				Category:    CategoryTLSLength,
				Field:       spec.FieldName,
				Permutation: perm,
				Build: func() ([][]byte, connectionID, connectionID, error) {
					ch, err := buildClientHello(sni)
					if err != nil {
						return nil, nil, nil, err
					}
					patched := applyPermutation(ch, spec.Select(ch), perm)
					raw, dst, src, err := buildStandardPacket(1, saltV1, patched)
					if err != nil {
						return nil, nil, nil, err
					}
					return [][]byte{raw}, dst, src, nil
				},
			})
		}
	}

	return vectors, nil
}
