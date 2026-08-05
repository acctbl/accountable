terraform {
  required_version = ">= 1.10.0"

  backend "s3" {
    bucket              = "accountable-tofu-state-453722413624-eu-west-2"
    key                 = "bootstrap/development.tfstate"
    region              = "eu-west-2"
    encrypt             = true
    kms_key_id          = "arn:aws:kms:eu-west-2:453722413624:alias/accountable-development-tofu-state"
    use_lockfile        = true
    allowed_account_ids = ["453722413624"]
  }
}
