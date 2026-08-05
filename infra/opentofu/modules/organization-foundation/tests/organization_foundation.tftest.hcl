mock_provider "aws" {
  alias = "mock"

  mock_data "aws_caller_identity" {
    defaults = {
      account_id = "906543084690"
    }
  }
}

variables {
  budget_alert_email         = "cloud@example.com"
  management_account_id      = "906543084690"
  security_backup_account_id = "063280428550"
  account_budgets = {
    development = {
      account_id        = "453722413624"
      monthly_limit_usd = 50
    }
    staging = {
      account_id        = "723225039926"
      monthly_limit_usd = 10
    }
  }
}

run "budgets_are_central_and_account_scoped" {
  command = plan

  providers = {
    aws = aws.mock
  }

  assert {
    condition = (
      aws_budgets_budget.account["development"].limit_amount == "50" &&
      one(aws_budgets_budget.account["development"].cost_filter).name == "LinkedAccount" &&
      contains(one(aws_budgets_budget.account["development"].cost_filter).values, "453722413624")
    )
    error_message = "The management account must keep a development budget filtered to the development linked account."
  }
}

run "cloudtrail_is_delegated_outside_workload_accounts" {
  command = plan

  providers = {
    aws = aws.mock
  }

  assert {
    condition     = aws_cloudtrail_organization_delegated_admin_account.security_backup.account_id == "063280428550"
    error_message = "The security-backup account must be the organization CloudTrail delegated administrator."
  }

  assert {
    condition     = aws_organizations_aws_service_access.cloudtrail.service_principal == "cloudtrail.amazonaws.com"
    error_message = "CloudTrail trusted access must be enabled before the delegated administrator is registered."
  }
}

run "wrong_management_account_is_rejected" {
  command = plan

  providers = {
    aws = aws.mock
  }

  variables {
    management_account_id = "000000000000"
  }

  expect_failures = [
    aws_budgets_budget.account,
    aws_organizations_aws_service_access.cloudtrail,
  ]
}
