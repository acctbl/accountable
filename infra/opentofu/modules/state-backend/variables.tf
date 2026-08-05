variable "account_id" {
  type = string

  validation {
    condition     = can(regex("^[0-9]{12}$", var.account_id))
    error_message = "account_id must be a 12-digit AWS account ID."
  }
}

variable "environment" {
  type = string

  validation {
    condition     = contains(["organization", "development", "staging", "production", "security-backup"], var.environment)
    error_message = "environment must identify an Accountable AWS account."
  }
}

variable "region" {
  type = string

  validation {
    condition     = can(regex("^[a-z]{2}(?:-gov)?-[a-z]+-[0-9]$", var.region))
    error_message = "region must be an AWS region."
  }
}
