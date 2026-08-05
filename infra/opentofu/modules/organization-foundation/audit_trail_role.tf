locals {
  audit_trail_arn = "arn:aws:cloudtrail:*:${var.management_account_id}:trail/accountable-organization"
}

data "aws_iam_policy_document" "audit_trail_assume_role" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "AWS"
      identifiers = ["arn:aws:iam::${var.security_backup_account_id}:root"]
    }
  }
}

resource "aws_iam_role" "audit_trail" {
  name               = "accountable-organization-audit-trail"
  assume_role_policy = data.aws_iam_policy_document.audit_trail_assume_role.json

  lifecycle {
    precondition {
      condition     = data.aws_caller_identity.current.account_id == var.management_account_id
      error_message = "The organization audit trail role must be managed from the AWS Organizations management account."
    }
  }
}

data "aws_iam_policy_document" "audit_trail" {
  statement {
    actions = [
      "cloudtrail:AddTags",
      "cloudtrail:CreateTrail",
      "cloudtrail:DeleteTrail",
      "cloudtrail:GetEventSelectors",
      "cloudtrail:GetInsightSelectors",
      "cloudtrail:GetTrailStatus",
      "cloudtrail:ListTags",
      "cloudtrail:PutEventSelectors",
      "cloudtrail:PutInsightSelectors",
      "cloudtrail:RemoveTags",
      "cloudtrail:StartLogging",
      "cloudtrail:StopLogging",
      "cloudtrail:UpdateTrail",
    ]
    resources = [local.audit_trail_arn]
  }

  statement {
    actions = [
      "cloudtrail:DescribeTrails",
      "organizations:DescribeOrganization",
      "organizations:ListAWSServiceAccessForOrganization",
      "organizations:ListAccounts",
    ]
    resources = ["*"]
  }
}

resource "aws_iam_role_policy" "audit_trail" {
  name   = "accountable-organization-audit-trail"
  role   = aws_iam_role.audit_trail.id
  policy = data.aws_iam_policy_document.audit_trail.json
}
