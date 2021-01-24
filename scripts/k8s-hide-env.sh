#!/bin/sh
#
# Sources passed in file before executing command. Passed in file will be deleted immediately after sourcing.
#
# Usage:
# 		./k8s-hide-env.sh path-to-env-file 'commands'
#
# Example:
#		./k8s-hide-env.sh ./envs.sh 'echo $SOME_ENV'
#
set -eu

# parameter validation
[ -z "$1" ] && echo "Please define pass an env file" >&2 && exit 1
[ ! -e "$1" ] && echo "Env file does not exist" >&2 && exit 1
[ -z "$2" ] && echo "Please define main command" >&2 && exit 1
env_file="$1"
cmd="$2"

# make sure source file is deleted in any case
trap '[ -e $env_file ] && rm $env_file' EXIT
# source env file
# shellcheck disable=SC1090
. "$env_file"
# remove env file
rm "$env_file"
# execute main command
sh -c "$cmd"