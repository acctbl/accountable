output "budget_arns" {
  value = {
    for name, budget in aws_budgets_budget.account : name => budget.arn
  }
}

output "cloudtrail_delegated_administrator_id" {
  value = aws_cloudtrail_organization_delegated_admin_account.security_backup.account_id
}
