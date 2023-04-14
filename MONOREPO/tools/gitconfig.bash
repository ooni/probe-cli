#doc:
#doc: # MONOREPO/tools/gitconfig.bash
#doc:
#doc: Git configuration.

reporoot=$(dirname $(dirname $(dirname $(realpath $0))))

#doc:
#doc: ## repositories (array)
#doc:
#doc: List of repositories to track
repositories=(
	. # the dot is git@github.com:ooni/probe-cli and MUST be first
	git@github.com:ooni/probe-android
	git@github.com:ooni/probe-desktop
)

#doc:
#doc: If a file named ./MONOREPO/tools/local.bash exists, we
#doc: will source it right after defining the repositories.
#doc:
#doc: This file does not exist by default but you can create
#doc: it manually to customize the repositories to track.
#doc:
#doc: This is especially important for non-OONI developers
#doc: because the default config fetches private repos.
if [[ -f $reporoot/MONOREPO/tools/local.bash ]]; then
	source $reporoot/MONOREPO/tools/local.bash
fi
