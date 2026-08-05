variable "api_desired_count" {
  description = "Zero before preflight and one for the deployed proof window"
  type        = number
}

variable "api_machine_identity_id" {
  description = "Infisical AWS IAM machine identity used by the proof API"
  type        = string
}

variable "bootstrap_machine_identity_id" {
  description = "Infisical AWS IAM machine identity used by proof database bootstrap"
  type        = string
}

variable "configuration_revision" {
  description = "Immutable source revision embedded in runtime configuration"
  type        = string
}

variable "image_uri" {
  description = "Production image URI pinned to a sha256 manifest digest"
  type        = string
}

variable "infisical_project_id" {
  description = "Infisical project containing proof cell secrets"
  type        = string
}

variable "migrate_machine_identity_id" {
  description = "Infisical AWS IAM machine identity used by proof migrations"
  type        = string
}
