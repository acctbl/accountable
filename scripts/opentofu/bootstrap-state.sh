#!/usr/bin/env bash
set -euo pipefail

if [[ "$#" -ne 3 ]]; then
	echo "usage: $0 <environment> <aws-profile> <account-id>" >&2
	exit 2
fi

environment="$1"
aws_profile="$2"
account_id="$3"
region="eu-west-2"

case "$environment" in
	organization | development | staging | production | security-backup) ;;
	*)
		echo "unsupported environment: $environment" >&2
		exit 2
		;;
esac

if [[ ! "$account_id" =~ ^[0-9]{12}$ ]]; then
	echo "account ID must contain 12 digits" >&2
	exit 2
fi

confirmation="${environment}:${account_id}"
if [[ "${ACCOUNTABLE_CONFIRM_BOOTSTRAP:-}" != "$confirmation" ]]; then
	echo "set ACCOUNTABLE_CONFIRM_BOOTSTRAP=$confirmation to approve this bootstrap" >&2
	exit 2
fi

repo_root="$(cd "$(dirname "$0")/../.." && pwd)"
source_root="$repo_root/infra/opentofu"
source_environment_root="$source_root/environments/$environment/bootstrap"
state_bucket="accountable-tofu-state-${account_id}-${region}"
state_key_alias="alias/accountable-${environment}-tofu-state"

if [[ ! -d "$source_environment_root" ]]; then
	echo "bootstrap root does not exist: $source_environment_root" >&2
	exit 2
fi

caller_account_id="$(aws sts get-caller-identity --profile "$aws_profile" --query Account --output text)"
if [[ "$caller_account_id" != "$account_id" ]]; then
	echo "authenticated account $caller_account_id does not match $account_id" >&2
	exit 1
fi

if aws s3api head-bucket --profile "$aws_profile" --bucket "$state_bucket" >/dev/null 2>&1; then
	echo "state bucket already exists: $state_bucket" >&2
	exit 1
fi

if aws kms describe-key --profile "$aws_profile" --region "$region" --key-id "$state_key_alias" >/dev/null 2>&1; then
	echo "state key alias already exists: $state_key_alias" >&2
	exit 1
fi

work_dir="$(mktemp -d "${TMPDIR:-/tmp}/accountable-state-bootstrap.XXXXXX")"
work_source="$work_dir/opentofu"
work_root="$work_source/environments/$environment/bootstrap"
remote_main="$work_dir/main.remote.tf"
completed=0

cleanup() {
	if [[ "$completed" == "1" ]]; then
		rm -rf -- "$work_dir"
	else
		echo "bootstrap recovery files retained at $work_dir" >&2
	fi
}
trap cleanup EXIT

mkdir -p "$work_source"
tar -C "$source_root" \
	--exclude=.terraform \
	--exclude='*/.terraform' \
	--exclude='*.tfstate' \
	--exclude='*.tfstate.*' \
	-cf - . | tar -C "$work_source" -xf -

cp "$work_root/main.tf" "$remote_main"
awk '
	!skipping && /backend "s3"/ { skipping = 1; depth = 1; next }
	skipping {
		line = $0
		depth += gsub(/\{/, "{", line) - gsub(/\}/, "}", line)
		if (depth == 0) skipping = 0
		next
	}
	{ print }
' "$work_root/main.tf" >"$work_root/main.local.tf"
mv "$work_root/main.local.tf" "$work_root/main.tf"
cp "$source_root/.terraform.lock.hcl" "$work_root/.terraform.lock.hcl"

export AWS_PROFILE="$aws_profile"
export TF_DATA_DIR="$work_dir/.terraform-data"

tofu -chdir="$work_root" init -backend=false -input=false -lockfile=readonly
tofu -chdir="$work_root" plan -input=false -lock=false -out="$work_dir/bootstrap.plan"
tofu -chdir="$work_root" apply -input=false "$work_dir/bootstrap.plan"

cp "$remote_main" "$work_root/main.tf"
tofu -chdir="$work_root" init -input=false -migrate-state -force-copy

if [[ -z "$(tofu -chdir="$work_root" state list)" ]]; then
	echo "remote bootstrap state is empty" >&2
	exit 1
fi

tofu -chdir="$work_root" plan -input=false -lock=false -detailed-exitcode
completed=1
