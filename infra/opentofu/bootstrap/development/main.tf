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
  account_id        = "453722413624"
  state_bucket_name = "accountable-tofu-state-${local.account_id}-${var.aws_region}"
}
