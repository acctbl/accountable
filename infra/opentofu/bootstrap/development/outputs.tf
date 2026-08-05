output "aws_account_id" {
  value = local.account_id
}

output "state_bucket_name" {
  value = aws_s3_bucket.state.bucket
}

output "state_kms_key_arn" {
  value = aws_kms_key.state.arn
}
