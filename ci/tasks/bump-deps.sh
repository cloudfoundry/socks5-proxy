#!/bin/bash

set -euo pipefail

git clone socks5-proxy bumped-socks5-proxy

cd bumped-socks5-proxy

go get -u ./...
go mod tidy
go mod vendor

if [ "$(git status --porcelain)" != "" ]; then
  git status
  git add go.sum go.mod
  git config user.name "CI Bot"
  git config user.email "cf-bosh-eng@pivotal.io"
  git commit -m "Update vendored dependencies"
fi
