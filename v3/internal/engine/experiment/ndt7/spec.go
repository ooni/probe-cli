package ndt7

// This file vendors data structures from the following repositories:
//
// - github.com/m-lab/ndt7-client-go
// - github.com/m-lab/ndt-server
// - github.com/m-lab/tcp-info
//
// It is available under the Apache License v2.0.
//
// Because m-lab uses mainly Linux as a development platform, they may
// unwillingly break our Windows builds. Also, they use lots of depdencies
// that we don't actually need. Hence, vendoring FTW.
//
// The data structures are supposed to stay constant in time or to not
// change dramatically, hence this vendoring shouldn't be too bad.

type (
	// OriginKind indicates the origin of a measurement.
	OriginKind string

	// TestKind indicates the direction of a measurement.
	TestKind string
)

const (
	// OriginClient indicates that the measurement origin is the client.
	OriginClient = OriginKind("client")

	// OriginServer indicates that the measurement origin is the server.
	OriginServer = OriginKind("server")

	// TestDownload indicates that this is a download.
	TestDownload = TestKind("download")

	// TestUpload indicates that this is an upload.
	TestUpload = TestKind("upload")
)

// LinuxTCPInfo is the linux defined structure returned in RouteAttr DIAG_INFO messages.
// It corresponds to the struct tcp_info in include/uapi/linux/tcp.h
type LinuxTCPInfo struct {
	State       uint8 `csv:"TCP.State"`
	CAState     uint8 `csv:"TCP.CAState"`
	Retransmits uint8 `csv:"TCP.Retransmits"`
	Probes      uint8 `csv:"TCP.Probes"`
	Backoff     uint8 `csv:"TCP.Backoff"`
	Options     uint8 `csv:"TCP.Options"`
	WScale      uint8 `csv:"TCP.WScale"`     //snd_wscale : 4, tcpi_rcv_wscale : 4;
	AppLimited  uint8 `csv:"TCP.AppLimited"` //delivery_rate_app_limited:1;

	RTO    uint32 `csv:"TCP.RTO"` // offset 8
	ATO    uint32 `csv:"TCP.ATO"`
	SndMSS uint32 `csv:"TCP.SndMSS"`
	RcvMSS uint32 `csv:"TCP.RcvMSS"`

	Unacked uint32 `csv:"TCP.Unacked"` // offset 24
	Sacked  uint32 `csv:"TCP.Sacked"`
	Lost    uint32 `csv:"TCP.Lost"`
	Retrans uint32 `csv:"TCP.Retrans"`
	Fackets uint32 `csv:"TCP.Fackets"`

	/* Times. */
	// These seem to be elapsed time, so they increase on almost every sample.
	// We can probably use them to get more info about intervals between samples.
	LastDataSent uint32 `csv:"TCP.LastDataSent"` // offset 44
	LastAckSent  uint32 `csv:"TCP.LastAckSent"`  /* Not remembered, sorry. */ // offset 48
	LastDataRecv uint32 `csv:"TCP.LastDataRecv"` // offset 52
	LastAckRecv  uint32 `csv:"TCP.LastDataRecv"` // offset 56

	/* Metrics. */
	PMTU        uint32 `csv:"TCP.PMTU"`
	RcvSsThresh uint32 `csv:"TCP.RcvSsThresh"`
	RTT         uint32 `csv:"TCP.RTT"`
	RTTVar      uint32 `csv:"TCP.RTTVar"`
	SndSsThresh uint32 `csv:"TCP.SndSsThresh"`
	SndCwnd     uint32 `csv:"TCP.SndCwnd"`
	AdvMSS      uint32 `csv:"TCP.AdvMSS"`
	Reordering  uint32 `csv:"TCP.Reordering"`

	RcvRTT   uint32 `csv:"TCP.RcvRTT"`
	RcvSpace uint32 `csv:"TCP.RcvSpace"`

	TotalRetrans uint32 `csv:"TCP.TotalRetrans"`

	PacingRate    int64 `csv:"TCP.PacingRate"`    // This is often -1, so better for it to be signed
	MaxPacingRate int64 `csv:"TCP.MaxPacingRate"` // This is often -1, so better to be signed.

	// NOTE: In linux, these are uint64, but we make them int64 here for compatibility with BigQuery
	BytesAcked    int64 `csv:"TCP.BytesAcked"`    /* RFC4898 tcpEStatsAppHCThruOctetsAcked */
	BytesReceived int64 `csv:"TCP.BytesReceived"` /* RFC4898 tcpEStatsAppHCThruOctetsReceived */
	SegsOut       int32 `csv:"TCP.SegsOut"`       /* RFC4898 tcpEStatsPerfSegsOut */
	SegsIn        int32 `csv:"TCP.SegsIn"`        /* RFC4898 tcpEStatsPerfSegsIn */

	NotsentBytes uint32 `csv:"TCP.NotsentBytes"`
	MinRTT       uint32 `csv:"TCP.MinRTT"`
	DataSegsIn   uint32 `csv:"TCP.DataSegsIn"`  /* RFC4898 tcpEStatsDataSegsIn */
	DataSegsOut  uint32 `csv:"TCP.DataSegsOut"` /* RFC4898 tcpEStatsDataSegsOut */

	// NOTE: In linux, this is uint64, but we make it int64 here for compatibility with BigQuery
	DeliveryRate int64 `csv:"TCP.DeliveryRate"`

	BusyTime      int64 `csv:"TCP.BusyTime"`      /* Time (usec) busy sending data */
	RWndLimited   int64 `csv:"TCP.RWndLimited"`   /* Time (usec) limited by receive window */
	SndBufLimited int64 `csv:"TCP.SndBufLimited"` /* Time (usec) limited by send buffer */

	Delivered   uint32 `csv:"TCP.Delivered"`
	DeliveredCE uint32 `csv:"TCP.DeliveredCE"`

	// NOTE: In linux, these are uint64, but we make them int64 here for compatibility with BigQuery
	BytesSent    int64 `csv:"TCP.BytesSent"`    /* RFC4898 tcpEStatsPerfHCDataOctetsOut */
	BytesRetrans int64 `csv:"TCP.BytesRetrans"` /* RFC4898 tcpEStatsPerfOctetsRetrans */

	DSackDups uint32 `csv:"TCP.DSackDups"` /* RFC4898 tcpEStatsStackDSACKDups */
	ReordSeen uint32 `csv:"TCP.ReordSeen"` /* reordering events seen */
}

// AppInfo contains an application level measurement. This structure is
// described in the ndt7 specification.
type AppInfo struct {
	NumBytes    int64
	ElapsedTime int64
}

// ConnectionInfo contains connection info. This structure is described
// in the ndt7 specification.
type ConnectionInfo struct {
	Client string
	Server string
	UUID   string `json:",omitempty"`
}

// InetDiagBBRInfo implements the struct associated with INET_DIAG_BBRINFO attribute, corresponding with
// linux struct tcp_bbr_info in uapi/linux/inet_diag.h.
type InetDiagBBRInfo struct {
	BW         int64  `csv:"BBR.BW"`         // Max-filtered BW (app throughput) estimate in bytes/second
	MinRTT     uint32 `csv:"BBR.MinRTT"`     // Min-filtered RTT in uSec
	PacingGain uint32 `csv:"BBR.PacingGain"` // Pacing gain shifted left 8 bits
	CwndGain   uint32 `csv:"BBR.CwndGain"`   // Cwnd gain shifted left 8 bits
}

// The BBRInfo struct contains information measured using BBR. This structure is
// an extension to the ndt7 specification. Variables here have the same
// measurement unit that is used by the Linux kernel.
type BBRInfo struct {
	InetDiagBBRInfo
	ElapsedTime int64
}

// The TCPInfo struct contains information measured using TCP_INFO. This
// structure is described in the ndt7 specification.
type TCPInfo struct {
	LinuxTCPInfo
	ElapsedTime int64
}

// The Measurement struct contains measurement results. This message is
// an extension of the one inside of v0.9.0 of the ndt7 spec.
type Measurement struct {
	// AppInfo contains application level measurements.
	AppInfo *AppInfo `json:",omitempty"`

	// BBRInfo is the data measured using TCP BBR instrumentation.
	BBRInfo *BBRInfo `json:",omitempty"`

	// ConnectionInfo contains info on the connection.
	ConnectionInfo *ConnectionInfo `json:",omitempty"`

	// Origin indicates who performed this measurement.
	Origin OriginKind `json:",omitempty"`

	// Test contains the test name.
	Test TestKind `json:",omitempty"`

	// TCPInfo contains metrics measured using TCP_INFO instrumentation.
	TCPInfo *TCPInfo `json:",omitempty"`
}
