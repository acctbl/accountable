terraform {
  required_version = ">= 1.10.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
  }

  backend "s3" {
    bucket              = "accountable-tofu-state-576329195663-eu-west-2"
    key                 = "bootstrap/state.tfstate"
    region              = "eu-west-2"
    encrypt             = true
    kms_key_id          = "arn:aws:kms:eu-west-2:576329195663:alias/accountable-production-tofu-state"
    use_lockfile        = true
    allowed_account_ids = ["576329195663"]
  }
}

provider "aws" {
  region              = "eu-west-2"
  allowed_account_ids = ["576329195663"]

  default_tags {
    tags = {
      Environment = "production"
      ManagedBy   = "opentofu"
      Project     = "accountable"
    }
  }
}

module "state" {
  source = "../../../modules/state-backend"

  account_id  = "576329195663"
  environment = "production"
  region      = "eu-west-2"
}
