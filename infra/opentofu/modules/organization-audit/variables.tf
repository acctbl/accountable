variable "management_account_id" {
  type = string

  validation {
    condition     = can(regex("^[0-9]{12}$", var.management_account_id))
    error_message = "management_account_id must be a 12-digit AWS account ID."
  }
}

variable "organization_id" {
  type = string

  validation {
    condition     = can(regex("^o-[a-z0-9]{10,32}$", var.organization_id))
    error_message = "organization_id must be an AWS Organizations ID."
  }
}

variable "region" {
  type = string

  validation {
    condition     = can(regex("^[a-z]{2}(?:-gov)?-[a-z]+-[0-9]$", var.region))
    error_message = "region must be an AWS region."
  }
}

variable "security_backup_account_id" {
  type = string

  validation {
    condition     = can(regex("^[0-9]{12}$", var.security_backup_account_id))
    error_message = "security_backup_account_id must be a 12-digit AWS account ID."
  }
}
