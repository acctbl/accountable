mock_provider "aws" {
  alias = "mock"

  mock_data "aws_caller_identity" {
    defaults = {
      account_id = "453722413624"
    }
  }
}

variables {
  account_id  = "453722413624"
  environment = "development"
}

run "account_public_access_is_fail_closed" {
  command = plan

  providers = {
    aws = aws.mock
  }

  assert {
    condition = (
      aws_s3_account_public_access_block.account.block_public_acls &&
      aws_s3_account_public_access_block.account.block_public_policy &&
      aws_s3_account_public_access_block.account.ignore_public_acls &&
      aws_s3_account_public_access_block.account.restrict_public_buckets
    )
    error_message = "The account foundation must enable every S3 public-access control."
  }
}

run "wrong_account_is_rejected" {
  command = plan

  providers = {
    aws = aws.mock
  }

  variables {
    account_id = "000000000000"
  }

  expect_failures = [aws_s3_account_public_access_block.account]
}
