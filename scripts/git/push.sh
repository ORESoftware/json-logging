#!/usr/bin/env bash

set -eo pipefail;

if [[ -f "$(cd .. && pwd)/go.mod" ]]; then
    cd ..
fi

if [[ -f "$PWD/json-logging/go.mod" ]]; then
    cd "$PWD/json-logging"
fi

## Make sure it can compile before pushing
echo 'Compile go project to /dev/null so we dont push code that doesnt work lol'
go build -p 6 -o '/dev/null' -v "$PWD/jlog/lib"
echo 'Compile ./jlog to /dev/null....'
go build -p 6 -o '/dev/null' -v "$PWD/jlog/mult"
echo 'Compile ./test to /dev/null'
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