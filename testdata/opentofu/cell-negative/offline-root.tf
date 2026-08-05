terraform {
  backend "local" {}
}

provider "aws" {
  skip_credentials_validation = true
  skip_metadata_api_check     = true
  skip_region_validation      = true
  skip_requesting_account_id  = false

  endpoints {
    iam = "OFFLINE_AWS_ENDPOINT"
    sts = "OFFLINE_AWS_ENDPOINT"
  }
}

provider "aws" {
  alias = "global"

  skip_credentials_validation = true
  skip_metadata_api_check     = true
  skip_region_validation      = true
  skip_requesting_account_id  = false

  endpoints {
    iam = "OFFLINE_AWS_ENDPOINT"
    sts = "OFFLINE_AWS_ENDPOINT"
  }
}
