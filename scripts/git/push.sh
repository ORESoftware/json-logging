#!/usr/bin/env bash

set -eo pipefail;

if [[ -f "$PWD/wss/package.json" ]]; then
    cd "$PWD/wss"
fi

## Make sure it can compile before pushing
echo 'Compile go project to /dev/null so we dont push code that doesnt work lol'
go build -p 6 -o '/dev/null' -v "$PWD/jlog/lib"
go build -p 6 -o '/dev/null' -v "$PWD/jlog/mult"
go build -p 6 -o '/dev/null' -v "$PWD/test"

ssh-add -D
ssh-add ~/.ssh/id_ed25519

combined=""
for arg in "${@}"; do
  combined="${combined} ${arg}"
done

trimmed="$(echo "$combined" | xargs)"

if test "${trimmed}" == '' ; then
  trimmed="squash-me";
fi

git add .
git add -A
git commit -am "${trimmed}" || {
  echo "could not create a new commit"
}

git push origin || {
  echo
}

#git push gitlab || {
#  echo
#}