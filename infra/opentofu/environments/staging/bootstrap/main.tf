terraform {
  required_version = ">= 1.10.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
  }

  backend "s3" {
    bucket              = "accountable-tofu-state-723225039926-eu-west-2"
    key                 = "bootstrap/state.tfstate"
    region              = "eu-west-2"
    encrypt             = true
    kms_key_id          = "arn:aws:kms:eu-west-2:723225039926:alias/accountable-staging-tofu-state"
    use_lockfile        = true
    allowed_account_ids = ["723225039926"]
  }
}

provider "aws" {
  region              = "eu-west-2"
  allowed_account_ids = ["723225039926"]

  default_tags {
    tags = {
      Environment = "staging"
      ManagedBy   = "opentofu"
      Project     = "accountable"
    }
  }
}

module "state" {
  source = "../../../modules/state-backend"

  account_id  = "723225039926"
  environment = "staging"
  region      = "eu-west-2"
}
