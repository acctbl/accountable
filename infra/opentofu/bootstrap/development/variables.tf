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
