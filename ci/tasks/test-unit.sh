#!/usr/bin/env bash

set -euo pipefail

cd socks5-proxy

go run github.com/onsi/ginkgo/v2/ginkgo
