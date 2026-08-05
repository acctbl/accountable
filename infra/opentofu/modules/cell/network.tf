resource "aws_vpc" "cell" {
  cidr_block           = var.vpc_cidr
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = {
    Name = local.name
  }
}

resource "aws_internet_gateway" "cell" {
  vpc_id = aws_vpc.cell.id

  tags = {
    Name = local.name
  }
}

resource "aws_subnet" "public" {
  for_each = local.public_subnets

  availability_zone       = each.key
  cidr_block              = each.value
  map_public_ip_on_launch = false
  vpc_id                  = aws_vpc.cell.id

  tags = {
    Name = "${local.name}-public-${each.key}"
    Tier = "public"
  }
}

resource "aws_subnet" "application" {
  for_each = local.application_subnets

  availability_zone       = each.key
  cidr_block              = each.value
  map_public_ip_on_launch = false
  vpc_id                  = aws_vpc.cell.id

  tags = {
    Name = "${local.name}-application-${each.key}"
    Tier = "application"
  }
}

resource "aws_subnet" "database" {
  for_each = local.database_subnets

  availability_zone       = each.key
  cidr_block              = each.value
  map_public_ip_on_launch = false
  vpc_id                  = aws_vpc.cell.id

  tags = {
    Name = "${local.name}-database-${each.key}"
    Tier = "database"
  }
}

resource "aws_route_table" "public" {
  vpc_id = aws_vpc.cell.id

  tags = {
    Name = "${local.name}-public"
  }
}

resource "aws_route" "public_internet" {
  destination_cidr_block = "0.0.0.0/0"
  gateway_id             = aws_internet_gateway.cell.id
  route_table_id         = aws_route_table.public.id
}

resource "aws_route_table_association" "public" {
  for_each = aws_subnet.public

  route_table_id = aws_route_table.public.id
  subnet_id      = each.value.id
}

resource "aws_eip" "nat" {
  domain = "vpc"

  depends_on = [aws_internet_gateway.cell]

  tags = {
    Name = "${local.name}-nat"
  }
}

resource "aws_nat_gateway" "cell" {
  allocation_id = aws_eip.nat.id
  subnet_id     = aws_subnet.public[var.availability_zones[0]].id

  tags = {
    Name = local.name
  }
}

resource "aws_route_table" "application" {
  vpc_id = aws_vpc.cell.id

  tags = {
    Name = "${local.name}-application"
  }
}

resource "aws_route" "application_internet" {
  destination_cidr_block = "0.0.0.0/0"
  nat_gateway_id         = aws_nat_gateway.cell.id
  route_table_id         = aws_route_table.application.id
}

resource "aws_route_table_association" "application" {
  for_each = aws_subnet.application

  route_table_id = aws_route_table.application.id
  subnet_id      = each.value.id
}

resource "aws_route_table" "database" {
  vpc_id = aws_vpc.cell.id

  tags = {
    Name = "${local.name}-database"
  }
}

resource "aws_route_table_association" "database" {
  for_each = aws_subnet.database

  route_table_id = aws_route_table.database.id
  subnet_id      = each.value.id
}

resource "aws_vpc_endpoint" "s3" {
  service_name      = "com.amazonaws.${data.aws_region.current.region}.s3"
  route_table_ids   = [aws_route_table.application.id]
  vpc_endpoint_type = "Gateway"
  vpc_id            = aws_vpc.cell.id

  tags = {
    Name = "${local.name}-s3"
  }
}

data "aws_region" "current" {}
