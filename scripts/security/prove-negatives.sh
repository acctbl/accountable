#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
WORKTREE="$(mktemp -d "${TMPDIR:-/tmp}/accountable-security-negative.XXXXXX")"
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
	"$@" >/tmp/accountable-prove-security-"$name".out 2>&1
	local status=$?
	set -e
	if [[ "$status" -eq 0 ]]; then
		echo "FAIL: $name expected to fail but passed"
		cat /tmp/accountable-prove-security-"$name".out || true
		FAIL=$((FAIL + 1))
	else
		echo "PASS: $name failed as expected (exit $status)"
		PASS=$((PASS + 1))
	fi
}

echo "==> negative: gitleaks secret"
cp testdata/negative/security/fake-secret.py app_secret.py
git add app_secret.py
git -c user.email=prove@accountable.local -c user.name=prove \
	commit --no-gpg-sign -m "security-negative: planted secret" >/dev/null
prove gitleaks task security:gitleaks
git reset --soft HEAD~1 >/dev/null
git rm -f --cached app_secret.py >/dev/null
rm -f app_secret.py

echo "==> negative: govulncheck vulnerable module"
go get golang.org/x/text@v0.3.0 >/tmp/accountable-prove-security-govuln-setup.out 2>&1
cat > internal/server/z_negative_vuln.go <<'EOF'
package server

import _ "golang.org/x/text/language"
EOF
prove govulncheck task security:govulncheck
rm -f internal/server/z_negative_vuln.go
git checkout -- go.mod go.sum

echo "==> negative: trivy vulnerable dependency"
cp testdata/negative/security/vulnerable-requirements.txt requirements.txt
prove trivy-vuln task security:trivy
rm -f requirements.txt

echo "==> negative: trivy IaC misconfiguration"
cp testdata/negative/security/bad-infra.tf infra/opentofu/negative_open.tf
prove trivy-iac task security:trivy
rm -f infra/opentofu/negative_open.tf

echo "==> negative: floating lockfile tag"
cp testdata/negative/security/Dockerfile.latest Dockerfile
prove lockfile task security:lockfile
rm -f Dockerfile

echo "==> negative: SBOM with no package manifests"
EMPTY_SBOM="$(mktemp -d "${TMPDIR:-/tmp}/accountable-empty-sbom.XXXXXX")"
prove sbom env SBOM_TARGET="$EMPTY_SBOM" task security:sbom
rm -rf "$EMPTY_SBOM"

echo "==> config: CodeQL, Dependabot, and security workflow"
for path in \
	.github/workflows/codeql.yml \
	.github/workflows/security.yml \
	.github/dependabot.yml; do
	if [[ -f "$path" ]]; then
		echo "PASS: $path exists"
		PASS=$((PASS + 1))
	else
		echo "FAIL: $path missing"
		FAIL=$((FAIL + 1))
	fi
done

if grep -A4 'uses: actions/checkout@' .github/workflows/security.yml | grep -q 'fetch-depth: 0'; then
	echo "PASS: security workflow fetches complete Git history for gitleaks"
	PASS=$((PASS + 1))
else
	echo "FAIL: security workflow must set checkout fetch-depth to 0 for gitleaks"
	FAIL=$((FAIL + 1))
fi

if grep -q -- '--include-dev-deps' scripts/security/trivy.sh; then
	echo "PASS: trivy includes development dependencies"
	PASS=$((PASS + 1))
else
	echo "FAIL: trivy must include development dependencies"
	FAIL=$((FAIL + 1))
fi

echo
echo "Security negative proofs: $PASS passed, $FAIL failed"
if [[ "$FAIL" -ne 0 ]]; then
	exit 1
fi
