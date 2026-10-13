package main

//
// Reading captures with gopacket: Ethernet captures, as pasta --pcap writes them, in pcap or
// pcapng files, over IPv4 or IPv6. Ethernet padding is not payload (gopacket trims an IP packet
// to its length; a length of 0, from segmentation offload, keeps the whole frame).
//

import (
	"bufio"
	"bytes"
	"cmp"
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcapgo"
)

// packet is one captured IP packet carrying TCP or UDP.
type packet struct {
	time     time.Time
	src, dst string // IP addresses
	tcp      *layers.TCP
	udp      *layers.UDP
}

// capture is what readCapture read: every frame's count and time span, and the TCP and UDP
// packets in capture order.
type capture struct {
	frames      int
	first, last time.Time
	packets     []packet
}

// captureReader is what pcapgo's pcap and pcapng readers have in common.
type captureReader interface {
	gopacket.PacketDataSource
	LinkType() layers.LinkType
}

// readCapture reads a pcap or pcapng file of Ethernet frames.
func readCapture(path string) (*capture, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := bufio.NewReader(f)
	magic, err := r.Peek(4)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	var source captureReader
	if bytes.Equal(magic, []byte{0x0a, 0x0d, 0x0d, 0x0a}) {
		source, err = pcapgo.NewNgReader(r, pcapgo.DefaultNgReaderOptions) // fails on mixed link types
	} else {
		source, err = pcapgo.NewReader(r)
	}
	if err != nil {
		return nil, fmt.Errorf("%s: not a pcap or pcapng file: %w", path, err)
	}
	if source.LinkType() != layers.LinkTypeEthernet {
		return nil, fmt.Errorf("%s: not an Ethernet capture; capture with pasta --pcap", path)
	}

	c := &capture{}
	for {
		frame, ci, err := source.ReadPacketData()
		if err == io.EOF {
			return c, nil
		}
		if err == io.ErrUnexpectedEOF { // pasta stopped mid-write
			fmt.Fprintf(os.Stderr, "hellotool: %s ends in a partial frame; read the %d frames before it\n", path, c.frames)
			return c, nil
		}
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		if c.frames == 0 {
			c.first = ci.Timestamp
		}
		c.frames, c.last = c.frames+1, ci.Timestamp
		if p, ok := decodeFrame(frame); ok {
			p.time = ci.Timestamp
			c.packets = append(c.packets, p)
		}
	}
}

// decodeFrame decodes an Ethernet frame down to its TCP or UDP layer. ok is false for anything else.
func decodeFrame(frame []byte) (p packet, ok bool) {
	pkt := gopacket.NewPacket(frame, layers.LayerTypeEthernet, gopacket.NoCopy)
	switch ip := pkt.NetworkLayer().(type) {
	case *layers.IPv4:
		p.src, p.dst = ip.SrcIP.String(), ip.DstIP.String()
	case *layers.IPv6:
		p.src, p.dst = ip.SrcIP.String(), ip.DstIP.String()
	default:
		return p, false
	}
	switch l4 := pkt.TransportLayer().(type) {
	case *layers.TCP:
		p.tcp = l4
	case *layers.UDP:
		p.udp = l4
	default:
		return p, false
	}
	return p, true
}

// segment is a TCP payload at an offset from its stream's start.
type segment struct {
	time    time.Time
	payload []byte
}

// halfStream reassembles one direction of a TCP connection. Offsets count from the stream's
// start: after the SYN, or the earliest data captured when the SYN was not; sequence numbers may
// wrap past 2^32. When segments overlap, the first copy captured wins.
type halfStream struct {
	start, first uint32 // the SYN's sequence number + 1; the first data segment's
	haveSYN      bool
	segments     map[uint32]segment // by sequence number
}

func newHalfStream() *halfStream {
	return &halfStream{segments: map[uint32]segment{}}
}

// startAt sets the stream's start: the sequence number after a SYN.
func (h *halfStream) startAt(seq uint32) {
	h.start, h.haveSYN = seq, true
}

// add records a data segment.
func (h *halfStream) add(seq uint32, payload []byte, t time.Time) {
	if len(h.segments) == 0 {
		h.first = seq
	}
	if _, ok := h.segments[seq]; !ok {
		h.segments[seq] = segment{time: t, payload: append([]byte(nil), payload...)}
	}
}

// timedOffset says when the bytes from offset on arrived.
type timedOffset struct {
	offset int
	time   time.Time
}

// bytes returns the stream's contiguous bytes from its start, and when each segment's new bytes
// began. It stops at the first gap.
func (h *halfStream) bytes() ([]byte, []timedOffset) {
	start := h.start
	if !h.haveSYN { // the segment furthest before the first one captured
		start = h.first
		for seq := range h.segments {
			if int32(seq-start) < 0 {
				start = seq
			}
		}
	}
	seqs := slices.Collect(maps.Keys(h.segments))
	slices.SortFunc(seqs, func(a, b uint32) int { return cmp.Compare(a-start, b-start) }) // wraps as TCP does
	var data []byte
	var times []timedOffset
	next := uint32(0)
	for _, seq := range seqs {
		s, o := h.segments[seq], seq-start
		end := o + uint32(len(s.payload))
		if o <= next && next < end {
			times = append(times, timedOffset{offset: len(data), time: s.time})
			data = append(data, s.payload[next-o:]...)
			next = end
		}
	}
	return data, times
}

// tcpConnection is both directions of one connection, client first.
type tcpConnection struct {
	client, server         string
	clientPort, serverPort uint16
	first, last            time.Time
	up, down               *halfStream // client to server, server to client
	closed                 *closing
}

// closing is a connection's first FIN or RST.
type closing struct {
	time     time.Time
	byClient bool
	reset    bool
}

// endpoints identify one direction of a TCP connection.
type endpoints struct {
	src, dst         string
	srcPort, dstPort uint16
}

func (e endpoints) reversed() endpoints { return endpoints{e.dst, e.src, e.dstPort, e.srcPort} }

// tcpConnections groups TCP packets into connections, in the order they were first seen. The
// client is the side that sent the SYN; without one, the side talking to port 443 or 80, else the
// side with the higher port. A SYN on endpoints whose connection already carried data or closed
// starts a new connection.
func tcpConnections(packets []packet) []*tcpConnection {
	current := map[endpoints]*tcpConnection{} // by client to server
	var order []*tcpConnection
	for _, p := range packets {
		if p.tcp == nil {
			continue
		}
		key := endpoints{p.src, p.dst, uint16(p.tcp.SrcPort), uint16(p.tcp.DstPort)}
		syn := p.tcp.SYN && !p.tcp.ACK
		conn, fromClient := current[key], true
		if conn == nil {
			conn, fromClient = current[key.reversed()], false
		}
		if syn && conn != nil && (len(conn.up.segments) > 0 || len(conn.down.segments) > 0 || conn.closed != nil) {
			delete(current, conn.endpoints())
			conn = nil
		}
		if conn == nil {
			web := func(port uint16) bool { return port == 443 || port == 80 }
			fromClient = syn || (!(p.tcp.SYN && p.tcp.ACK) && (web(key.dstPort) || (!web(key.srcPort) && key.dstPort < key.srcPort)))
			client := key
			if !fromClient {
				client = key.reversed()
			}
			conn = &tcpConnection{client: client.src, server: client.dst, clientPort: client.srcPort,
				serverPort: client.dstPort, first: p.time, up: newHalfStream(), down: newHalfStream()}
			current[client] = conn
			order = append(order, conn)
		}
		conn.last = p.time
		half := conn.down
		if fromClient {
			half = conn.up
		}
		if p.tcp.SYN {
			half.startAt(p.tcp.Seq + 1)
		}
		if (p.tcp.FIN || p.tcp.RST) && conn.closed == nil {
			conn.closed = &closing{time: p.time, byClient: fromClient, reset: p.tcp.RST}
		}
		if len(p.tcp.Payload) > 0 {
			half.add(p.tcp.Seq, p.tcp.Payload, p.time)
		}
	}
	return order
}

func (c *tcpConnection) endpoints() endpoints {
	return endpoints{c.client, c.server, c.clientPort, c.serverPort}
}
