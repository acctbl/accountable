terraform {
  required_version = ">= 1.10.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
  }

  backend "s3" {
    bucket              = "accountable-tofu-state-906543084690-eu-west-2"
    key                 = "foundations/organization.tfstate"
    region              = "eu-west-2"
    encrypt             = true
    kms_key_id          = "arn:aws:kms:eu-west-2:906543084690:alias/accountable-organization-tofu-state"
    use_lockfile        = true
    allowed_account_ids = ["906543084690"]
  }
}

provider "aws" {
  region              = "eu-west-2"
  allowed_account_ids = ["906543084690"]

  default_tags {
    tags = {
      Environment = "organization"
      ManagedBy   = "opentofu"
      Project     = "accountable"
    }
  }
}

module "organization" {
  source = "../../../modules/organization-foundation"

  budget_alert_email         = var.budget_alert_email
  management_account_id      = "906543084690"
  security_backup_account_id = "063280428550"
  account_budgets = {
    development = {
      account_id        = "453722413624"
      monthly_limit_usd = 50
    }
    management = {
      account_id        = "906543084690"
      monthly_limit_usd = 10
    }
    production = {
      account_id        = "576329195663"
      monthly_limit_usd = 10
    }
    security-backup = {
      account_id        = "063280428550"
      monthly_limit_usd = 15
    }
    staging = {
      account_id        = "723225039926"
      monthly_limit_usd = 10
    }
  }
}
