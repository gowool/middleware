#!/bin/bash

set -e

DIR="./*/"

print_error() {
	echo -e "ERROR: $*"
}

bump_wool() {
  for d in ${DIR}; do
    if [ -f "${d}go.mod" ]; then
      pushd "$d"
        go get "github.com/gowool/wool@$*"
        go mod tidy
      popd
    fi
  done
}

if [ "$1" = "" ]; then
  print_error "required WOOL version or commit hash"
  exit 1
fi

bump_wool "$1"

exit 0
