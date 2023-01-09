#doc:
#doc: # MONOREPO/tools/libgit.bash
#doc:
#doc: Functions implementing the monorepo.

reporoot=$(dirname $(dirname $(dirname $(realpath $0))))

source $reporoot/MONOREPO/tools/gitconfig.bash
source $reporoot/MONOREPO/tools/libcore.bash

#doc:
#doc: ## repo_to_dir (function)
#doc:
#doc: Returns the absolute directory of a given repository.
repo_to_dir() {
	# Special case of the probe-cli repository itself
	if [[ $1 == "." ]]; then
		echo "$reporoot"
		return
	fi
	# The repository may be in the git@github.com:foo/bar format or
	# it may already be a path, so we use basename to normalize.
	echo "$reporoot/MONOREPO/repo/$(basename $1)"
}

#doc:
#doc: ## get_branch_name (function)
#doc:
#doc: Returns the branch name of a given repository.
get_branch_name() {
	local dirname=$(repo_to_dir $1)
	(cd $dirname && git symbolic-ref --short HEAD)
}

#doc:
#doc: ## fail_if_dirty (function)
#doc:
#doc: Exits with error if the given repository is dirty.
fail_if_dirty() {
	local dirname=$(repo_to_dir $1)
	(
		cd $dirname && [[ -z $(git status -s) ]] || {
			fatal "$dirname contains modified or untracked files"
			exit 1
		}
	)
}

#doc:
#doc: ## fail_if_not_main (function)
#doc:
#doc: Exits with error if we're not at the main branch.
fail_if_not_main() {
	local dirname=$(repo_to_dir $1)
	local branchname=$(get_branch_name $1)
	if [[ $branchname != "main" && $branchname != "master" ]]; then
		fatal "${dirname}'s branch is neither main nor master"
		exit 1
	fi
}

#doc:
#doc: ## fail_if_main (function)
#doc:
#doc: Exits with error if we're at the main branch.
fail_if_main() {
	local dirname=$(repo_to_dir $1)
	local branchname=$(get_branch_name $1)
	if [[ $branchname == "main" || $branchname == "master" ]]; then
		fatal "${dirname}'s branch is in its main (or master) branch"
		exit 1
	fi
}

#doc:
#doc: ## for_each_repo (function)
#doc:
#doc: Executes the function given as argument for each
#doc: repo in the `repositories` bash array.
for_each_repo() {
	local action=$1
	shift
	for repo in ${repositories[@]}; do
		$action $repo "$@"
	done
}

#doc:
#doc: ## maybe_clone (function)
#doc:
#doc: Calls git clone unless we already cloned the given repository.
maybe_clone() {
	local dirname=$(repo_to_dir $1)
	[[ -d $dirname ]] || run git clone -q $1 $dirname
}

#doc:
#doc: ## maybe_pull (function)
#doc:
#doc: Calls git pull on the given repository.
maybe_pull() {
	local dirname=$(repo_to_dir $1)
	(run cd $dirname && run git pull)
}

#doc:
#doc: ## prune_remote_branches (function)
#doc:
#doc: Prune branches removed also from the remote.
prune_remote_branches() {
	local dirname=$(repo_to_dir $1)
	(run cd $dirname && run git remote prune origin)
}

#doc:
#doc: ## sync_one_repo (function)
#doc:
#doc: Sync algorithm applied to a single repo.
sync_one_repo() {
	maybe_clone $1
	fail_if_not_main $1
	fail_if_dirty $1
	maybe_pull $1
	prune_remote_branches $1
}

#doc:
#doc: ## subcommand_sync (function)
#doc:
#doc: Implementation of the sync subcommand.
subcommand_sync() {
	for_each_repo sync_one_repo
}

#doc:
#doc: ## checkout_one_repo (function)
#doc:
#doc: Checkout algorithm applied to a single repo.
checkout_one_repo() {
	local dirname=$(repo_to_dir $1)
	local branch_name=$2
	(
		run cd $dirname
		run git checkout $branch_name || git checkout -b $branch_name
	)
}

#doc:
#doc: ## subcommand_checkout (function)
#doc:
#doc: Implementation of the checkout subcommand.
subcommand_checkout() {
	if [[ $# -ne 1 ]]; then
		warn "missing required positional argument {branch}"
		echo "" 1>&2
		echo "usage: gitx checkout {branch}" 1>&2
		echo "" 1>&2
		exit 1
	fi
	local branch_name=$1
	for_each_repo checkout_one_repo $branch_name
}

#doc:
#doc: ## clean_one_repo (function)
#doc:
#doc: Clean algorithm applied to a single repo.
clean_one_repo() {
	local dirname=$(repo_to_dir $1)
	shift
	(
		run cd $dirname
		local extraflags=""
		if [[ $(basename $dirname) == "probe-cli" ]]; then
			# Avoid completely removing all the cloned subrepos
			# as well as the important local.bash config file
			extraflags="-e MONOREPO/repo/ -e MONOREPO/tools/local.bash"
		fi
		run git clean -dffx $extraflags
	)
}

#doc:
#doc: ## subcommand_clean (function)
#doc:
#doc: Implementation of the clean subcommand.
subcommand_clean() {
	for_each_repo clean_one_repo
}

#doc:
#doc: ## commit_one_repo (function)
#doc:
#doc: Attempts to commit changes for a single repo.
commit_one_repo() {
	local dirname=$(repo_to_dir $1)
	local commit_message_file=$2
	fail_if_main $1
	(
		run cd $dirname
		if [[ -z $(git status -s) ]]; then
			info "Nothing to commit"
			return
		fi
		run git commit -aF $commit_message_file
	)
}

#doc:
#doc: ## subcommand_commit (function)
#doc:
#doc: Implementation of the commit subcommand.
subcommand_commit() {
	if [[ $# -ne 1 ]]; then
		warn "missing required positional argument {commit-message-file}"
		echo "" 1>&2
		echo "usage: $0 {commit-message-file}" 1>&2
		echo "" 1>&2
		exit 1
	fi
	local commit_message_file=$(realpath $1)
	if [[ ! -f $commit_message_file ]]; then
		fatal "$commit_message_file does not exist" 1>&2
	fi
	for_each_repo commit_one_repo $commit_message_file
}

#doc:
#doc: ## status_one_repo (function)
#doc:
#doc: Shows the status of a given repo.
status_one_repo() {
	local dirname=$(repo_to_dir $1)
	local namelen=$(printf $(basename $dirname) | wc -c)
	printf "[$(basename $dirname)]"
	local padding=$((15 - namelen))
	while [[ $padding -gt 0 ]]; do
		printf " "
		padding=$((padding - 1))
	done
	local branch_name=$(get_branch_name $1)
	local ref=$(cd $dirname && git --no-pager log -1 --oneline)
	printf "[%s] %s\n" "$branch_name" "$ref"
	(
		cd $dirname
		git status -s
	)
}

#doc:
#doc: ## subcommand_status (function)
#doc:
#doc: Implements the status subcommand.
subcommand_status() {
	for_each_repo status_one_repo
}

#doc:
#doc: ## reset_one_repo (function)
#doc:
#doc: Calls git reset --hard on one repo and returns
#doc: to the main branch of the repository.
reset_one_repo() {
	local dirname=$(repo_to_dir $1)
	shift
	echo ""
	(
		run cd $dirname
		run git reset --hard HEAD
		if [[ -n $(git branch --list main) ]]; then
			run git checkout main
		elif [[ -n $(git branch --list master) ]]; then
			run git checkout master
		else
			fatal "default branch not named master or main"
		fi
		current=$(git branch --show-current)
		if [[ $# > 0 && "$1" == "-f" ]]; then
			for branch in "$(git for-each-ref --format='%(refname:short)' refs/heads/)"; do
				info "current: $current"
				info "branch: $branch"
				if [[ $branch != $current && $branch != "" ]]; then
					run git branch -D $branch
				fi
			done
		fi
	)
}

#doc:
#doc: ## subcommand_reset (function)
#doc:
#doc: Implements the reset subcommand.
subcommand_reset() {
	for_each_repo reset_one_repo "$@"
}

#doc:
#doc: ## diff_one_repo (function)
#doc:
#doc: Calls git diff on a given repo.
diff_one_repo() {
	local dirname=$(repo_to_dir $1)
	shift
	local diff_args="$@"
	(cd $dirname && git --no-pager diff --src-prefix=$dirname/a/ \
		--dst-prefix=$dirname/b/ --color=always $diff_args)
}

#doc:
#doc: ## subcommand_diff (function)
#doc:
#doc: Implements the diff subcommand.
subcommand_diff() {
	local diff_args="$@"
	if [[ $diff_args == "" ]]; then
		diff_args="--"
	fi
	for_each_repo diff_one_repo "$diff_args"
}

#doc:
#doc: ## push_one_repo (function)
#doc:
#doc: Pushes changes in one repo to the remote.
push_one_repo() {
	local dirname=$(repo_to_dir $1)
	local branchname=$(get_branch_name $1)
	fail_if_main $1
	fail_if_dirty $1
	(
		run cd $dirname
		if [[ -n $(git branch --list main) ]]; then
			# Note: need to use `--` to ensure git knows we want to
			# diff with respect to the branch just in case there's
			# a file named main in tree.
			if [[ -z "$(git diff main --)" ]]; then
				info "Nothing to push"
				return
			fi
		elif [[ -n $(git branch --list master) ]]; then
			# Same as above
			if [[ -z "$(git diff master --)" ]]; then
				info "Nothing to push"
				return
			fi
		else
			fatal "default branch not named master or main"
		fi
		run git push -u origin $branchname
	)
}

#doc:
#doc: subcommand_push (function)
#doc:
#doc: Implements the push subcommand
subcommand_push() {
	for_each_repo push_one_repo
}
