moved {
  from = aws_kms_key.storage
  to   = module.managed_contract.aws_kms_key.storage
}

moved {
  from = aws_kms_alias.storage
  to   = module.managed_contract.aws_kms_alias.storage
}

moved {
  from = aws_kms_key.crypto
  to   = module.managed_contract.aws_kms_key.crypto
}

moved {
  from = aws_kms_alias.crypto
  to   = module.managed_contract.aws_kms_alias.crypto
}

moved {
  from = aws_s3_bucket.contract
  to   = module.managed_contract.aws_s3_bucket.contract
}

moved {
  from = aws_s3_bucket_ownership_controls.contract
  to   = module.managed_contract.aws_s3_bucket_ownership_controls.contract
}

moved {
  from = aws_s3_bucket_public_access_block.contract
  to   = module.managed_contract.aws_s3_bucket_public_access_block.contract
}

moved {
  from = aws_s3_bucket_server_side_encryption_configuration.contract
  to   = module.managed_contract.aws_s3_bucket_server_side_encryption_configuration.contract
}

moved {
  from = aws_s3_bucket_policy.contract
  to   = module.managed_contract.aws_s3_bucket_policy.contract
}

moved {
  from = aws_iam_role.contract
  to   = module.managed_contract.aws_iam_role.contract
}

moved {
  from = aws_iam_role_policy.contract
  to   = module.managed_contract.aws_iam_role_policy.contract
}
