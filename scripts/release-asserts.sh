#!/usr/bin/env bash
# release-asserts — seeded-violation proof for bin/release (release-entrypoint Wave 1).
# Seed the violation, watch the script refuse with a clear message, restore.
# State is normalized on entry and after every mutating case, so aborted runs
# cannot poison later cases. Refusals run before the happy path.
# The integration-line check is parameterised so the mechanical sequence is
# provable on the plan branch; the default (autonomous) check is proven by A2.
set -u
cd "$(dirname "$0")/.."   # repo root
pass=0; fail=0
BR=plan/release-entrypoint
BEFORE=$(git rev-parse HEAD)

restore() {  # back to BEFORE: undo release commits/tags, restore version.json
  git tag -l 'v0.*' | while read -r t; do git tag -d "$t" >/dev/null; done
  git reset -q --hard "$BEFORE"
  rm -rf .dojo/scratch
}
restore   # normalize: earlier aborted runs can leave stale tags or bumps

next_tag() { # the tag bin/release would compute from version.json right now
  cur=$(sed -n 's/.*"version"[[:space:]]*:[[:space:]]*"\([0-9]*\.[0-9]*\.[0-9]*\)".*/\1/p' version.json)
  echo "v${cur%.*}.$(( ${cur##*.} + 1 ))"
}
expect_refuse() { # name message_part
  local name="$1" msg_part="$2"
  out=$(KUMITE_INTEGRATION_BRANCH=$BR bin/release patch 2>&1); rc=$?
  if [ $rc -ne 0 ] && echo "$out" | grep -q "REFUSED" && echo "$out" | grep -q "$msg_part"; then
    ok "$name (rc=$rc, message names the cause)"
  else
    bad "$name — rc=$rc, output: $out"
  fi
}
ok()  { echo "  ok: $1"; pass=$((pass+1)); }
bad() { echo "  FAIL: $1"; fail=$((fail+1)); }

echo "A1 dirty tree (uncommitted source at repo root):"
echo dirt > uncommitted-source.tmp
expect_refuse "A1" "dirty"
rm uncommitted-source.tmp

echo "A2 wrong branch (default integration line 'autonomous'):"
out=$(bin/release patch 2>&1); rc=$?
if [ $rc -ne 0 ] && echo "$out" | grep -q "REFUSED" && echo "$out" | grep -q "'autonomous'"; then
  ok "A2 (running off the integration line refused, default named)"
else
  bad "A2 — rc=$rc, output: $out"
fi

echo "A3 stale proof (tree_sha256 forged to a foreign tree):"
bash scripts/dojo-check.sh >/dev/null 2>&1
cp .dojo/proof/check-proof .dojo/proof/check-proof.bak
sed -i 's/^tree_sha256=.*/tree_sha256=0000dead0000/' .dojo/proof/check-proof
expect_refuse "A3" "exact tree"
mv .dojo/proof/check-proof.bak .dojo/proof/check-proof

echo "A4 tag for the computed version already exists:"
git tag "$(next_tag)" 2>/dev/null || true
expect_refuse "A4" "already exists"
restore

echo "A5 bad level:"
out=$(bin/release minorplus 2>&1); rc=$?
[ $rc -eq 2 ] && echo "$out" | grep -q "unknown level" && ok "A5 (rc=2, usage error)" || bad "A5 — rc=$rc: $out"
out=$(bin/release 2>&1); rc=$?
[ $rc -eq 2 ] && echo "$out" | grep -q "usage" && ok "A5b no args (rc=2)" || bad "A5b — rc=$rc: $out"

echo "A6 missing proof:"
mv .dojo/proof/check-proof .dojo/proof/check-proof.bak
expect_refuse "A6" "no proof artifact"
mv .dojo/proof/check-proof.bak .dojo/proof/check-proof

echo "A6b non-passing proof:"
cp .dojo/proof/check-proof .dojo/proof/check-proof.bak
sed -i 's/^exit=0$/exit=1/' .dojo/proof/check-proof
expect_refuse "A6b" "did not pass"
mv .dojo/proof/check-proof.bak .dojo/proof/check-proof

echo "A7 happy path (local only, scratch artifact, transient tag/commit undone):"
bash scripts/dojo-check.sh >/dev/null 2>&1   # fresh proof over this exact tree
KUMITE_ARTIFACT_OUT=.dojo/scratch/kumite-test.exe KUMITE_INTEGRATION_BRANCH=$BR bin/release patch > /tmp/rel-out.txt 2>&1
rc=$?
if grep -q "version  0.1.0 -> 0.1.1" /tmp/rel-out.txt \
  && git rev-parse -q --verify refs/tags/v0.1.1 >/dev/null \
  && grep -q '"0.1.1"' version.json \
  && [ -f .dojo/scratch/kumite-test.exe ] \
  && grep -q "NOT pushed" /tmp/rel-out.txt; then
  ok "A7 (bump+tag+artifact+facts printed, nothing published)"
else
  bad "A7 — rc=$rc: $(cat /tmp/rel-out.txt)"
fi
restore

echo "A8 stale proof blocks the happy path (stale green is not a gate):"
bash scripts/dojo-check.sh >/dev/null 2>&1
sed -i 's/^tree_sha256=.*/tree_sha256=0000dead0000/' .dojo/proof/check-proof
out=$(KUMITE_INTEGRATION_BRANCH=$BR KUMITE_ARTIFACT_OUT=.dojo/scratch/t.exe bin/release patch 2>&1); rc=$?
[ $rc -ne 0 ] && echo "$out" | grep -q "exact tree" && ok "A8" || bad "A8 — rc=$rc: $out"
restore

echo
echo "asserts: $pass passed, $fail failed"
[ $fail -eq 0 ]