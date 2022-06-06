package netx

//
// Config struct.
//

import (
	"crypto/tls"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/bytecounter"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/tracex"
)

// Config contains configuration for creating new transports, dialers, etc. When
// any field of Config is nil/empty, we will use a suitable default.
type Config struct {
	BaseResolver        model.Resolver       // default: system resolver
	BogonIsError        bool                 // default: bogon is not error
	ByteCounter         *bytecounter.Counter // default: no explicit byte counting
	CacheResolutions    bool                 // default: no caching
	ContextByteCounting bool                 // default: no implicit byte counting
	DNSCache            map[string][]string  // default: cache is empty
	Dialer              model.Dialer         // default: dialer.DNSDialer
	FullResolver        model.Resolver       // default: base resolver + goodies
	QUICDialer          model.QUICDialer     // default: quicdialer.DNSDialer
	HTTP3Enabled        bool                 // default: disabled
	Logger              model.Logger         // default: no logging
	ProxyURL            *url.URL             // default: no proxy
	ReadWriteSaver      *tracex.Saver        // default: not saving I/O events
	Saver               *tracex.Saver        // default: not saving non-I/O events
	TLSConfig           *tls.Config          // default: attempt using h2
	TLSDialer           model.TLSDialer      // default: dialer.TLSDialer
}
