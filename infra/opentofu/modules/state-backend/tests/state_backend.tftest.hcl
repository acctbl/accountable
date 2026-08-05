mock_provider "aws" {
  mock_data "aws_caller_identity" {
    defaults = {
      account_id = "453722413624"
    }
  }
}

variables {
  account_id  = "453722413624"
  environment = "development"
  region      = "eu-west-2"
}

run "state_is_recoverable_and_private" {
  command = plan

  assert {
    condition     = aws_s3_bucket_versioning.state.versioning_configuration[0].status == "Enabled"
    error_message = "State versioning must remain enabled."
  }

  assert {
    condition = alltrue([
      aws_s3_bucket_public_access_block.state.block_public_acls,
      aws_s3_bucket_public_access_block.state.block_public_policy,
      aws_s3_bucket_public_access_block.state.ignore_public_acls,
      aws_s3_bucket_public_access_block.state.restrict_public_buckets,
    ])
    error_message = "State must remain private."
  }
}

run "wrong_account_fails" {
  command = plan

  variables {
    account_id = "723225039926"
  }

  expect_failures = [aws_kms_key.state]
}
