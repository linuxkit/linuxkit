#!/bin/sh

set -e

cd /src

>&2 echo "gofmt..."
test -z $(gofmt -s -l .| grep -v .pb. | grep -v */vendor/ | tee /dev/stderr)

>&2 echo "govet..."
test -z $(go tool vet -printf=false . 2>&1 | grep -v */vendor/ | tee /dev/stderr)

>&2 echo "golint..."
test -z $(find . -type f -name "*.go" -not -path "*/vendor/*" -not -name "*.pb.*" -exec golint {} \; | tee /dev/stderr)

>&2 echo "Successful lint check!"