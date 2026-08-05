terraform {
  required_version = ">= 1.10.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
  }

  backend "s3" {}
}

provider "aws" {
  region              = var.aws_region
  allowed_account_ids = [local.account_id]

  default_tags {
    tags = {
      Environment = "development"
      ManagedBy   = "opentofu"
      Project     = "accountable"
    }
  }
}

locals {
  account_id           = "453722413624"
  secure_bucket_name   = "accountable-development-contract-${local.account_id}-${var.aws_region}"
  insecure_bucket_name = "accountable-development-insecure-${local.account_id}-${var.aws_region}"
  github_oidc_subject  = "${var.github_oidc_subject_prefix}:environment:${var.github_environment}"
  contract_role_name   = "accountable-development-contract"
}
