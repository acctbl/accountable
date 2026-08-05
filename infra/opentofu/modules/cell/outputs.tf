output "api_security_group_id" {
  value = aws_security_group.api.id
}

output "api_load_balancer_arn" {
  value = aws_lb.api.arn
}

output "api_task_role_arn" {
  value = aws_iam_role.api.arn
}

output "application_subnet_ids" {
  value = [for subnet in aws_subnet.application : subnet.id]
}

output "bootstrap_security_group_id" {
  value = aws_security_group.bootstrap.id
}

output "bootstrap_task_role_arn" {
  value = aws_iam_role.bootstrap.arn
}

output "bootstrap_task_definition_arn" {
  value = aws_ecs_task_definition.bootstrap.arn
}

output "data_bucket_name" {
  value = aws_s3_bucket.data.bucket
}

output "cloudfront_distribution_id" {
  value = aws_cloudfront_distribution.cell.id
}

output "cloudfront_domain" {
  value = aws_cloudfront_distribution.cell.domain_name
}

output "ecs_cluster_arn" {
  value = aws_ecs_cluster.cell.arn
}

output "ecs_service_name" {
  value = aws_ecs_service.api.name
}

output "migrate_security_group_id" {
  value = aws_security_group.migrate.id
}

output "migrate_task_role_arn" {
  value = aws_iam_role.migrate.arn
}

output "migrate_task_definition_arn" {
  value = aws_ecs_task_definition.migrate.arn
}

output "preflight_task_definition_arn" {
  value = aws_ecs_task_definition.preflight.arn
}

output "rds_instance_identifier" {
  value = aws_db_instance.cell.identifier
}

output "web_bucket_name" {
  value = aws_s3_bucket.web.bucket
}
