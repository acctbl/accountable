variable "budget_alert_email" {
  description = "Email address receiving organization budget alerts"
  type        = string

  validation {
    condition     = can(regex("^[^@[:space:]]+@[^@[:space:]]+\\.[^@[:space:]]+$", var.budget_alert_email))
    error_message = "budget_alert_email must be an email address."
  }
}

variable "management_account_id" {
  description = "AWS Organizations management account"
  type        = string

  validation {
    condition     = can(regex("^[0-9]{12}$", var.management_account_id))
    error_message = "management_account_id must be a 12-digit AWS account ID."
  }
}

variable "security_backup_account_id" {
  description = "Account delegated to manage organization CloudTrail"
  type        = string

  validation {
    condition     = can(regex("^[0-9]{12}$", var.security_backup_account_id))
    error_message = "security_backup_account_id must be a 12-digit AWS account ID."
  }
}

variable "account_budgets" {
  description = "Monthly alarm ceilings for every organization account"
  type = map(object({
    account_id        = string
    monthly_limit_usd = number
  }))

  validation {
    condition = alltrue([
      for budget in values(var.account_budgets) :
      can(regex("^[0-9]{12}$", budget.account_id)) && budget.monthly_limit_usd >= 1
    ])
    error_message = "Every account budget needs a 12-digit account ID and a monthly limit of at least 1 USD."
  }
}
