#!/usr/bin/env bash

set -eo pipefail

cd "$(dirname "$BASH_SOURCE")";


if [[ ! -f 'readme.md' ]]; then
  echo 'readme.md file is not in pwd, something is wrong.';
  exit 1;
fi

#pth_to_make="$GOPATH/src/ores/json-logging"
local_proj_path="$PWD";

if [[ ! -f "$local_proj_path/readme.md" ]]; then
  echo 'readme.md file is not in pwd, something is wrong.';
  exit 1;
fi

export go_proj_path_root="$GOPATH/pkg/mod/github.com/oresoftware"
chmod -R 777 "$go_proj_path_root"

tmp_path="$HOME/.oresoftware/go-mods/temp/json-logging"

mkdir -p "$go_proj_path_root"
mkdir -p "$tmp_path"

my_regex='^json-logging@'

find_dirs(){
  cd "$go_proj_path_root" && cd .. && find "$go_proj_path_root" -mindepth 1 -maxdepth 1 # -type 'd,l'
}

export -f find_dirs

for go_proj_path in $(find_dirs); do

    if [[ -f "$go_proj_path" ]]; then
       echo "$0: the following is a unexpectedly a file: '$go_proj_path'...";
       continue;
    fi

    last_dir="$(basename "$go_proj_path")"

    echo "$last_dir"

    if [[ ! "$last_dir" =~ $my_regex ]]; then
         echo "$0: the following file did not match regex: '$last_dir' ..."
         continue;
    fi

    if [[ -L "$go_proj_path" ]]; then
      echo "$0: The following path is a symlink: $go_proj_path"
      echo "$0: The above path is symlinked to '$(realpath "$go_proj_path")'..."
    fi

    if [[ ! -d "$go_proj_path" ]]; then
      echo "$0: The following path does not exist: '$go_proj_path'..."
      echo "$0: The above path needs to exist in order for this script to proceed.";
      continue;
    fi

    tmp_proj_path="$tmp_path/$last_dir"

    if [[ -L "$tmp_proj_path" ]]; then
       echo "$0: the following path is already a symlink: '$tmp_proj_path' ...."
       rm -rf "$tmp_proj_path"
    fi

    if [[ -d "$tmp_proj_path" ]]; then
       echo "$0: the following path is already a dir: '$tmp_proj_path' ...."
       rm -rf "$tmp_proj_path"
    fi

    mkdir -p "$tmp_proj_path"
    rm -rf "$tmp_proj_path"

    mv "$go_proj_path" "$tmp_proj_path"
    rsync -r --exclude='.git' --exclude='.idea' "$local_proj_path" "$go_proj_path"
#    ln -s "$local_proj_path" "$go_proj_path"

done;




#rm -rf "$GOPATH/src/github.com/oresoftware/json-logging"
#mv "/tmp/gotemp/json-logging" "$GOPATH/src/github.com/oresoftware/json-logging"  # put it back


#mkdir -p "$pth_to_make";
#rm -rf "$pth_to_make";

#ln -sf "$PWD" "$go_proj_path";