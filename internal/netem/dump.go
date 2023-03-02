package netem

//
// Code to dump packets
//

import (
	"os"
	"sync"
	"time"

	"github.com/apex/log"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcapgo"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// PCAPDumper is a [DPIStack] but also an open PCAP file. Remember
// to call Close when done to flush the PCAP file. The zero
// value is invalid; use [NewPCAPDumper] to instantiate.
type PCAPDumper struct {
	// closeOnce provides "once" semantics for close.
	closeOnce sync.Once

	// filep is the file where we're writing.
	filep *os.File

	// snaplen is the snapshot length
	snaplen uint32

	// w is the PCAP writer
	w *pcapgo.Writer

	// DPIStack is the wrapped stack
	DPIStack
}

// NewPCAPDumper wraps an existing [DPIStack], intercepts the packets read
// and written, and stores them into the given PCAP file. This function
// calls [runtimex.PanicOnError] in case of failure.
func NewPCAPDumper(filename string, snaplen uint32, stack DPIStack) *PCAPDumper {
	filep := runtimex.Try1(os.Create(filename))
	w := pcapgo.NewWriter(filep)
	runtimex.Try0(w.WriteFileHeader(snaplen, layers.LinkTypeIPv4))
	pd := &PCAPDumper{
		closeOnce: sync.Once{},
		filep:     filep,
		snaplen:   snaplen,
		w:         w,
		DPIStack:  stack,
	}
	return pd
}

// ReadPacket implements DPIStack
func (pd *PCAPDumper) ReadPacket() ([]byte, error) {
	// read the packet from the stack
	packet, err := pd.DPIStack.ReadPacket()
	if err != nil {
		return nil, err
	}

	// write into the PCAP
	pd.writePCAP(packet)

	// provide it to the caller
	return packet, nil
}

// writePCAP writes the given packet into the PCAP file.
func (pd *PCAPDumper) writePCAP(packet []byte) {

	// make sure the capture length makes sense
	captureLen := pd.snaplen
	if lp := uint32(len(packet)); captureLen > lp {
		captureLen = lp
	}

	// write the packet into the PCAP
	ci := gopacket.CaptureInfo{
		Timestamp:      time.Now(),
		CaptureLength:  int(captureLen),
		Length:         len(packet),
		InterfaceIndex: 0,
		AncillaryData:  []interface{}{},
	}
	if err := pd.w.WritePacket(ci, packet[:captureLen]); err != nil {
		log.Warnf("netem: PCAPDumper.WritePacket: %s", err.Error())
	}
}

// WritePacket implements DPIStack
func (pd *PCAPDumper) WritePacket(packet []byte) error {
	// write into the PCAP
	pd.writePCAP(packet)

	// provide packet to the stack
	return pd.DPIStack.WritePacket(packet)
}

// Close implements DPIStack
func (pd *PCAPDumper) Close() error {
	pd.closeOnce.Do(func() {
		pd.DPIStack.Close()
		runtimex.Try0(pd.filep.Close()) // fatal if we cannot close file
	})
	return nil
}
