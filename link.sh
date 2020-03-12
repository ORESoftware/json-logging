#!/usr/bin/env bash

set -eo pipefail

cd "$(dirname "$BASH_SOURCE")";


if [[ ! -f 'readme.md' ]]; then
  echo 'readme.md file is not in pwd, something is wrong.';
  exit 1;
fi

#pth_to_make="$GOPATH/src/ores/json-logging"

go_proj_path="$GOPATH/src/github.com/oresoftware/json-logging"
tmp_path="/tmp/gotemp/json-logging"

mkdir -p "$go_proj_path"
mkdir -p "$tmp_path"

if [[ ! -d "$go_proj_path" ]]; then
  echo "$0: The following path does not exist: $go_proj_path"
  echo "$0: The above path needs to exist in order for this script to proceed.";
  exit 1;
fi

if [[ -L "$go_proj_path" ]]; then
  echo "$0: The following path is already symlinked: $go_proj_path"
  echo "$0: The above path is symlinked to '$(realpath "$go_proj_path")'..."
  if [[ "$(realpath "$go_proj_path")" != "$tmp_path" ]]; then
      echo "But the symlinked path does not match the expected path: $tmp_path"
  fi
  exit 1;
fi

rm -rf "$tmp_path"

mv "$go_proj_path"  "$tmp_path"

#rm -rf "$GOPATH/src/github.com/oresoftware/json-logging"
#mv "/tmp/gotemp/json-logging" "$GOPATH/src/github.com/oresoftware/json-logging"  # put it back


#mkdir -p "$pth_to_make";
#rm -rf "$pth_to_make";

ln -sf "$PWD" "$go_proj_path";