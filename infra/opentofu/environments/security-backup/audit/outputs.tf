output "audit_bucket_name" {
  value = module.audit.audit_bucket_name
}

output "audit_key_arn" {
  value = module.audit.audit_key_arn
}

output "organization_trail_arn" {
  value = module.audit.organization_trail_arn
}
