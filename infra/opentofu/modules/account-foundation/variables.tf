variable "account_id" {
  description = "AWS account that owns this environment foundation"
  type        = string

  validation {
    condition     = can(regex("^[0-9]{12}$", var.account_id))
    error_message = "account_id must be a 12-digit AWS account ID."
  }
}

variable "environment" {
  description = "Accountable environment represented by the AWS account"
  type        = string

  validation {
    condition     = contains(["development", "staging", "production", "security-backup"], var.environment)
    error_message = "environment must be development, staging, production, or security-backup."
  }
}
