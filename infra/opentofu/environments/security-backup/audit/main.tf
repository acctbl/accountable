terraform {
  required_version = ">= 1.10.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
  }

  backend "s3" {
    bucket              = "accountable-tofu-state-063280428550-eu-west-2"
    key                 = "audit/organization.tfstate"
    region              = "eu-west-2"
    encrypt             = true
    kms_key_id          = "arn:aws:kms:eu-west-2:063280428550:alias/accountable-security-backup-tofu-state"
    use_lockfile        = true
    allowed_account_ids = ["063280428550"]
  }
}

provider "aws" {
  region              = "eu-west-2"
  allowed_account_ids = ["063280428550"]

  default_tags {
    tags = {
      Environment = "security-backup"
      ManagedBy   = "opentofu"
      Project     = "accountable"
    }
  }
}

provider "aws" {
  alias               = "management"
  region              = "eu-west-2"
  allowed_account_ids = ["906543084690"]

  assume_role {
    role_arn     = "arn:aws:iam::906543084690:role/accountable-organization-audit-trail"
    session_name = "accountable-organization-audit"
  }

  default_tags {
    tags = {
      Environment = "security-backup"
      ManagedBy   = "opentofu"
      Project     = "accountable"
    }
  }
}

module "audit" {
  source = "../../../modules/organization-audit"

  providers = {
    aws            = aws
    aws.management = aws.management
  }

  management_account_id      = "906543084690"
  organization_id            = "o-ov472p8q83"
  region                     = "eu-west-2"
  security_backup_account_id = "063280428550"
}
