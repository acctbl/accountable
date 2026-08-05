#!/usr/bin/env bash
set -euo pipefail

outputs="$(cat)"

read_output() {
	jq -er --arg name "$1" '.[$name].value | strings | select(length > 0)' <<<"$outputs"
}

load_balancer_arn="$(read_output api_load_balancer_arn)"
cluster_arn="$(read_output ecs_cluster_arn)"
service_name="$(read_output ecs_service_name)"
rds_identifier="$(read_output rds_instance_identifier)"
data_bucket="$(read_output data_bucket_name)"
web_bucket="$(read_output web_bucket_name)"

aws elbv2 describe-load-balancers --load-balancer-arns "$load_balancer_arn" --output json |
	jq -e '.LoadBalancers | length == 1 and .[0].Scheme == "internal"' >/dev/null

aws rds describe-db-instances --db-instance-identifier "$rds_identifier" --output json |
	jq -e '.DBInstances | length == 1 and .[0].MultiAZ and (.[0].PubliclyAccessible | not) and .[0].StorageEncrypted' >/dev/null

aws ecs describe-services --cluster "$cluster_arn" --services "$service_name" --output json |
	jq -e '.services | length == 1 and .[0].networkConfiguration.awsvpcConfiguration.assignPublicIp == "DISABLED"' >/dev/null

for bucket in "$data_bucket" "$web_bucket"; do
	aws s3api get-public-access-block --bucket "$bucket" --output json |
		jq -e '.PublicAccessBlockConfiguration | .BlockPublicAcls and .BlockPublicPolicy and .IgnorePublicAcls and .RestrictPublicBuckets' >/dev/null
	status="$(curl --silent --show-error --output /dev/null --write-out '%{http_code}' "https://$bucket.s3.eu-west-2.amazonaws.com/")"
	if [[ "$status" != "403" ]]; then
		echo "anonymous S3 access returned HTTP $status for $bucket" >&2
		exit 1
	fi
done

echo "live-cell-posture: pass"
