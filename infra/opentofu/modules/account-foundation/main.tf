data "aws_caller_identity" "current" {}

resource "aws_s3_account_public_access_block" "account" {
  account_id = var.account_id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true

  lifecycle {
    precondition {
      condition     = data.aws_caller_identity.current.account_id == var.account_id
      error_message = "Authenticated AWS account does not match the account foundation."
    }
  }
}
