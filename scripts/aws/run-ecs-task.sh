#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 4 ]]; then
	echo "usage: run-ecs-task.sh <cluster-arn> <task-definition-arn> <subnet-csv> <security-group-id>" >&2
	exit 2
fi

cluster_arn="$1"
task_definition_arn="$2"
subnet_csv="$3"
security_group_id="$4"

response="$(aws ecs run-task \
	--cluster "$cluster_arn" \
	--task-definition "$task_definition_arn" \
	--launch-type FARGATE \
	--platform-version 1.4.0 \
	--network-configuration "awsvpcConfiguration={subnets=[$subnet_csv],securityGroups=[$security_group_id],assignPublicIp=DISABLED}" \
	--output json)"

failure_count="$(jq '.failures | length' <<<"$response")"
if [[ "$failure_count" != "0" ]]; then
	jq '.failures' <<<"$response" >&2
	exit 1
fi

task_arn="$(jq -r '.tasks[0].taskArn // empty' <<<"$response")"
if [[ -z "$task_arn" ]]; then
	echo "ECS returned no task ARN" >&2
	exit 1
fi

aws ecs wait tasks-stopped --cluster "$cluster_arn" --tasks "$task_arn"
task="$(aws ecs describe-tasks --cluster "$cluster_arn" --tasks "$task_arn" --output json)"
exit_code="$(jq -r '.tasks[0].containers[0].exitCode // empty' <<<"$task")"
if [[ "$exit_code" != "0" ]]; then
	jq '{stoppedReason: .tasks[0].stoppedReason, containers: .tasks[0].containers | map({name, exitCode, reason})}' <<<"$task" >&2
	exit 1
fi
