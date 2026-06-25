#!/bin/sh
set -e

TOOLS_DIR="${PRECOMMIT_TOOLS_DIR:-/go/precommit-tools}"
export PATH="${TOOLS_DIR}/bin:${PATH}"

if [ ! -f "${TOOLS_DIR}/.installed" ]; then
	apk add --no-cache python3 git pre-commit curl

	mkdir -p "${TOOLS_DIR}/bin"

	BUF_VERSION=1.47.2
	curl -sSL \
		"https://github.com/bufbuild/buf/releases/download/v${BUF_VERSION}/buf-$(uname -s)-$(uname -m)" \
		-o "${TOOLS_DIR}/bin/buf"
	chmod +x "${TOOLS_DIR}/bin/buf"

	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh \
		| sh -s -- -b "${TOOLS_DIR}/bin"

	touch "${TOOLS_DIR}/.installed"
fi

cd /app
go mod download

exec pre-commit run --all-files "$@"
