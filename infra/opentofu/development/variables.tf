variable "aws_region" {
  description = "AWS region holding the development contract foundation"
  type        = string
  default     = "eu-west-2"
}

variable "aws_account_id" {
  description = "AWS development member account ID"
  type        = string

  validation {
    condition     = can(regex("^[0-9]{12}$", var.aws_account_id))
    error_message = "aws_account_id must be a 12-digit AWS account ID"
  }
}

variable "github_oidc_subject_prefix" {
  description = "Immutable GitHub repository subject prefix trusted by the development contract role"
  type        = string
  default     = "repo:acctbl@309473689/accountable@1318144297"
}

variable "github_environment" {
  description = "Protected GitHub environment trusted by the development contract role"
  type        = string
  default     = "development"
}

variable "budget_alert_email" {
  description = "Email address receiving development account budget alerts"
  type        = string
}

variable "monthly_budget_usd" {
  description = "Monthly development account budget in USD"
  type        = number
  default     = 10

  validation {
    condition     = var.monthly_budget_usd >= 1
    error_message = "monthly_budget_usd must be at least 1"
  }
}
