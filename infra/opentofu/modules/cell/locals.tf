data "aws_caller_identity" "current" {}

locals {
  name = "accountable-${var.cell_id}"

  public_subnets = {
    for index, availability_zone in var.availability_zones : availability_zone => cidrsubnet(var.vpc_cidr, 4, index)
  }
  application_subnets = {
    for index, availability_zone in var.availability_zones : availability_zone => cidrsubnet(var.vpc_cidr, 4, index + 4)
  }
  database_subnets = {
    for index, availability_zone in var.availability_zones : availability_zone => cidrsubnet(var.vpc_cidr, 4, index + 8)
  }

  api_port      = 8080
  database_port = 5432
  data_bucket   = "${local.name}-${var.account_id}-data"
  web_bucket    = "${local.name}-${var.account_id}-web"
  secret_root   = "/accountable/${var.cell_id}"
}
