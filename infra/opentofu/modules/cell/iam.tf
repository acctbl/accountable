locals {
  execution_global_actions = ["ecr:GetAuthorizationToken"]
  execution_repository_actions = [
    "ecr:BatchCheckLayerAvailability",
    "ecr:BatchGetImage",
    "ecr:GetDownloadUrlForLayer",
  ]
  execution_log_actions = [
    "logs:CreateLogStream",
    "logs:PutLogEvents",
  ]
  bootstrap_secret_actions = ["secretsmanager:GetSecretValue"]
  bootstrap_key_actions = [
    "kms:Decrypt",
    "kms:DescribeKey",
  ]
}

data "aws_iam_policy_document" "ecs_task_assume" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["ecs-tasks.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "execution" {
  name               = "${local.name}-execution"
  assume_role_policy = data.aws_iam_policy_document.ecs_task_assume.json
}

data "aws_iam_policy_document" "execution" {
  statement {
    actions   = local.execution_global_actions
    resources = ["*"]
  }

  statement {
    actions   = local.execution_repository_actions
    resources = [var.image_repository_arn]
  }

  statement {
    actions   = local.execution_log_actions
    resources = ["${aws_cloudwatch_log_group.runtime.arn}:*"]
  }

}

resource "aws_iam_role_policy" "execution" {
  name   = "${local.name}-execution"
  policy = data.aws_iam_policy_document.execution.json
  role   = aws_iam_role.execution.id
}

resource "aws_iam_role" "bootstrap_execution" {
  name               = "${local.name}-bootstrap-execution"
  assume_role_policy = data.aws_iam_policy_document.ecs_task_assume.json
}

data "aws_iam_policy_document" "bootstrap_execution" {
  source_policy_documents = [data.aws_iam_policy_document.execution.json]

  statement {
    actions   = local.bootstrap_secret_actions
    resources = [aws_db_instance.cell.master_user_secret[0].secret_arn]
  }

  statement {
    actions   = local.bootstrap_key_actions
    resources = [aws_kms_key.cell.arn]
  }
}

resource "aws_iam_role_policy" "bootstrap_execution" {
  name   = "${local.name}-bootstrap-execution"
  policy = data.aws_iam_policy_document.bootstrap_execution.json
  role   = aws_iam_role.bootstrap_execution.id
}

resource "aws_iam_role" "api" {
  name               = "${local.name}-api"
  assume_role_policy = data.aws_iam_policy_document.ecs_task_assume.json
}

data "aws_iam_policy_document" "api" {
  statement {
    actions = [
      "s3:GetBucketPublicAccessBlock",
      "s3:GetEncryptionConfiguration",
      "s3:ListBucket",
    ]
    resources = [aws_s3_bucket.data.arn]
  }

  statement {
    actions = [
      "s3:DeleteObject",
      "s3:GetObject",
      "s3:PutObject",
    ]
    resources = ["${aws_s3_bucket.data.arn}/foundation/*"]
  }

  statement {
    actions = [
      "kms:Decrypt",
      "kms:DescribeKey",
      "kms:Encrypt",
      "kms:GenerateDataKey",
    ]
    resources = [aws_kms_key.cell.arn]
  }
}

resource "aws_iam_role_policy" "api" {
  name   = "${local.name}-api"
  policy = data.aws_iam_policy_document.api.json
  role   = aws_iam_role.api.id
}

resource "aws_iam_role" "migrate" {
  name               = "${local.name}-migrate"
  assume_role_policy = data.aws_iam_policy_document.ecs_task_assume.json
}

resource "aws_iam_role" "bootstrap" {
  name               = "${local.name}-bootstrap"
  assume_role_policy = data.aws_iam_policy_document.ecs_task_assume.json
}
