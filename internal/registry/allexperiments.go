package registry

//
// List of all implemented experiments.
//

import (
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/experiment/dash"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/dnscheck"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/dnsping"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/example"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/fbmessenger"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/hhfm"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/hirl"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/httphostheader"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/ndt7"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/psiphon"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/quicping"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/riseupvpn"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/run"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/signal"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/simplequicping"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/sniblocking"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/stunreachability"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/tcpping"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/telegram"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/tlsping"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/tlstool"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/tor"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/torsf"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/urlgetter"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/vanillator"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/webconnectivity"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/whatsapp"
	"github.com/ooni/probe-cli/v3/internal/model"
)

var experimentsByName = map[string]func() *Factory{
	"dash": func() *Factory {
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return dash.NewExperimentMeasurer(
					*config.(*dash.Config),
				)
			},
			config:        &dash.Config{},
			interruptible: true,
			inputPolicy:   model.InputNone,
		}
	},

	"dnscheck": func() *Factory {
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return dnscheck.NewExperimentMeasurer(
					*config.(*dnscheck.Config),
				)
			},
			config:      &dnscheck.Config{},
			inputPolicy: model.InputOrStaticDefault,
		}
	},

	"dnsping": func() *Factory {
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return dnsping.NewExperimentMeasurer(
					*config.(*dnsping.Config),
				)
			},
			config:      &dnsping.Config{},
			inputPolicy: model.InputOrStaticDefault,
		}
	},

	"example": func() *Factory {
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return example.NewExperimentMeasurer(
					*config.(*example.Config), "example",
				)
			},
			config: &example.Config{
				Message:   "Good day from the example experiment!",
				SleepTime: int64(time.Second),
			},
			interruptible: true,
			inputPolicy:   model.InputNone,
		}
	},

	"facebook_messenger": func() *Factory {
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return fbmessenger.NewExperimentMeasurer(
					*config.(*fbmessenger.Config),
				)
			},
			config:      &fbmessenger.Config{},
			inputPolicy: model.InputNone,
		}
	},

	"http_header_field_manipulation": func() *Factory {
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return hhfm.NewExperimentMeasurer(
					*config.(*hhfm.Config),
				)
			},
			config:      &hhfm.Config{},
			inputPolicy: model.InputNone,
		}
	},

	"http_host_header": func() *Factory {
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return httphostheader.NewExperimentMeasurer(
					*config.(*httphostheader.Config),
				)
			},
			config:      &httphostheader.Config{},
			inputPolicy: model.InputOrQueryBackend,
		}
	},

	"http_invalid_request_line": func() *Factory {
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return hirl.NewExperimentMeasurer(
					*config.(*hirl.Config),
				)
			},
			config:      &hirl.Config{},
			inputPolicy: model.InputNone,
		}
	},

	"ndt": func() *Factory {
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return ndt7.NewExperimentMeasurer(
					*config.(*ndt7.Config),
				)
			},
			config:        &ndt7.Config{},
			interruptible: true,
			inputPolicy:   model.InputNone,
		}
	},

	"psiphon": func() *Factory {
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return psiphon.NewExperimentMeasurer(
					*config.(*psiphon.Config),
				)
			},
			config:      &psiphon.Config{},
			inputPolicy: model.InputOptional,
		}
	},

	"quicping": func() *Factory {
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return quicping.NewExperimentMeasurer(
					*config.(*quicping.Config),
				)
			},
			config:      &quicping.Config{},
			inputPolicy: model.InputStrictlyRequired,
		}
	},

	"riseupvpn": func() *Factory {
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return riseupvpn.NewExperimentMeasurer(
					*config.(*riseupvpn.Config),
				)
			},
			config:      &riseupvpn.Config{},
			inputPolicy: model.InputNone,
		}
	},

	"run": func() *Factory {
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return run.NewExperimentMeasurer(
					*config.(*run.Config),
				)
			},
			config:      &run.Config{},
			inputPolicy: model.InputStrictlyRequired,
		}
	},

	"simplequicping": func() *Factory {
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return simplequicping.NewExperimentMeasurer(
					*config.(*simplequicping.Config),
				)
			},
			config:      &simplequicping.Config{},
			inputPolicy: model.InputStrictlyRequired,
		}
	},

	"signal": func() *Factory {
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return signal.NewExperimentMeasurer(
					*config.(*signal.Config),
				)
			},
			config:      &signal.Config{},
			inputPolicy: model.InputNone,
		}
	},

	"sni_blocking": func() *Factory {
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return sniblocking.NewExperimentMeasurer(
					*config.(*sniblocking.Config),
				)
			},
			config:      &sniblocking.Config{},
			inputPolicy: model.InputOrQueryBackend,
		}
	},

	"stunreachability": func() *Factory {
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return stunreachability.NewExperimentMeasurer(
					*config.(*stunreachability.Config),
				)
			},
			config:      &stunreachability.Config{},
			inputPolicy: model.InputOrStaticDefault,
		}
	},

	"tcpping": func() *Factory {
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return tcpping.NewExperimentMeasurer(
					*config.(*tcpping.Config),
				)
			},
			config:      &tcpping.Config{},
			inputPolicy: model.InputStrictlyRequired,
		}
	},

	"tlsping": func() *Factory {
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return tlsping.NewExperimentMeasurer(
					*config.(*tlsping.Config),
				)
			},
			config:      &tlsping.Config{},
			inputPolicy: model.InputStrictlyRequired,
		}
	},

	"telegram": func() *Factory {
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return telegram.NewExperimentMeasurer(
					*config.(*telegram.Config),
				)
			},
			config:      &telegram.Config{},
			inputPolicy: model.InputNone,
		}
	},

	"tlstool": func() *Factory {
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return tlstool.NewExperimentMeasurer(
					*config.(*tlstool.Config),
				)
			},
			config:      &tlstool.Config{},
			inputPolicy: model.InputOrQueryBackend,
		}
	},

	"tor": func() *Factory {
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return tor.NewExperimentMeasurer(
					*config.(*tor.Config),
				)
			},
			config:      &tor.Config{},
			inputPolicy: model.InputNone,
		}
	},

	"torsf": func() *Factory {
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return torsf.NewExperimentMeasurer(
					*config.(*torsf.Config),
				)
			},
			config:      &torsf.Config{},
			inputPolicy: model.InputNone,
		}
	},

	"urlgetter": func() *Factory {
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return urlgetter.NewExperimentMeasurer(
					*config.(*urlgetter.Config),
				)
			},
			config:      &urlgetter.Config{},
			inputPolicy: model.InputStrictlyRequired,
		}
	},

	"vanilla_tor": func() *Factory {
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return vanillator.NewExperimentMeasurer(
					*config.(*vanillator.Config),
				)
			},
			config:      &vanillator.Config{},
			inputPolicy: model.InputNone,
		}
	},

	"web_connectivity": func() *Factory {
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return webconnectivity.NewExperimentMeasurer(
					*config.(*webconnectivity.Config),
				)
			},
			config:      &webconnectivity.Config{},
			inputPolicy: model.InputOrQueryBackend,
		}
	},

	"whatsapp": func() *Factory {
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return whatsapp.NewExperimentMeasurer(
					*config.(*whatsapp.Config),
				)
			},
			config:      &whatsapp.Config{},
			inputPolicy: model.InputNone,
		}
	},
}

// ExperimentNames returns the name of all experiments
func ExperimentNames() (names []string) {
	for key := range experimentsByName {
		names = append(names, key)
	}
	return
}
