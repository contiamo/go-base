#!/bin/bash

cd "$(git rev-parse --show-toplevel)"

source .bingo/variables.env

semver=""
if [ -f "${GIT_SEMVER}" ]; then
    semver=$($GIT_SEMVER | tr '+' '.')
else
    semver=$(git describe --tags --always 2>/dev/null)
fi

if ! git diff --quiet; then
    semver="$semver.dirty.$(date +"%s")"
fi

echo "$semver"