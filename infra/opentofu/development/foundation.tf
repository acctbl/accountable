resource "aws_kms_key" "storage" {
  description             = "Encrypts Accountable development contract objects"
  deletion_window_in_days = 30
  enable_key_rotation     = true

  tags = {
    Purpose = "contract-storage"
  }
}

resource "aws_kms_alias" "storage" {
  name          = "alias/accountable-development-storage"
  target_key_id = aws_kms_key.storage.key_id
}

resource "aws_kms_key" "crypto" {
  description             = "Proves Accountable application encryption against AWS KMS"
  deletion_window_in_days = 30
  enable_key_rotation     = true

  tags = {
    Purpose = "contract-crypto"
  }
}

resource "aws_kms_alias" "crypto" {
  name          = "alias/accountable-development-crypto"
  target_key_id = aws_kms_key.crypto.key_id
}

locals {
  contract_buckets = {
    secure   = local.secure_bucket_name
    insecure = local.insecure_bucket_name
  }
}

resource "aws_s3_bucket" "contract" {
  for_each = local.contract_buckets
  bucket   = each.value
}

resource "aws_s3_bucket_ownership_controls" "contract" {
  for_each = aws_s3_bucket.contract
  bucket   = each.value.id

  rule {
    object_ownership = "BucketOwnerEnforced"
  }
}

# The insecure instance exists only to prove the adapter refuses a bucket when any public-access guard is off.
#trivy:ignore:AVD-AWS-0086:exp:2026-11-05 trivy:ignore:AVD-AWS-0087:exp:2026-11-05 trivy:ignore:AVD-AWS-0091:exp:2026-11-05 trivy:ignore:AVD-AWS-0093:exp:2026-11-05
resource "aws_s3_bucket_public_access_block" "contract" {
  for_each = aws_s3_bucket.contract
  bucket   = each.value.id

  block_public_acls       = each.key == "secure"
  block_public_policy     = each.key == "secure"
  ignore_public_acls      = each.key == "secure"
  restrict_public_buckets = each.key == "secure"
}

resource "aws_s3_bucket_server_side_encryption_configuration" "contract" {
  for_each = aws_s3_bucket.contract
  bucket   = each.value.id

  rule {
    apply_server_side_encryption_by_default {
      kms_master_key_id = aws_kms_key.storage.arn
      sse_algorithm     = "aws:kms"
    }

    bucket_key_enabled = true
  }
}

data "aws_iam_policy_document" "contract_bucket" {
  for_each = aws_s3_bucket.contract

  statement {
    sid     = "RequireTLS"
    effect  = "Deny"
    actions = ["s3:*"]
    resources = [
      each.value.arn,
      "${each.value.arn}/*",
    ]

    principals {
      type        = "AWS"
      identifiers = ["*"]
    }

    condition {
      test     = "Bool"
      variable = "aws:SecureTransport"
      values   = ["false"]
    }
  }
}

resource "aws_s3_bucket_policy" "contract" {
  for_each = aws_s3_bucket.contract
  bucket   = each.value.id
  policy   = data.aws_iam_policy_document.contract_bucket[each.key].json
}

resource "aws_budgets_budget" "development" {
  name         = "accountable-development"
  budget_type  = "COST"
  limit_amount = tostring(var.monthly_budget_usd)
  limit_unit   = "USD"
  time_unit    = "MONTHLY"

  notification {
    comparison_operator        = "GREATER_THAN"
    notification_type          = "FORECASTED"
    threshold                  = 80
    threshold_type             = "PERCENTAGE"
    subscriber_email_addresses = [var.budget_alert_email]
  }

  notification {
    comparison_operator        = "GREATER_THAN"
    notification_type          = "ACTUAL"
    threshold                  = 100
    threshold_type             = "PERCENTAGE"
    subscriber_email_addresses = [var.budget_alert_email]
  }
}
