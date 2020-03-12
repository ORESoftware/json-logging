#!/usr/bin/env bash

set -eo pipefail

cd "$(dirname "$BASH_SOURCE")";

if [[ ! -f 'readme.md' ]]; then
  echo 'readme.md file is not in pwd, something is wrong.';
  exit 1;
fi

go_proj_path="$GOPATH/src/github.com/oresoftware/json-logging"
tmp_path="/tmp/gotemp/json-logging"

mkdir -p "$go_proj_path"
mkdir -p "$tmp_path"

if [[ ! -d "$tmp_path" ]]; then
  echo "$0: The following path does not exist: '$tmp_path'..."
  echo "$0: The above path needs to exist in order for this script to proceed.";
  exit 1;
fi

if [[ -L "$tmp_path" ]]; then
  echo "$0: The following path is a symlink but should not be: '$tmp_path'..."
  echo "$0: This script cannot proceed.";
  exit 1;
fi

if [[ ! -L "$go_proj_path" ]]; then
  echo "$0: The following path is not a symlink: '$go_proj_path'..."
  echo "$0: This script cannot proceed.";
  exit 1;
fi

if [[ ! -d "$go_proj_path" ]]; then
  echo "$0: The following path is a symlink but not a directory: '$go_proj_path'..."
  echo "$0: This script cannot proceed.";
  exit 1;
fi

rm -rf "$go_proj_path"
mv "$tmp_path" "$go_proj_path"
