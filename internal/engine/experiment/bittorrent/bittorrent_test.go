package bittorrent

import (
	"context"
	"errors"
	"net"
	"net/url"
	"testing"

	"github.com/anacrolix/dht/v2"
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/mockable"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func SeedFile(file string) (*torrent.Client, *string, []*string, error) {
	// Initialize torrent client
	conf := torrent.NewDefaultClientConfig()
	conf.LookupTrackerIp = func(u *url.URL) ([]net.IP, error) {
		return []net.IP{}, errors.New("The client should not be looking up any trackers")
	}
	// Don't boostrap the DHT, but don't disable it
	conf.DhtStartingNodes = func(network string) dht.StartingNodesGetter {
		return func() (addrs []dht.Addr, err error) {
			return []dht.Addr{}, nil
		}
	}
	// Only listen on localhost
	conf.ListenHost = func(network string) string {
		return "localhost"
	}
	conf.Seed = true
	conf.DisableWebtorrent = true
	conf.DisableWebseeds = true

	// Try disable DHT security on torrent's DHT servers
	conf.ConfigureAnacrolixDhtServer = func (dht *dht.ServerConfig) {
		dht.NoSecurity = true
	}

	client, err := torrent.NewClient(conf)
	if err != nil {
		println("Failed client")
		return nil, nil, nil, err
	}

	magnet, mi, err := MagnetFromFile(file)

	// Start seeding torrent
	_, err = client.AddTorrent(mi)
	if err != nil {
		println("Failed seed")
		return nil, nil, nil, err
	}

	println("Starting to seed")

	magnetStr := magnet.String()
	addrsStr := []*string{}
	for _, addr := range client.ListenAddrs() {
		// Add peer address naively
		addrStr := addr.String()
		magnetStr = magnetStr + "&x.pe=" + addrStr
		addrsStr = append(addrsStr, &addrStr)
	}

	println("Complete magnet: " + magnetStr)

	return client, &magnetStr, addrsStr, nil
}

func MagnetFromFile(file string) (*metainfo.Magnet, *metainfo.MetaInfo, error) {
	// Create torrent from repo Readme.md
	mi := metainfo.MetaInfo{}
	mi.SetDefaults()
	// Make sure we don't announce
	mi.AnnounceList = make([][]string, 0)
	info := metainfo.Info{}
	err := info.BuildFromFilePath(file)
	if err != nil {
		println("Failed infobuild")
		return nil, nil, err
	}
	mi.InfoBytes, err = bencode.Marshal(info)
	if err != nil {
		println("Failed bencode")
		return nil, nil, err
	}

	magnet := mi.Magnet(nil, &info)
	return &magnet, &mi, nil

}

func TestMeasurer_run(t *testing.T) {
	// runHelper is an helper function to run this set of tests.
	runHelper := func(input string, bootstrapNodes []*string) (*model.Measurement, model.ExperimentMeasurer, error) {
		measurer := NewExperimentMeasurer(Config{
			BootstrapNodes: bootstrapNodes,
		})
		ctx := context.Background()
		measurement := &model.Measurement{
			Input: model.MeasurementTarget(input),
		}
		session := &mockable.Session{
			//MockableLogger: model.DiscardLogger,
			// Display run log inside test for the moment
			MockableLogger: log.Log,
		}

		args := &model.ExperimentArgs{
			Callbacks:   model.NewPrinterCallbacks(model.DiscardLogger),
			Measurement: measurement,
			Session:     session,
		}

		err := measurer.Run(ctx, args)
		return measurement, measurer, err
	}

	t.Run("with empty input", func(t *testing.T) {
		_, _, err := runHelper("", []*string{})
		if !errors.Is(err, errNoInputProvided) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("with invalid URL", func(t *testing.T) {
		_, _, err := runHelper("\t", []*string{})
		if !errors.Is(err, errInputIsNotAnURL) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("with invalid scheme", func(t *testing.T) {
		_, _, err := runHelper("https://8.8.8.8:443/", []*string{})
		if !errors.Is(err, errInvalidScheme) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("with local peer", func(t *testing.T) {
		// Tests are run from test dir
		server, magnet, addrs, err := SeedFile("../../../../Readme.md")
		//server, magnet, err := SeedFile("/home/baraka/Downloads/pop-os_22.04_amd64_intel_16.iso")
		if err != nil {
			t.Fatal(err)
		}
		defer server.Close()

		meas, _, err := runHelper(*magnet, addrs)
		if err != nil {
			t.Fatal("failed to fetch", err)
		}

		tk := meas.TestKeys.(*TestKeys)
		if tk.Failure != "" {
			t.Fatal(tk.Failure)
		}

		t.Fatal("dummy")
	})
}
