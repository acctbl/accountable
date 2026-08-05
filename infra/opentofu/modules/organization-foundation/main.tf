data "aws_caller_identity" "current" {}

resource "aws_cloudtrail_organization_delegated_admin_account" "security_backup" {
  account_id = var.security_backup_account_id

  lifecycle {
    precondition {
      condition     = data.aws_caller_identity.current.account_id == var.management_account_id
      error_message = "CloudTrail delegation must be managed from the AWS Organizations management account."
    }
  }
}

resource "aws_budgets_budget" "account" {
  for_each = var.account_budgets

  name         = "accountable-${each.key}"
  budget_type  = "COST"
  limit_amount = tostring(each.value.monthly_limit_usd)
  limit_unit   = "USD"
  time_unit    = "MONTHLY"

  cost_filter {
    name   = "LinkedAccount"
    values = [each.value.account_id]
  }

  notification {
    comparison_operator        = "GREATER_THAN"
    notification_type          = "ACTUAL"
    threshold                  = 50
    threshold_type             = "PERCENTAGE"
    subscriber_email_addresses = [var.budget_alert_email]
  }

  notification {
    comparison_operator        = "GREATER_THAN"
    notification_type          = "FORECASTED"
    threshold                  = 80
    threshold_type             = "PERCENTAGE"
    subscriber_email_addresses = [var.budget_alert_email]
  }

  notification {
    comparison_operator        = "GREATER_THAN"
    notification_type          = "ACTUAL"
    threshold                  = 100
    threshold_type             = "PERCENTAGE"
    subscriber_email_addresses = [var.budget_alert_email]
  }

  lifecycle {
    precondition {
      condition     = data.aws_caller_identity.current.account_id == var.management_account_id
      error_message = "Organization budgets must be managed from the AWS Organizations management account."
    }
  }
}
