#!/usr/bin/env bash
set -euo pipefail

# One-time migration from the two retired development roots into the new
# bootstrap, foundation, and managed-contract state boundaries. This changes
# state ownership only; it never applies AWS resource changes.

umask 077

expected_account_id="453722413624"
region="eu-west-2"
state_bucket="accountable-tofu-state-${expected_account_id}-${region}"
repo_root="$(cd "$(dirname "$0")/../.." && pwd)"

if [[ "${ACCOUNTABLE_CONFIRM_STATE_MIGRATION:-}" != "development-453722413624" ]]; then
	echo "Refusing state migration without ACCOUNTABLE_CONFIRM_STATE_MIGRATION=development-453722413624" >&2
	exit 1
fi

for command in aws jq tofu; do
	if ! command -v "$command" >/dev/null 2>&1; then
		echo "$command is required" >&2
		exit 1
	fi
done

actual_account_id="$(aws sts get-caller-identity --query Account --output text)"
if [[ "$actual_account_id" != "$expected_account_id" ]]; then
	echo "Authenticated AWS account is $actual_account_id; expected $expected_account_id" >&2
	exit 1
fi

work_dir="$(mktemp -d "${TMPDIR:-/tmp}/accountable-state-migration.XXXXXX")"
echo "State snapshots and automatic backups will remain at: $work_dir"

object_exists() {
	local key="$1"
	local listing
	if ! listing="$(aws s3api list-objects-v2 --bucket "$state_bucket" --prefix "$key" --output json)"; then
		echo "Unable to inspect s3://$state_bucket/$key" >&2
		exit 1
	fi
	jq -e --arg key "$key" 'any(.Contents[]?; .Key == $key)' <<<"$listing" >/dev/null
}

legacy_bootstrap_key="bootstrap/development.tfstate"
legacy_development_key="development/foundation.tfstate"
new_bootstrap_key="bootstrap/state.tfstate"
new_foundation_key="foundations/account.tfstate"
new_contracts_key="contracts/managed-adapters.tfstate"
source_keys=("$legacy_bootstrap_key" "$legacy_development_key")
destination_keys=("$new_bootstrap_key" "$new_foundation_key" "$new_contracts_key")

for key in "${source_keys[@]}"; do
	if ! object_exists "$key"; then
		echo "Required legacy state does not exist: s3://$state_bucket/$key" >&2
		exit 1
	fi
done

for key in "${destination_keys[@]}"; do
	if object_exists "$key"; then
		echo "Destination state already exists; refusing to overwrite s3://$state_bucket/$key" >&2
		exit 1
	fi
done

for key in "${source_keys[@]}" "${destination_keys[@]}"; do
	if object_exists "${key}.tflock"; then
		echo "State lock exists; refusing migration while s3://$state_bucket/${key}.tflock is present" >&2
		exit 1
	fi
done

# Download independent, versioned recovery copies before OpenTofu rewrites any
# local state or publishes any destination state.
for key in "${source_keys[@]}"; do
	backup_path="$work_dir/original-${key//\//-}"
	aws s3api get-object --bucket "$state_bucket" --key "$key" "$backup_path" >/dev/null
	if ! jq -e '(.version | type == "number") and (.serial | type == "number") and (.lineage | type == "string")' "$backup_path" >/dev/null; then
		echo "Downloaded backup is not a valid OpenTofu state: $backup_path" >&2
		exit 1
	fi
done

legacy_bootstrap_root="$repo_root/infra/opentofu/migrations/development-legacy/bootstrap"
legacy_development_root="$repo_root/infra/opentofu/migrations/development-legacy/combined"
TF_DATA_DIR="$work_dir/data/legacy-bootstrap" tofu -chdir="$legacy_bootstrap_root" init -input=false -reconfigure >/dev/null
TF_DATA_DIR="$work_dir/data/legacy-development" tofu -chdir="$legacy_development_root" init -input=false -reconfigure >/dev/null

new_bootstrap_root="$repo_root/infra/opentofu/environments/development/bootstrap"
new_foundation_root="$repo_root/infra/opentofu/environments/development/foundation"
new_contracts_root="$repo_root/infra/opentofu/environments/development/contracts"

for root in "$new_bootstrap_root" "$new_foundation_root" "$new_contracts_root"; do
	TF_DATA_DIR="$work_dir/data/$(basename "$root")" tofu -chdir="$root" init -input=false -reconfigure >/dev/null
done

legacy_bootstrap_state="$work_dir/legacy-bootstrap.tfstate"
legacy_development_state="$work_dir/legacy-development.tfstate"
bootstrap_state="$work_dir/bootstrap.tfstate"
foundation_state="$work_dir/foundation.tfstate"
contracts_state="$work_dir/contracts.tfstate"

TF_DATA_DIR="$work_dir/data/legacy-bootstrap" tofu -chdir="$legacy_bootstrap_root" state pull >"$legacy_bootstrap_state"
TF_DATA_DIR="$work_dir/data/legacy-development" tofu -chdir="$legacy_development_root" state pull >"$legacy_development_state"

move_state() {
	local source_state="$1"
	local destination_state="$2"
	local source_address="$3"
	local destination_address="$4"
	tofu state mv \
		-state="$source_state" \
		-state-out="$destination_state" \
		"$source_address" "$destination_address"
}

bootstrap_addresses=(
	aws_kms_key.state
	aws_kms_alias.state
	aws_s3_bucket.state
	aws_s3_bucket_ownership_controls.state
	aws_s3_bucket_public_access_block.state
	aws_s3_bucket_versioning.state
	aws_s3_bucket_server_side_encryption_configuration.state
	aws_s3_bucket_policy.state
)
for address in "${bootstrap_addresses[@]}"; do
	move_state "$legacy_bootstrap_state" "$bootstrap_state" "$address" "module.state.$address"
done

contract_addresses=(
	aws_kms_key.storage
	aws_kms_alias.storage
	aws_kms_key.crypto
	aws_kms_alias.crypto
	'aws_s3_bucket.contract["secure"]'
	'aws_s3_bucket.contract["insecure"]'
	'aws_s3_bucket_ownership_controls.contract["secure"]'
	'aws_s3_bucket_ownership_controls.contract["insecure"]'
	'aws_s3_bucket_public_access_block.contract["secure"]'
	'aws_s3_bucket_public_access_block.contract["insecure"]'
	'aws_s3_bucket_server_side_encryption_configuration.contract["secure"]'
	'aws_s3_bucket_server_side_encryption_configuration.contract["insecure"]'
	'aws_s3_bucket_policy.contract["secure"]'
	'aws_s3_bucket_policy.contract["insecure"]'
	aws_iam_role.contract
	aws_iam_role_policy.contract
)
for address in "${contract_addresses[@]}"; do
	move_state "$legacy_development_state" "$contracts_state" "$address" "module.managed_contract.$address"
done

move_state \
	"$legacy_development_state" "$foundation_state" \
	aws_budgets_budget.development \
	module.environment.module.account.aws_budgets_budget.account
move_state \
	"$legacy_development_state" "$foundation_state" \
	aws_iam_openid_connect_provider.github \
	module.environment.module.github.aws_iam_openid_connect_provider.github

# Data sources do not own remote objects and are deliberately refreshed in the
# new roots instead of being carried between states.
for state in "$legacy_bootstrap_state" "$legacy_development_state"; do
	while IFS= read -r address; do
		[[ -z "$address" ]] && continue
		if [[ "$address" != data.* ]]; then
			echo "Unexpected managed address remains in legacy state: $address" >&2
			exit 1
		fi
		tofu state rm -state="$state" "$address"
	done < <(tofu state list -state="$state")
done

for state in "$legacy_bootstrap_state" "$legacy_development_state"; do
	if [[ -n "$(tofu state list -state="$state")" ]]; then
		echo "Legacy state is not empty after splitting: $state" >&2
		exit 1
	fi
done

# Publish destination ownership first. The retired keys are emptied last so a
# failed command cannot leave a resource unowned. Do not run any OpenTofu apply
# concurrently with this migration.
TF_DATA_DIR="$work_dir/data/bootstrap" tofu -chdir="$new_bootstrap_root" state push "$bootstrap_state"
TF_DATA_DIR="$work_dir/data/foundation" tofu -chdir="$new_foundation_root" state push "$foundation_state"
TF_DATA_DIR="$work_dir/data/contracts" tofu -chdir="$new_contracts_root" state push "$contracts_state"
TF_DATA_DIR="$work_dir/data/legacy-bootstrap" tofu -chdir="$legacy_bootstrap_root" state push "$legacy_bootstrap_state"
TF_DATA_DIR="$work_dir/data/legacy-development" tofu -chdir="$legacy_development_root" state push "$legacy_development_state"

echo "State ownership migrated. Review saved plans in each new root before any apply."
