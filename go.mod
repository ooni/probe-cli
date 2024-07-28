module github.com/ooni/probe-cli/v3

go 1.21

toolchain go1.21.10

require (
	filippo.io/age v1.1.1
	github.com/AlecAivazis/survey/v2 v2.3.7
	github.com/Psiphon-Labs/psiphon-tunnel-core v1.0.11-0.20240424194431-3612a5a6fb4c
	github.com/alecthomas/kingpin/v2 v2.4.0
	github.com/amnezia-vpn/amneziawg-go v0.2.8
	github.com/apex/log v1.9.0
	github.com/armon/go-socks5 v0.0.0-20160902184237-e75332964ef5
	github.com/cloudflare/circl v1.3.8
	github.com/cretz/bine v0.2.0
	github.com/dop251/goja v0.0.0-20231027120936-b396bb4c349d
	github.com/dop251/goja_nodejs v0.0.0-20240418154818-2aae10d4cbcf
	github.com/fatih/color v1.17.0
	github.com/google/go-cmp v0.6.0
	github.com/google/gopacket v1.1.19
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510
	github.com/google/uuid v1.6.0
	github.com/gorilla/websocket v1.5.1
	github.com/hexops/gotextdiff v1.0.3
	github.com/mattn/go-colorable v0.1.13
	github.com/miekg/dns v1.1.59
	github.com/mitchellh/go-wordwrap v1.0.1
	github.com/montanaflynn/stats v0.7.1
	github.com/ooni/minivpn v0.0.6
	github.com/ooni/netem v0.0.0-20240208095707-608dcbcd82b8
	github.com/ooni/oocrypto v0.6.1
	github.com/ooni/oohttp v0.7.2
	github.com/ooni/probe-assets v0.23.0
	github.com/pborman/getopt/v2 v2.1.0
	github.com/pion/stun v0.6.1
	github.com/pkg/errors v0.9.1
	github.com/quic-go/quic-go v0.43.1
	github.com/rogpeppe/go-internal v1.12.0
	github.com/rubenv/sql-migrate v1.6.1
	github.com/schollz/progressbar/v3 v3.14.2
	github.com/upper/db/v4 v4.7.0
	gitlab.com/yawning/obfs4.git v0.0.0-20231012084234-c3e2d44b1033
	gitlab.com/yawning/utls.git v0.0.12-1
	gitlab.torproject.org/tpo/anti-censorship/pluggable-transports/goptlib v1.5.0
	gitlab.torproject.org/tpo/anti-censorship/pluggable-transports/snowflake/v2 v2.6.1
	golang.org/x/crypto v0.23.0
	golang.org/x/net v0.25.0
	golang.org/x/sys v0.20.0
)

require (
	filippo.io/bigmod v0.0.3 // indirect
	filippo.io/keygen v0.0.0-20230306160926-5201437acf8e // indirect
	github.com/Psiphon-Labs/bolt v0.0.0-20200624191537-23cedaef7ad7 // indirect
	github.com/Psiphon-Labs/goptlib v0.0.0-20200406165125-c0e32a7a3464 // indirect
	github.com/Psiphon-Labs/psiphon-tls v0.0.0-20240424193802-52b2602ec60c // indirect
	github.com/Psiphon-Labs/quic-go v0.0.0-20240424181006-45545f5e1536 // indirect
	github.com/andybalholm/brotli v1.0.6 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dlclark/regexp2 v1.10.0 // indirect
	github.com/flynn/noise v1.0.1 // indirect
	github.com/gaukas/godicttls v0.0.4 // indirect
	github.com/go-sourcemap/sourcemap v2.1.3+incompatible // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/google/btree v1.1.2 // indirect
	github.com/google/pprof v0.0.0-20240509144519-723abb6459b7 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/klauspost/compress v1.17.8 // indirect
	github.com/libp2p/go-reuseport v0.4.0 // indirect
	github.com/mitchellh/colorstring v0.0.0-20190213212951-d06e56a500db // indirect
	github.com/mroth/weightedrand v1.0.0 // indirect
	github.com/onsi/ginkgo/v2 v2.17.3 // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/pion/transport/v2 v2.2.5 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/quic-go/qpack v0.4.0 // indirect
	github.com/refraction-networking/conjure v0.7.11-0.20240130155008-c8df96195ab2 // indirect
	github.com/refraction-networking/ed25519 v0.1.2 // indirect
	github.com/refraction-networking/obfs4 v0.1.2 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/segmentio/fasthash v1.0.3 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/stretchr/testify v1.9.0 // indirect
	github.com/tevino/abool/v2 v2.1.0 // indirect
	github.com/xhit/go-str2duration/v2 v2.1.0 // indirect
	gitlab.com/yawning/edwards25519-extra v0.0.0-20231005122941-2149dcafc266 // indirect
	go.uber.org/mock v0.4.0 // indirect
	golang.org/x/exp v0.0.0-20240506185415-9bf2ced13842 // indirect
	golang.org/x/exp/typeparams v0.0.0-20230522175609-2e198f4a06a1 // indirect
	golang.org/x/sync v0.7.0 // indirect
	golang.org/x/time v0.5.0 // indirect
	golang.zx2c4.com/wintun v0.0.0-20230126152724-0fa3db229ce2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	gvisor.dev/gvisor v0.0.0-20230927004350-cbd86285d259 // indirect
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/AndreasBriese/bbloom v0.0.0-20190825152654-46b345b51c96 // indirect
	github.com/alecthomas/units v0.0.0-20231202071711-9a357b53e9c9 // indirect
	github.com/armon/go-proxyproto v0.1.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bifurcation/mint v0.0.0-20180306135233-198357931e61 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cheekybits/genny v1.0.0 // indirect
	github.com/cognusion/go-cache-lru v0.0.0-20170419142635-f73e2280ecea // indirect
	github.com/dchest/siphash v1.2.3 // indirect
	github.com/dgraph-io/badger v1.6.2 // indirect
	github.com/dgraph-io/ristretto v0.1.1 // indirect
	github.com/dsnet/compress v0.0.1 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/go-gorp/gorp/v3 v3.1.0 // indirect
	github.com/golang/glog v1.2.1 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/grafov/m3u8 v0.12.0 // indirect
	github.com/hashicorp/golang-lru v1.0.2 // indirect
	github.com/juju/ratelimit v1.0.2 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/klauspost/cpuid/v2 v2.2.6 // indirect
	github.com/klauspost/reedsolomon v1.11.8 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-sqlite3 v1.14.22
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	github.com/oschwald/maxminddb-golang v1.12.0
	github.com/pion/datachannel v1.5.6 // indirect
	github.com/pion/dtls/v2 v2.2.11 // indirect
	github.com/pion/ice/v2 v2.3.24 // indirect
	github.com/pion/interceptor v0.1.29 // indirect
	github.com/pion/logging v0.2.2 // indirect
	github.com/pion/mdns v0.0.12 // indirect
	github.com/pion/randutil v0.1.0 // indirect
	github.com/pion/rtcp v1.2.14 // indirect
	github.com/pion/rtp v1.8.6 // indirect
	github.com/pion/sctp v1.8.16 // indirect
	github.com/pion/sdp/v3 v3.0.9 // indirect
	github.com/pion/srtp/v2 v2.0.18 // indirect
	github.com/pion/turn/v2 v2.1.6 // indirect
	github.com/pion/webrtc/v3 v3.2.40 // indirect
	github.com/prometheus/client_golang v1.19.1
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.53.0 // indirect
	github.com/prometheus/procfs v0.14.0 // indirect
	github.com/refraction-networking/gotapdance v1.7.10 // indirect
	github.com/refraction-networking/utls v1.3.3 // indirect
	github.com/sergeyfrolov/bsbuffer v0.0.0-20180903213811-94e85abb8507 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/spf13/cobra v1.8.0
	github.com/syndtr/gocapability v0.0.0-20200815063812-42c35b437635 // indirect
	github.com/tailscale/hujson v0.0.0-20221223112325-20486734a56a
	github.com/templexxx/cpu v0.1.0 // indirect
	github.com/templexxx/xorsimd v0.4.2 // indirect
	github.com/tjfoc/gmsm v1.4.1 // indirect
	github.com/wader/filtertransport v0.0.0-20200316221534-bdd9e61eee78 // indirect
	github.com/xtaci/kcp-go/v5 v5.6.2 // indirect
	github.com/xtaci/smux v1.5.24 // indirect
	gitlab.com/yawning/bsaes.git v0.0.0-20190805113838-0a714cd429ec // indirect
	golang.org/x/mod v0.17.0 // indirect
	golang.org/x/term v0.20.0 // indirect
	golang.org/x/text v0.15.0 // indirect
	golang.org/x/tools v0.21.0 // indirect
	google.golang.org/protobuf v1.34.1 // indirect
)
