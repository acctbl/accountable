mock_provider "aws" {
  alias = "mock"

  mock_data "aws_caller_identity" {
    defaults = {
      account_id = "063280428550"
    }
  }

  mock_data "aws_iam_policy_document" {
    defaults = {
      json = "{\"Version\":\"2012-10-17\",\"Statement\":[]}"
    }
  }

  mock_resource "aws_kms_key" {
    defaults = {
      arn = "arn:aws:kms:eu-west-2:063280428550:key/00000000-0000-0000-0000-000000000000"
    }
  }
}

variables {
  management_account_id      = "906543084690"
  organization_id            = "o-ov472p8q83"
  region                     = "eu-west-2"
  security_backup_account_id = "063280428550"
}

run "organization_evidence_is_durable_and_validated" {
  command = plan

  providers = {
    aws = aws.mock
  }

  assert {
    condition = (
      aws_cloudtrail.organization.enable_log_file_validation &&
      aws_cloudtrail.organization.enable_logging &&
      aws_cloudtrail.organization.include_global_service_events &&
      aws_cloudtrail.organization.is_multi_region_trail &&
      aws_cloudtrail.organization.is_organization_trail
    )
    error_message = "CloudTrail must retain validated management events for every account and region."
  }

  assert {
    condition = (
      aws_s3_bucket_public_access_block.audit.block_public_acls &&
      aws_s3_bucket_public_access_block.audit.block_public_policy &&
      aws_s3_bucket_public_access_block.audit.ignore_public_acls &&
      aws_s3_bucket_public_access_block.audit.restrict_public_buckets &&
      aws_s3_bucket_versioning.audit.versioning_configuration[0].status == "Enabled" &&
      aws_kms_key.audit.enable_key_rotation
    )
    error_message = "Organization evidence storage must be private, versioned, and protected by a rotating KMS key."
  }
}

run "wrong_security_account_is_rejected" {
  command = plan

  providers = {
    aws = aws.mock
  }

  variables {
    security_backup_account_id = "000000000000"
  }

  expect_failures = [aws_kms_key.audit]
}
