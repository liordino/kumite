#!/usr/bin/env bash
# dojo-check for kumite — compile → vet → test, plus the proof artifact.
# The proof seals the exact tree (integration HEAD + working-tree delta hash),
# so a gate consumer (bin/release) can verify the green was over THIS tree.
# A failed run deletes the proof first and writes none: a red gate must never
# leave the previous green proof on disk for a stale-proof consumer to accept.
set -u
set -o pipefail
rm -f .dojo/proof/check-proof .dojo/proof/check-output.log
mkdir -p .dojo/proof
{
  status=0
  go build ./... || status=1
  go vet ./...   || status=1
  go test ./...  || status=1
  exit $status
} 2>&1 | tee .dojo/proof/check-output.log
run_status=${PIPESTATUS[0]}
if [ "$run_status" -ne 0 ]; then
  echo "dojo-check: FAILED (exit=$run_status) — no proof written" >&2
  exit "$run_status"
fi

tree_id=$(printf '%s:%s' "$(git rev-parse HEAD)" "$(git status --porcelain -- . ':(exclude).dojo' ':(exclude)dist' | sha256sum | awk '{print $1}')" | sha256sum | awk '{print $1}')
sha() { sha256sum "$1" | awk '{print $1}'; }
{
  echo "ts=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo "exit=$run_status"
  echo "output_sha256=$(sha .dojo/proof/check-output.log)"
  echo "tree_sha256=$tree_id"
} > .dojo/proof/check-proof