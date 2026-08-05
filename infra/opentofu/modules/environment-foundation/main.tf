module "account" {
  source = "../account-foundation"

  account_id  = var.account_id
  environment = var.environment
}

module "github" {
  source = "../github-deployment"

  account_id                 = var.account_id
  environment                = var.environment
  github_oidc_subject_prefix = var.github_oidc_subject_prefix
  region                     = var.region
}

resource "aws_ecr_repository" "application" {
  name                 = "accountable"
  image_tag_mutability = "IMMUTABLE"

  encryption_configuration {
    encryption_type = "AES256"
  }

  image_scanning_configuration {
    scan_on_push = true
  }
}

resource "aws_ecr_lifecycle_policy" "application" {
  repository = aws_ecr_repository.application.name
  policy = jsonencode({
    rules = [{
      action       = { type = "expire" }
      description  = "Retain the newest twenty production images"
      rulePriority = 1
      selection = {
        countNumber = 20
        countType   = "imageCountMoreThan"
        tagStatus   = "any"
      }
    }]
  })
}
