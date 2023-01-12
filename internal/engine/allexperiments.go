package engine

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
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/randomtraffic"
)

var experimentsByName = map[string]func(*Session) *ExperimentBuilder{
	"dash": func(session *Session) *ExperimentBuilder {
		return &ExperimentBuilder{
			build: func(config interface{}) *Experiment {
				return NewExperiment(session, dash.NewExperimentMeasurer(
					*config.(*dash.Config),
				))
			},
			config:        &dash.Config{},
			interruptible: true,
			inputPolicy:   InputNone,
		}
	},

	"dnscheck": func(session *Session) *ExperimentBuilder {
		return &ExperimentBuilder{
			build: func(config interface{}) *Experiment {
				return NewExperiment(session, dnscheck.NewExperimentMeasurer(
					*config.(*dnscheck.Config),
				))
			},
			config:      &dnscheck.Config{},
			inputPolicy: InputOrStaticDefault,
		}
	},

	"dnsping": func(session *Session) *ExperimentBuilder {
		return &ExperimentBuilder{
			build: func(config interface{}) *Experiment {
				return NewExperiment(session, dnsping.NewExperimentMeasurer(
					*config.(*dnsping.Config),
				))
			},
			config:      &dnsping.Config{},
			inputPolicy: InputOrStaticDefault,
		}
	},

	"example": func(session *Session) *ExperimentBuilder {
		return &ExperimentBuilder{
			build: func(config interface{}) *Experiment {
				return NewExperiment(session, example.NewExperimentMeasurer(
					*config.(*example.Config), "example",
				))
			},
			config: &example.Config{
				Message:   "Good day from the example experiment!",
				SleepTime: int64(time.Second),
			},
			interruptible: true,
			inputPolicy:   InputNone,
		}
	},

	"facebook_messenger": func(session *Session) *ExperimentBuilder {
		return &ExperimentBuilder{
			build: func(config interface{}) *Experiment {
				return NewExperiment(session, fbmessenger.NewExperimentMeasurer(
					*config.(*fbmessenger.Config),
				))
			},
			config:      &fbmessenger.Config{},
			inputPolicy: InputNone,
		}
	},

	"http_header_field_manipulation": func(session *Session) *ExperimentBuilder {
		return &ExperimentBuilder{
			build: func(config interface{}) *Experiment {
				return NewExperiment(session, hhfm.NewExperimentMeasurer(
					*config.(*hhfm.Config),
				))
			},
			config:      &hhfm.Config{},
			inputPolicy: InputNone,
		}
	},

	"http_host_header": func(session *Session) *ExperimentBuilder {
		return &ExperimentBuilder{
			build: func(config interface{}) *Experiment {
				return NewExperiment(session, httphostheader.NewExperimentMeasurer(
					*config.(*httphostheader.Config),
				))
			},
			config:      &httphostheader.Config{},
			inputPolicy: InputOrQueryBackend,
		}
	},

	"http_invalid_request_line": func(session *Session) *ExperimentBuilder {
		return &ExperimentBuilder{
			build: func(config interface{}) *Experiment {
				return NewExperiment(session, hirl.NewExperimentMeasurer(
					*config.(*hirl.Config),
				))
			},
			config:      &hirl.Config{},
			inputPolicy: InputNone,
		}
	},

	"ndt": func(session *Session) *ExperimentBuilder {
		return &ExperimentBuilder{
			build: func(config interface{}) *Experiment {
				return NewExperiment(session, ndt7.NewExperimentMeasurer(
					*config.(*ndt7.Config),
				))
			},
			config:        &ndt7.Config{},
			interruptible: true,
			inputPolicy:   InputNone,
		}
	},

	"psiphon": func(session *Session) *ExperimentBuilder {
		return &ExperimentBuilder{
			build: func(config interface{}) *Experiment {
				return NewExperiment(session, psiphon.NewExperimentMeasurer(
					*config.(*psiphon.Config),
				))
			},
			config:      &psiphon.Config{},
			inputPolicy: InputOptional,
		}
	},

	"quicping": func(session *Session) *ExperimentBuilder {
		return &ExperimentBuilder{
			build: func(config interface{}) *Experiment {
				return NewExperiment(session, quicping.NewExperimentMeasurer(
					*config.(*quicping.Config),
				))
			},
			config:      &quicping.Config{},
			inputPolicy: InputStrictlyRequired,
		}
	},

	"riseupvpn": func(session *Session) *ExperimentBuilder {
		return &ExperimentBuilder{
			build: func(config interface{}) *Experiment {
				return NewExperiment(session, riseupvpn.NewExperimentMeasurer(
					*config.(*riseupvpn.Config),
				))
			},
			config:      &riseupvpn.Config{},
			inputPolicy: InputNone,
		}
	},

	"run": func(session *Session) *ExperimentBuilder {
		return &ExperimentBuilder{
			build: func(config interface{}) *Experiment {
				return NewExperiment(session, run.NewExperimentMeasurer(
					*config.(*run.Config),
				))
			},
			config:      &run.Config{},
			inputPolicy: InputStrictlyRequired,
		}
	},

	"simplequicping": func(session *Session) *ExperimentBuilder {
		return &ExperimentBuilder{
			build: func(config interface{}) *Experiment {
				return NewExperiment(session, simplequicping.NewExperimentMeasurer(
					*config.(*simplequicping.Config),
				))
			},
			config:      &simplequicping.Config{},
			inputPolicy: InputStrictlyRequired,
		}
	},

	"signal": func(session *Session) *ExperimentBuilder {
		return &ExperimentBuilder{
			build: func(config interface{}) *Experiment {
				return NewExperiment(session, signal.NewExperimentMeasurer(
					*config.(*signal.Config),
				))
			},
			config:      &signal.Config{},
			inputPolicy: InputNone,
		}
	},

	"sni_blocking": func(session *Session) *ExperimentBuilder {
		return &ExperimentBuilder{
			build: func(config interface{}) *Experiment {
				return NewExperiment(session, sniblocking.NewExperimentMeasurer(
					*config.(*sniblocking.Config),
				))
			},
			config:      &sniblocking.Config{},
			inputPolicy: InputOrQueryBackend,
		}
	},

	"stunreachability": func(session *Session) *ExperimentBuilder {
		return &ExperimentBuilder{
			build: func(config interface{}) *Experiment {
				return NewExperiment(session, stunreachability.NewExperimentMeasurer(
					*config.(*stunreachability.Config),
				))
			},
			config:      &stunreachability.Config{},
			inputPolicy: InputOrStaticDefault,
		}
	},

	"tcpping": func(session *Session) *ExperimentBuilder {
		return &ExperimentBuilder{
			build: func(config interface{}) *Experiment {
				return NewExperiment(session, tcpping.NewExperimentMeasurer(
					*config.(*tcpping.Config),
				))
			},
			config:      &tcpping.Config{},
			inputPolicy: InputStrictlyRequired,
		}
	},

	"tlsping": func(session *Session) *ExperimentBuilder {
		return &ExperimentBuilder{
			build: func(config interface{}) *Experiment {
				return NewExperiment(session, tlsping.NewExperimentMeasurer(
					*config.(*tlsping.Config),
				))
			},
			config:      &tlsping.Config{},
			inputPolicy: InputStrictlyRequired,
		}
	},

	"telegram": func(session *Session) *ExperimentBuilder {
		return &ExperimentBuilder{
			build: func(config interface{}) *Experiment {
				return NewExperiment(session, telegram.NewExperimentMeasurer(
					*config.(*telegram.Config),
				))
			},
			config:      &telegram.Config{},
			inputPolicy: InputNone,
		}
	},

	"tlstool": func(session *Session) *ExperimentBuilder {
		return &ExperimentBuilder{
			build: func(config interface{}) *Experiment {
				return NewExperiment(session, tlstool.NewExperimentMeasurer(
					*config.(*tlstool.Config),
				))
			},
			config:      &tlstool.Config{},
			inputPolicy: InputOrQueryBackend,
		}
	},

	"tor": func(session *Session) *ExperimentBuilder {
		return &ExperimentBuilder{
			build: func(config interface{}) *Experiment {
				return NewExperiment(session, tor.NewExperimentMeasurer(
					*config.(*tor.Config),
				))
			},
			config:      &tor.Config{},
			inputPolicy: InputNone,
		}
	},

	"torsf": func(session *Session) *ExperimentBuilder {
		return &ExperimentBuilder{
			build: func(config interface{}) *Experiment {
				return NewExperiment(session, torsf.NewExperimentMeasurer(
					*config.(*torsf.Config),
				))
			},
			config:      &torsf.Config{},
			inputPolicy: InputNone,
		}
	},

	"urlgetter": func(session *Session) *ExperimentBuilder {
		return &ExperimentBuilder{
			build: func(config interface{}) *Experiment {
				return NewExperiment(session, urlgetter.NewExperimentMeasurer(
					*config.(*urlgetter.Config),
				))
			},
			config:      &urlgetter.Config{},
			inputPolicy: InputStrictlyRequired,
		}
	},

	"vanilla_tor": func(session *Session) *ExperimentBuilder {
		return &ExperimentBuilder{
			build: func(config interface{}) *Experiment {
				return NewExperiment(session, vanillator.NewExperimentMeasurer(
					*config.(*vanillator.Config),
				))
			},
			config:      &vanillator.Config{},
			inputPolicy: InputNone,
		}
	},

	"web_connectivity": func(session *Session) *ExperimentBuilder {
		return &ExperimentBuilder{
			build: func(config interface{}) *Experiment {
				return NewExperiment(session, webconnectivity.NewExperimentMeasurer(
					*config.(*webconnectivity.Config),
				))
			},
			config:      &webconnectivity.Config{},
			inputPolicy: InputOrQueryBackend,
		}
	},

	"whatsapp": func(session *Session) *ExperimentBuilder {
		return &ExperimentBuilder{
			build: func(config interface{}) *Experiment {
				return NewExperiment(session, whatsapp.NewExperimentMeasurer(
					*config.(*whatsapp.Config),
				))
			},
			config:      &whatsapp.Config{},
			inputPolicy: InputNone,
		}
	},
	"randomtraffic": func(session *Session) *ExperimentBuilder {
		return &ExperimentBuilder{
			build: func(config interface{}) *Experiment {
				return newExperiment(session, randomtraffic.NewExperimentMeasurer(
					*config.(*randomtraffic.Config),
				))
			},
			config: &randomtraffic.Config{},
			inputPolicy:   InputNone,
		}
	},
}

// AllExperiments returns the name of all experiments
func AllExperiments() []string {
	var names []string
	for key := range experimentsByName {
		names = append(names, key)
	}
	return names
}
