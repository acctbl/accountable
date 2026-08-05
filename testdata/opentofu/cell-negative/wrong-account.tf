provider "aws" {
  allowed_account_ids = ["000000000000"]
}

provider "aws" {
  alias               = "global"
  allowed_account_ids = ["000000000000"]
}

module "cell" {
  account_id = "000000000000"
}
