#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
WORKTREE="$(mktemp -d "${TMPDIR:-/tmp}/accountable-negative.XXXXXX")"
rmdir "$WORKTREE"

cleanup() {
	git worktree remove --force "$WORKTREE" 2>/dev/null || true
}
trap cleanup EXIT

git worktree add --detach "$WORKTREE" HEAD >/dev/null
if ! git diff --quiet HEAD; then
	git diff --binary HEAD | git -C "$WORKTREE" apply
fi

while IFS= read -r -d '' path; do
	mkdir -p "$WORKTREE/$(dirname "$path")"
	cp -p "$ROOT/$path" "$WORKTREE/$path"
done < <(git -C "$ROOT" ls-files --others --exclude-standard -z)

cd "$WORKTREE"

PASS=0
FAIL=0

prove() {
	local name="$1"
	shift
	set +e
	"$@" >/tmp/accountable-prove-"$name".out 2>&1
	local status=$?
	set -e
	if [[ "$status" -eq 0 ]]; then
		echo "FAIL: $name expected to fail but passed"
		cat /tmp/accountable-prove-"$name".out || true
		FAIL=$((FAIL + 1))
	else
		echo "PASS: $name failed as expected (exit $status)"
		PASS=$((PASS + 1))
	fi
}

echo "==> negative: dirty generated output"
TARGET="gen/go/accountable/common/v1/common.pb.go"
printf '\n// stale-for-negative-proof\n' >>"$TARGET"
prove generated-dirty task gen:fresh

echo "==> negative: prohibited import"
BAD_IMPORT="internal/apierror/z_negative_import.go"
cp "testdata/negative/arch/bad_import.go" "$BAD_IMPORT"
prove prohibited-import task lint
rm -f "$BAD_IMPORT"

echo "==> negative: http.ListenAndServe"
BAD_LISTEN="internal/server/z_negative_listen.go"
cp "testdata/negative/arch/bad_listen_and_serve.go" "$BAD_LISTEN"
prove forbidden-listen-and-serve task lint
rm -f "$BAD_LISTEN"

echo "==> negative: float64"
BAD_FLOAT="internal/server/z_negative_float.go"
cp "testdata/negative/arch/bad_float.go" "$BAD_FLOAT"
prove forbidden-float task lint
rm -f "$BAD_FLOAT"

echo "==> negative: domain imports net/http"
BAD_DOMAIN_DIR="internal/modules/negativetest/domain"
mkdir -p "$BAD_DOMAIN_DIR"
cp "testdata/negative/arch/bad_domain_http_import.go" "$BAD_DOMAIN_DIR/z_negative_http.go"
prove domain-http-import task lint
rm -rf "internal/modules/negativetest"

echo "==> negative: time.Now outside clock adapter"
BAD_TIME_NOW="internal/server/z_negative_time_now.go"
cp "testdata/negative/arch/bad_time_now.go" "$BAD_TIME_NOW"
prove forbidden-time-now task lint
rm -f "$BAD_TIME_NOW"

echo "==> negative: timestamppb.Now"
BAD_TSPB_NOW="internal/server/z_negative_timestamppb_now.go"
cp "testdata/negative/arch/bad_timestamppb_now.go" "$BAD_TSPB_NOW"
prove forbidden-timestamppb-now task lint
rm -f "$BAD_TSPB_NOW"

echo "==> negative: invalid migration"
BAD_MIGRATION="db/migrations/99999_bad.sql"
cp "testdata/negative/migrate/00001_bad.sql" "$BAD_MIGRATION"
prove invalid-migration task migrate

echo "==> negative: missing translation"
AR="apps/web/src/i18n/messages/ar.json"
node -e '
const fs = require("node:fs");
const path = process.argv[1];
const catalog = JSON.parse(fs.readFileSync(path, "utf8"));
delete catalog["home.title"];
fs.writeFileSync(path, JSON.stringify(catalog, null, 2) + "\n");
' "$AR"
prove missing-translation task i18n

echo "==> negative: accessibility defect"
PAGE="apps/web/src/routes/index.tsx"
node -e '
const fs = require("node:fs");
const path = process.argv[1];
let source = fs.readFileSync(path, "utf8");
source = source.replace(
  "<h1 className=\"font-medium\">",
  "<img src=\"/favicon.svg\" />\\n\\t\\t\\t\\t\\t<h1 className=\"font-medium\">",
);
fs.writeFileSync(path, source);
' "$PAGE"
prove accessibility-defect env CI=1 task a11y

echo
echo "Negative proofs: $PASS passed, $FAIL failed"
if [[ "$FAIL" -ne 0 ]]; then
	exit 1
fi
