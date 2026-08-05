output "contract" {
  value = {
    aws_account_id      = var.account_id
    aws_region          = var.region
    contract_role_arn   = aws_iam_role.contract.arn
    secure_bucket       = aws_s3_bucket.contract["secure"].bucket
    insecure_bucket     = aws_s3_bucket.contract["insecure"].bucket
    storage_kms_key_arn = aws_kms_key.storage.arn
    crypto_kms_key_arn  = aws_kms_key.crypto.arn
  }
}
