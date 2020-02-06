#!/usr/bin/env bash

set -eo pipefail

cd "$(dirname "$BASH_SOURCE")";

mkdir -p "$GOPATH/src/ores";

ln -s "$PWD" "$GOPATH/src/ores/json-logging";