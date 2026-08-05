output "account_id" {
  value = module.environment.account_id
}

output "environment" {
  value = module.environment.environment
}

output "image_repository_arn" {
  value = module.environment.image_repository_arn
}

output "image_repository_url" {
  value = module.environment.image_repository_url
}

output "apply_role_arn" {
  value = module.environment.apply_role_arn
}

output "plan_role_arn" {
  value = module.environment.plan_role_arn
}
