output "budget_arns" {
  value = module.organization.budget_arns
}

output "cloudtrail_delegated_administrator_id" {
  value = module.organization.cloudtrail_delegated_administrator_id
}

output "audit_trail_role_arn" {
  value = module.organization.audit_trail_role_arn
}
