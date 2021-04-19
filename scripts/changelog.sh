#!/bin/bash

LATEST_RELEASE=$(git describe --tags --abbrev=0)
CHANGELOG_START=${CHANGELOG_START:-"$LATEST_RELEASE"}
CHANGELOG_END=${CHANGELOG_END:-HEAD}

git --no-pager log --no-merges --pretty=format:"%h : %s (by %an)" "${CHANGELOG_START}...${CHANGELOG_END}"
