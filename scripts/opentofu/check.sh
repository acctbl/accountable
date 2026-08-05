#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "$0")/../.." && pwd)"
source_root="$repo_root/infra/opentofu"
validation_root="$(mktemp -d "${TMPDIR:-/tmp}/accountable-tofu-check.XXXXXX")"
provider_cache="${TMPDIR:-/tmp}/accountable-opentofu-provider-cache"
mkdir -p "$provider_cache"
export TF_PLUGIN_CACHE_DIR="$provider_cache"
cleanup() { rm -rf "$validation_root"; }
trap cleanup EXIT

tar -C "$source_root" \
	--exclude=.terraform \
	--exclude='*/.terraform' \
	--exclude='*.tfstate' \
	--exclude='*.tfstate.*' \
	-cf - . | tar -C "$validation_root" -xf -

(
	cd "$validation_root"
	tofu init -backend=false -input=false
	tofu validate
)

plugin_dir="$validation_root/.terraform/providers"
roots=(
	environments/development/bootstrap
	environments/development/contracts
	environments/development/foundation
	environments/development/cells/development-01
	environments/development/cells/proof-01
	environments/staging/bootstrap
	environments/staging/foundation
	environments/production/bootstrap
	environments/production/foundation
	environments/security-backup/bootstrap
	environments/security-backup/foundation
	environments/security-backup/audit
	environments/organization/bootstrap
	environments/organization/foundation
	migrations/development-legacy/bootstrap
	migrations/development-legacy/combined
	modules/account-foundation
	modules/environment-foundation
	modules/github-deployment
	modules/cell
	modules/managed-contract
	modules/organization-foundation
	modules/organization-audit
	modules/state-backend
)

for root in "${roots[@]}"; do
	(
		cd "$validation_root/$root"
		has_required_providers=0
		while IFS= read -r -d '' file; do
			if grep -q 'required_providers' "$file"; then
				has_required_providers=1
				break
			fi
		done < <(find . -type f -name '*.tf' -print0)
		if [[ "$has_required_providers" == "1" ]]; then
			cp "$validation_root/.terraform.lock.hcl" .terraform.lock.hcl
			TF_DATA_DIR="$validation_root/.terraform-data/$root" tofu init \
				-backend=false -input=false -lockfile=readonly -plugin-dir="$plugin_dir"
		else
			TF_DATA_DIR="$validation_root/.terraform-data/$root" tofu init \
				-backend=false -input=false
		fi
		TF_DATA_DIR="$validation_root/.terraform-data/$root" tofu validate
		if [[ -d tests ]]; then
			TF_DATA_DIR="$validation_root/.terraform-data/$root" tofu test
		fi
	)
done
