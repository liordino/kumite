#!/usr/bin/env bash
# dojo-check for kumite — compile → vet → test, plus the proof artifact.
# The proof seals the exact tree (integration HEAD + working-tree delta hash),
# so a gate consumer (bin/release) can verify the green was over THIS tree.
set -e
set -o pipefail
mkdir -p .dojo/proof
{
  go build ./...
  go vet ./...
  go test ./...
} 2>&1 | tee .dojo/proof/check-output.log

tree_id=$(printf '%s:%s' "$(git rev-parse HEAD)" "$(git status --porcelain -- . ':(exclude).dojo' ':(exclude)dist' | sha256sum | awk '{print $1}')" | sha256sum | awk '{print $1}')
sha() { sha256sum "$1" | awk '{print $1}'; }
{
  echo "ts=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo "exit=0"
  echo "output_sha256=$(sha .dojo/proof/check-output.log)"
  echo "tree_sha256=$tree_id"
} > .dojo/proof/check-proof