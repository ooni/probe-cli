package netem

//
// Code to dump packets
//

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/apex/log"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcapgo"
)

// PCAPDumper is a [DPIStack] but also an open PCAP file. The zero
// value is invalid; use [NewPCAPDumper] to instantiate.
type PCAPDumper struct {
	// cancel stops the background goroutines.
	cancel context.CancelFunc

	// closeOnce provides "once" semantics for close.
	closeOnce sync.Once

	// joined is closed when the background goroutine has terminated
	joined chan any

	// pic is the channel where we post packets to capture
	pic chan *pcapDumperPacketInfo

	// DPIStack is the wrapped stack
	DPIStack
}

// pcapDumperPacketInfo contains info about a packet.
type pcapDumperPacketInfo struct {
	originalLength int
	snapshot       []byte
}

// NewPCAPDumper wraps an existing [DPIStack], intercepts the packets read
// and written, and stores them into the given PCAP file. This function
// creates a background goroutine for writing into the PCAP file. To join
// the goroutine, call [PCAPDumper.Close].
func NewPCAPDumper(filename string, stack DPIStack) *PCAPDumper {
	const manyPackets = 4096
	ctx, cancel := context.WithCancel(context.Background())
	pd := &PCAPDumper{
		cancel:    cancel,
		closeOnce: sync.Once{},
		joined:    make(chan any),
		pic:       make(chan *pcapDumperPacketInfo, manyPackets),
		DPIStack:  stack,
	}
	go pd.loop(ctx, filename)
	return pd
}

// ReadPacket implements DPIStack
func (pd *PCAPDumper) ReadPacket() ([]byte, error) {
	// read the packet from the stack
	packet, err := pd.DPIStack.ReadPacket()
	if err != nil {
		return nil, err
	}

	// send packet information to the background writer
	pd.deliverPacketInfo(packet)

	// provide it to the caller
	return packet, nil
}

// deliverPacketInfo delivers packet info to the background writer.
func (pd *PCAPDumper) deliverPacketInfo(packet []byte) {
	// make sure the capture length makes sense
	packetLength := len(packet)
	captureLength := 256
	if packetLength < captureLength {
		captureLength = packetLength
	}

	// actually deliver the packet info
	pinfo := &pcapDumperPacketInfo{
		originalLength: len(packet),
		snapshot:       append([]byte{}, packet[:captureLength]...), // duplicate
	}
	select {
	case pd.pic <- pinfo:
	default:
		// just drop from the capture
	}
}

// loop is the loop that writes pcaps
func (pd *PCAPDumper) loop(ctx context.Context, filename string) {
	// synchronize with parent
	defer close(pd.joined)

	// open the file where to create the pcap
	filep, err := os.Create(filename)
	if err != nil {
		log.Warnf("netem: PCAPDumper: os.Create: %s", err.Error())
		return
	}
	defer func() {
		if err := filep.Close(); err != nil {
			log.Warnf("netem: PCAPDumper: filep.Close: %s", err.Error())
			// fallthrough
		}
	}()

	// write the PCAP header
	w := pcapgo.NewWriter(filep)
	const largeSnapLen = 262144
	if err := w.WriteFileHeader(largeSnapLen, layers.LinkTypeIPv4); err != nil {
		log.Warnf("netem: PCAPDumper: os.Create: %s", err.Error())
		return
	}

	// loop until we're done and write each entry
	for {
		select {
		case <-ctx.Done():
			return
		case pinfo := <-pd.pic:
			pd.doWritePCAPEntry(pinfo, w)
		}
	}
}

// doWritePCAPEntry writes the given packet entry into the PCAP file.
func (pd *PCAPDumper) doWritePCAPEntry(pinfo *pcapDumperPacketInfo, w *pcapgo.Writer) {
	ci := gopacket.CaptureInfo{
		Timestamp:      time.Now(),
		CaptureLength:  len(pinfo.snapshot),
		Length:         pinfo.originalLength,
		InterfaceIndex: 0,
		AncillaryData:  []interface{}{},
	}
	if err := w.WritePacket(ci, pinfo.snapshot); err != nil {
		log.Warnf("netem: w.WritePacket: %s", err.Error())
		// fallthrough
	}
}

// WritePacket implements DPIStack
func (pd *PCAPDumper) WritePacket(packet []byte) error {
	// send packet information to the background writer
	pd.deliverPacketInfo(packet)

	// provide packet to the stack
	return pd.DPIStack.WritePacket(packet)
}

// Close implements DPIStack
func (pd *PCAPDumper) Close() error {
	pd.closeOnce.Do(func() {
		// notify the underlying stack to stop
		pd.DPIStack.Close()

		// notify the background goroutine to terminate
		pd.cancel()

		// wait until the channel is drained
		log.Infof("netem: PCAPDumper: awaiting for background writer to finish writing")
		<-pd.joined
	})
	return nil
}
