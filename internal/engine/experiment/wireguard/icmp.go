package wireguard

import (
	"context"
	"log"
	"math"
	"sync"
	"time"

	"github.com/ooni/minivpn/extras/ping"
	"golang.zx2c4.com/wireguard/tun"
	"golang.zx2c4.com/wireguard/tun/netstack"
)

// PingReply is a single response in the ping sequence.
type PingReply struct {
	Seq int     `json:"seq"`
	Rtt float64 `json:"rtt"`
}

// PingResult holds the results for a pinger run.
type PingResult struct {
	Target      string      `json:"target"`
	Sequence    []PingReply `json:"sequence"`
	PacketsRecv int         `json:"pkt_rcv"`
	PacketsSent int         `json:"pkt_snt"`
	MinRtt      float64     `json:"min_rtt"`
	MaxRtt      float64     `json:"max_rtt"`
	AvgRtt      float64     `json:"avg_rtt"`
	StdRtt      float64     `json:"std_rtt"`
	Failure     *string     `json:"failure"`
}

// pingTimeout returns the timeout set on each pinger train.
func pingTimeout(count int) time.Duration {
	return 2*time.Second*time.Duration(count) + pingExtraWaitSeconds
}

func doSinglePing(wg *sync.WaitGroup, tun tun.Device, tnet *netstack.Net, target string, count int, tk *TestKeys) {
	defer wg.Done()
	pingConn, err := tnet.Dial("ping", target)
	if err != nil {
		e := err.Error()
		tk.Failure = &e
		return
	}

	pinger := ping.NewFromSharedConnection(target, pingConn)
	pinger.Count = count
	pinger.Timeout = pingTimeout(count)
	err = pinger.Run(context.Background())

	pingResult := parseStats(pinger, target)
	if err != nil {
		e := err.Error()
		pingResult.Failure = &e
	}
	tk.Pings = append(tk.Pings, pingResult)

}

func sendBlockingPing(wg *sync.WaitGroup, tun tun.Device, tnet *netstack.Net, target string, count int, tk *TestKeys) {
	wg.Add(1)
	go doSinglePing(wg, tun, tnet, target, count, tk)
	wg.Wait()
	log.Printf("ping train sent to %s ----", target)
}

// TODO(ainghazal): factor out with code in openvpn.
// parseStats accepts a pointer to a Pinger struct and a target string, and returns
// an pointer to a PingResult with all the fields filled.
func parseStats(pinger *ping.Pinger, target string) *PingResult {
	st := pinger.Statistics()
	replies := []PingReply{}
	for _, r := range st.Replies {
		replies = append(replies, PingReply{
			Seq: r.Seq,
			Rtt: toMs(r.Rtt),
		})
	}
	pingStats := &PingResult{
		Target:      target,
		PacketsRecv: st.PacketsRecv,
		PacketsSent: st.PacketsSent,
		Sequence:    replies,
		MinRtt:      toMs(st.MinRtt),
		MaxRtt:      toMs(st.MaxRtt),
		AvgRtt:      toMs(st.AvgRtt),
		StdRtt:      toMs(st.StdDevRtt),
	}
	return pingStats
}

// toMs converts time.Duration to a float64 number representing milliseconds
// with fixed precision (3 decimal places).
func toMs(t time.Duration) float64 {
	return math.Round(t.Seconds()*1e6) / 1e3
}
