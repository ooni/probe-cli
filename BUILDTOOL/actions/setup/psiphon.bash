#help:
#help: Sets the OONI_PSIPHON_TAGS environment variable properly depending
#help: on whether psiphon files are present inside ./internal/engine.
#help:

if [[ ! -f ./internal/engine/psiphon-config.json.age ]]; then
	run export OONI_PSIPHON_TAGS=""
elif [[ ! -f ./internal/engine/psiphon-config.key ]]; then
	run export OONI_PSIPHON_TAGS=""
else
	run export OONI_PSIPHON_TAGS="ooni_psiphon_config"
fi
