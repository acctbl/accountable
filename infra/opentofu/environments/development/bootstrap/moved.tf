moved {
  from = aws_kms_key.state
  to   = module.state.aws_kms_key.state
}

moved {
  from = aws_kms_alias.state
  to   = module.state.aws_kms_alias.state
}

moved {
  from = aws_s3_bucket.state
  to   = module.state.aws_s3_bucket.state
}

moved {
  from = aws_s3_bucket_ownership_controls.state
  to   = module.state.aws_s3_bucket_ownership_controls.state
}

moved {
  from = aws_s3_bucket_public_access_block.state
  to   = module.state.aws_s3_bucket_public_access_block.state
}

moved {
  from = aws_s3_bucket_versioning.state
  to   = module.state.aws_s3_bucket_versioning.state
}

moved {
  from = aws_s3_bucket_server_side_encryption_configuration.state
  to   = module.state.aws_s3_bucket_server_side_encryption_configuration.state
}

moved {
  from = aws_s3_bucket_policy.state
  to   = module.state.aws_s3_bucket_policy.state
}
