// Command is a server implementing the OONI remote protocol.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/remote"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/shellx"
	"github.com/songgao/water"
	"github.com/spf13/cobra"
)

func main() {
	cmd := &cobra.Command{
		Use:   "remoted",
		Short: "Linux-only daemon implementing the OONI remote protocol",
	}

	rs := &remoteServer{
		listenerFactory: nil,
		logger:          log.Log,
		outputIface:     "",
		tunDeviceAddr:   "",
		tunDeviceName:   "",
	}

	flags := cmd.Flags()
	flags.StringVar(&rs.outputIface, "output-interface", "eth0", "Interface where to emit traffic")
	flags.StringVar(&rs.tunDeviceAddr, "tun-device-address", "10.14.17.1", "Address of the TUN device")
	flags.StringVar(&rs.tunDeviceName, "tun-device-name", "miniooni0", "TUN device name")
	bindAddr := flag.String("bind", ":5555", "Address to bind")

	runtimex.Try0(flags.Parse(os.Args[1:]))

	rs.listenerFactory = &remote.TCPListenerFactory{
		Endpoint: *bindAddr,
	}

	rs.main()
}

// remoteServer implements the OONI remote protocol server. The zero value of this
// struct is invalid; please, fill all the fields marked as MANDATORY.
type remoteServer struct {
	// listenerFactory is the MANDATORY listener factory.
	listenerFactory remote.ListenerFactory

	// logger is the MANDATORY logger.
	logger model.Logger

	// outputIface is the MANDATORY output interface.
	outputIface string

	// tunDeviceAddr is the MANDATORY address to assign to the TUN device.
	tunDeviceAddr string

	// tunDeviceName is the MANDATORY TUN device name.
	tunDeviceName string
}

func (rs *remoteServer) main() error {
	// create TUN device
	log.Infof("remoted: creating TUN device: %s", rs.tunDeviceName)
	tunConfig := water.Config{
		DeviceType: water.TUN,
		PlatformSpecificParams: water.PlatformSpecificParams{
			Name: rs.tunDeviceName,
		},
	}
	tun := runtimex.Try1(water.New(tunConfig))
	defer tun.Close()

	// assign the correct IP address to the TUN device
	rs.mustAssignAddress()
	defer rs.cleanupIPTables()

	// create the listener we should use
	listener := runtimex.Try1(rs.listenerFactory.Listen())

	// listen for signals and cleanup when we receive them
	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigch
		log.Infof("remoted: interrupted by signal")
		listener.Close()
	}()

	// accept connections and route traffic
	for {
		conn, err := listener.Accept()
		if err != nil && errors.Is(err, net.ErrClosed) {
			return nil // this is how we terminate successfully
		}
		if err != nil {
			log.Warnf("remote: listener.Accept failed: %s", err.Error())
			continue
		}
		go rs.route(conn, tun)
	}
}

func (rs *remoteServer) mustAssignAddress() {
	script := []string{
		fmt.Sprintf("ip addr add %s/24 dev %s", rs.tunDeviceAddr, rs.tunDeviceName),
		fmt.Sprintf("ip link set dev %s up", rs.tunDeviceName),
		fmt.Sprintf("iptables -t nat -I POSTROUTING -o %s -j MASQUERADE", rs.outputIface),
		"sysctl net.ipv4.ip_forward=1",
	}
	for _, cmd := range script {
		runtimex.Try0(shellx.RunCommandLine(rs.logger, cmd))
	}
}

func (rs *remoteServer) cleanupIPTables() {
	_ = shellx.RunCommandLine(rs.logger, fmt.Sprintf(
		"iptables -t nat -D POSTROUTING -o %s -j MASQUERADE", rs.outputIface))
}

func (rs *remoteServer) route(clientConn net.Conn, tunDevice io.ReadWriter) {
	go func() {
		defer runtimex.CatchLogAndIgnorePanic(rs.logger, "remoted")
		for {
			ipPacket := runtimex.Try1(remote.ReadPacket(clientConn))
			_ = runtimex.Try1(tunDevice.Write(ipPacket))
		}
	}()

	go func() {
		defer runtimex.CatchLogAndIgnorePanic(rs.logger, "remoted")
		for {
			buffer := make([]byte, remote.MaxPacketSize)
			count := runtimex.Try1(tunDevice.Read(buffer))
			ipPacket := buffer[:count]
			runtimex.Try0(remote.WritePacket(clientConn, ipPacket))
		}
	}()
}
