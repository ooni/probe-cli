module github.com/ooni/probe-cli/v3

go 1.22.0

toolchain go1.22.2

require (
	filippo.io/age v1.2.0
	github.com/AlecAivazis/survey/v2 v2.3.7
	github.com/Psiphon-Labs/psiphon-tunnel-core v1.0.11-0.20241119203223-a6c5cfa67fd9
	github.com/alecthomas/kingpin/v2 v2.4.0
	github.com/apex/log v1.9.0
	github.com/armon/go-socks5 v0.0.0-20160902184237-e75332964ef5
	github.com/cloudflare/circl v1.5.0
	github.com/cretz/bine v0.2.0
	github.com/dop251/goja v0.0.0-20241024094426-79f3a7efcdbd
	github.com/dop251/goja_nodejs v0.0.0-20240728170619-29b559befffc
	github.com/fatih/color v1.18.0
	github.com/google/go-cmp v0.6.0
	github.com/google/gopacket v1.1.19
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510
	github.com/google/uuid v1.6.0
	github.com/gorilla/websocket v1.5.3
	github.com/hexops/gotextdiff v1.0.3
	github.com/mattn/go-colorable v0.1.13
	github.com/miekg/dns v1.1.62
	github.com/mitchellh/go-wordwrap v1.0.1
	github.com/montanaflynn/stats v0.7.1
	github.com/ooni/minivpn v0.0.7
	github.com/ooni/netem v0.0.0-20240208095707-608dcbcd82b8
	github.com/ooni/oocrypto v0.7.0
	github.com/ooni/oohttp v0.8.0
	github.com/ooni/probe-assets v0.24.0
	github.com/pborman/getopt/v2 v2.1.0
	github.com/pion/stun v0.6.1
	github.com/pkg/errors v0.9.1
	github.com/quic-go/quic-go v0.43.1
	github.com/rogpeppe/go-internal v1.13.1
	github.com/rubenv/sql-migrate v1.7.0
	github.com/schollz/progressbar/v3 v3.17.1
	github.com/upper/db/v4 v4.9.0
	gitlab.com/yawning/obfs4.git v0.0.0-20231012084234-c3e2d44b1033
	gitlab.com/yawning/utls.git v0.0.12-1
	gitlab.torproject.org/tpo/anti-censorship/pluggable-transports/goptlib v1.6.0
	gitlab.torproject.org/tpo/anti-censorship/pluggable-transports/snowflake/v2 v2.10.1
	golang.org/x/crypto v0.29.0
	golang.org/x/net v0.31.0
	golang.org/x/sys v0.27.0
)

require (
	filippo.io/bigmod v0.0.3 // indirect
	filippo.io/keygen v0.0.0-20240718133620-7f162efbbd87 // indirect
	github.com/Psiphon-Labs/bolt v0.0.0-20200624191537-23cedaef7ad7 // indirect
	github.com/Psiphon-Labs/consistent v0.0.0-20240322131436-20aaa4e05737 // indirect
	github.com/Psiphon-Labs/goptlib v0.0.0-20200406165125-c0e32a7a3464 // indirect
	github.com/Psiphon-Labs/psiphon-tls v0.0.0-20240824224428-ca6969e315a9 // indirect
	github.com/Psiphon-Labs/quic-go v0.0.0-20240821052333-b6316b594e39 // indirect
	github.com/Psiphon-Labs/utls v1.1.1-0.20241107183331-b18909f8ccaa // indirect
	github.com/alexbrainman/sspi v0.0.0-20231016080023-1a75b4708caa // indirect
	github.com/andybalholm/brotli v1.1.1 // indirect
	github.com/aws/aws-sdk-go-v2 v1.32.2 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.28.0 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.17.41 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.17 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.21 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.21 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.12.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.12.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/sqs v1.36.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.24.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.28.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.32.2 // indirect
	github.com/aws/smithy-go v1.22.0 // indirect
	github.com/bits-and-blooms/bitset v1.16.0 // indirect
	github.com/bits-and-blooms/bloom/v3 v3.7.0 // indirect
	github.com/cespare/xxhash v1.1.0 // indirect
	github.com/coreos/go-iptables v0.8.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dblohm7/wingoes v0.0.0-20240820181039-f2b84150679e // indirect
	github.com/dlclark/regexp2 v1.11.4 // indirect
	github.com/flynn/noise v1.1.0 // indirect
	github.com/fxamacker/cbor/v2 v2.7.0 // indirect
	github.com/go-sourcemap/sourcemap v2.1.4+incompatible // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/mock v1.6.0 // indirect
	github.com/google/btree v1.1.3 // indirect
	github.com/google/nftables v0.2.0 // indirect
	github.com/google/pprof v0.0.0-20241122213907-cbe949e5a41b // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/josharian/native v1.1.1-0.20230202152459-5c7d0dd6ab86 // indirect
	github.com/jsimonetti/rtnetlink v1.4.2 // indirect
	github.com/klauspost/compress v1.17.11 // indirect
	github.com/libp2p/go-reuseport v0.4.0 // indirect
	github.com/mdlayher/netlink v1.7.2 // indirect
	github.com/mdlayher/socket v0.5.1 // indirect
	github.com/mitchellh/colorstring v0.0.0-20190213212951-d06e56a500db // indirect
	github.com/mroth/weightedrand v1.0.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/onsi/ginkgo/v2 v2.22.0 // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/pion/dtls/v3 v3.0.1 // indirect
	github.com/pion/stun/v3 v3.0.0 // indirect
	github.com/pion/transport/v2 v2.2.10 // indirect
	github.com/pion/transport/v3 v3.0.7 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/quic-go/qpack v0.4.0 // indirect
	github.com/realclientip/realclientip-go v1.0.0 // indirect
	github.com/refraction-networking/conjure v0.7.11-0.20240130155008-c8df96195ab2 // indirect
	github.com/refraction-networking/ed25519 v0.1.2 // indirect
	github.com/refraction-networking/obfs4 v0.1.2 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/segmentio/fasthash v1.0.3 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/stretchr/testify v1.10.0 // indirect
	github.com/tailscale/goupnp v1.0.1-0.20210804011211-c64d0f06ea05 // indirect
	github.com/tailscale/netlink v1.1.1-0.20211101221916-cabfb018fe85 // indirect
	github.com/txthinking/runnergroup v0.0.0-20210608031112-152c7c4432bf // indirect
	github.com/txthinking/socks5 v0.0.0-20230325130024-4230056ae301 // indirect
	github.com/vishvananda/netlink v1.3.0 // indirect
	github.com/vishvananda/netns v0.0.5 // indirect
	github.com/wlynxg/anet v0.0.5 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	github.com/xhit/go-str2duration/v2 v2.1.0 // indirect
	gitlab.com/yawning/edwards25519-extra v0.0.0-20231005122941-2149dcafc266 // indirect
	gitlab.torproject.org/tpo/anti-censorship/pluggable-transports/ptutil v0.0.0-20240710081135-6c4d8ed41027 // indirect
	go.uber.org/mock v0.5.0 // indirect
	go4.org/mem v0.0.0-20240501181205-ae6ca9944745 // indirect
	go4.org/netipx v0.0.0-20231129151722-fdeea329fbba // indirect
	golang.org/x/exp v0.0.0-20241108190413-2d47ceb2692f // indirect
	golang.org/x/sync v0.9.0 // indirect
	golang.org/x/time v0.8.0 // indirect
	golang.zx2c4.com/wireguard v0.0.0-20231211153847-12269c276173 // indirect
	golang.zx2c4.com/wireguard/windows v0.5.3 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	gvisor.dev/gvisor v0.0.0-20230928000133-4fe30062272c // indirect
	tailscale.com v1.58.2 // indirect
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/AndreasBriese/bbloom v0.0.0-20190825152654-46b345b51c96 // indirect
	github.com/alecthomas/units v0.0.0-20240927000941-0f3dac36c52b // indirect
	github.com/armon/go-proxyproto v0.1.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bifurcation/mint v0.0.0-20180306135233-198357931e61 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cheekybits/genny v1.0.0 // indirect
	github.com/cognusion/go-cache-lru v0.0.0-20170419142635-f73e2280ecea // indirect
	github.com/dchest/siphash v1.2.3 // indirect
	github.com/dgraph-io/badger v1.6.2 // indirect
	github.com/dgraph-io/ristretto v0.2.0 // indirect
	github.com/dsnet/compress v0.0.1 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/go-gorp/gorp/v3 v3.1.0 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/grafov/m3u8 v0.12.0 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/klauspost/cpuid/v2 v2.2.9 // indirect
	github.com/klauspost/reedsolomon v1.12.4 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-sqlite3 v1.14.24
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	github.com/oschwald/maxminddb-golang v1.13.1
	github.com/pion/datachannel v1.5.9 // indirect
	github.com/pion/dtls/v2 v2.2.12 // indirect
	github.com/pion/ice/v2 v2.3.37 // indirect
	github.com/pion/interceptor v0.1.37 // indirect
	github.com/pion/logging v0.2.2 // indirect
	github.com/pion/mdns v0.0.12 // indirect
	github.com/pion/randutil v0.1.0 // indirect
	github.com/pion/rtcp v1.2.14 // indirect
	github.com/pion/rtp v1.8.9 // indirect
	github.com/pion/sctp v1.8.34 // indirect
	github.com/pion/sdp/v3 v3.0.9 // indirect
	github.com/pion/srtp/v2 v2.0.20 // indirect
	github.com/pion/turn/v2 v2.1.6 // indirect
	github.com/pion/webrtc/v3 v3.3.4 // indirect
	github.com/prometheus/client_golang v1.20.5
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.60.1 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/refraction-networking/gotapdance v1.7.10 // indirect
	github.com/refraction-networking/utls v1.6.7 // indirect
	github.com/sergeyfrolov/bsbuffer v0.0.0-20180903213811-94e85abb8507 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/spf13/cobra v1.8.1
	github.com/syndtr/gocapability v0.0.0-20200815063812-42c35b437635 // indirect
	github.com/tailscale/hujson v0.0.0-20221223112325-20486734a56a
	github.com/templexxx/cpu v0.1.1 // indirect
	github.com/templexxx/xorsimd v0.4.3 // indirect
	github.com/tjfoc/gmsm v1.4.1 // indirect
	github.com/wader/filtertransport v0.0.0-20200316221534-bdd9e61eee78 // indirect
	github.com/xtaci/kcp-go/v5 v5.6.18 // indirect
	github.com/xtaci/smux v1.5.32 // indirect
	gitlab.com/yawning/bsaes.git v0.0.0-20190805113838-0a714cd429ec // indirect
	golang.org/x/mod v0.22.0 // indirect
	golang.org/x/term v0.26.0 // indirect
	golang.org/x/text v0.20.0 // indirect
	golang.org/x/tools v0.27.0 // indirect
	google.golang.org/protobuf v1.35.2 // indirect
)
