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
    key                 = "bootstrap/state.tfstate"
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

module "state" {
  source = "../../../modules/state-backend"

  account_id  = "906543084690"
  environment = "organization"
  region      = "eu-west-2"
}
