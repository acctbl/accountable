variable "budget_alert_email" {
  description = "Email address receiving organization budget alerts"
  type        = string

  validation {
    condition     = can(regex("^[^@[:space:]]+@[^@[:space:]]+\\.[^@[:space:]]+$", var.budget_alert_email))
    error_message = "budget_alert_email must be an email address."
  }
}
