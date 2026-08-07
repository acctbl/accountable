resource "aws_lb" "api" {
  name                       = local.name
  drop_invalid_header_fields = true
  enable_deletion_protection = var.cell_lifecycle == "durable"
  internal                   = true
  load_balancer_type         = "application"
  security_groups            = [aws_security_group.alb.id]
  subnets                    = [for subnet in aws_subnet.application : subnet.id]

  lifecycle {
    postcondition {
      condition = (
        self.internal &&
        self.enable_deletion_protection == (var.cell_lifecycle == "durable")
      )
      error_message = "The cell ALB must remain internal and use deletion protection exactly when the cell is durable."
    }
  }
}

resource "aws_lb_target_group" "api" {
  name        = local.name
  port        = local.api_port
  protocol    = "HTTP"
  target_type = "ip"
  vpc_id      = aws_vpc.cell.id

  deregistration_delay = 15

  health_check {
    enabled             = true
    healthy_threshold   = 2
    interval            = 15
    matcher             = "200"
    path                = "/_health/live"
    port                = "traffic-port"
    protocol            = "HTTP"
    timeout             = 5
    unhealthy_threshold = 3
  }
}

# The listener is reachable only through CloudFront's private VPC-origin connection.
#trivy:ignore:AVD-AWS-0054:exp:2026-11-05
resource "aws_lb_listener" "api" {
  load_balancer_arn = aws_lb.api.arn
  port              = 80
  protocol          = "HTTP"

  default_action {
    target_group_arn = aws_lb_target_group.api.arn
    type             = "forward"
  }
}

resource "aws_ecs_cluster" "cell" {
  name = local.name

  setting {
    name  = "containerInsights"
    value = "enabled"
  }
}

resource "aws_cloudwatch_log_group" "runtime" {
  name              = "/accountable/${var.cell_id}"
  retention_in_days = 30
}

locals {
  log_configuration = {
    logDriver = "awslogs"
    options = {
      awslogs-group         = aws_cloudwatch_log_group.runtime.name
      awslogs-region        = data.aws_region.current.region
      awslogs-stream-prefix = "runtime"
    }
  }

  runtime_scratch = {
    linuxParameters = {
      tmpfs = [{
        containerPath = "/run/accountable"
        mountOptions  = ["rw", "noexec", "nosuid", "nodev", "mode=1777"]
        size          = 64
      }]
    }
  }
}

resource "aws_ecs_task_definition" "api" {
  container_definitions = jsonencode([merge(local.runtime_scratch, {
    command   = ["api"]
    essential = true
    environment = [{
      name  = "ACCOUNTABLE_CONFIG_BASE64"
      value = base64encode(local.api_config)
    }]
    image            = var.image_uri
    logConfiguration = local.log_configuration
    name             = "api"
    portMappings = [{
      appProtocol   = "http"
      containerPort = local.api_port
      hostPort      = local.api_port
      name          = "http"
      protocol      = "tcp"
    }]
    readonlyRootFilesystem = true
  })])
  cpu                      = 256
  execution_role_arn       = aws_iam_role.execution.arn
  family                   = "${local.name}-api"
  memory                   = 512
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  task_role_arn            = aws_iam_role.api.arn

  runtime_platform {
    cpu_architecture        = "ARM64"
    operating_system_family = "LINUX"
  }
}

resource "aws_ecs_task_definition" "migrate" {
  container_definitions = jsonencode([merge(local.runtime_scratch, {
    command   = ["migrate"]
    essential = true
    environment = [{
      name  = "ACCOUNTABLE_CONFIG_BASE64"
      value = base64encode(local.migrate_config)
    }]
    image                  = var.image_uri
    logConfiguration       = local.log_configuration
    name                   = "migrate"
    readonlyRootFilesystem = true
  })])
  cpu                      = 256
  execution_role_arn       = aws_iam_role.execution.arn
  family                   = "${local.name}-migrate"
  memory                   = 512
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  task_role_arn            = aws_iam_role.migrate.arn

  runtime_platform {
    cpu_architecture        = "ARM64"
    operating_system_family = "LINUX"
  }
}

resource "aws_ecs_task_definition" "preflight" {
  container_definitions = jsonencode([merge(local.runtime_scratch, {
    command   = ["preflight"]
    essential = true
    environment = [{
      name  = "ACCOUNTABLE_CONFIG_BASE64"
      value = base64encode(local.preflight_config)
    }]
    image                  = var.image_uri
    logConfiguration       = local.log_configuration
    name                   = "preflight"
    readonlyRootFilesystem = true
  })])
  cpu                      = 256
  execution_role_arn       = aws_iam_role.execution.arn
  family                   = "${local.name}-preflight"
  memory                   = 512
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  task_role_arn            = aws_iam_role.api.arn

  runtime_platform {
    cpu_architecture        = "ARM64"
    operating_system_family = "LINUX"
  }
}

resource "aws_ecs_task_definition" "bootstrap" {
  container_definitions = jsonencode([merge(local.runtime_scratch, {
    command   = ["bootstrap"]
    essential = true
    environment = [{
      name  = "ACCOUNTABLE_CONFIG_BASE64"
      value = base64encode(local.bootstrap_config)
    }]
    secrets = [{
      name      = "ACCOUNTABLE_DATABASE_MASTER_PASSWORD"
      valueFrom = "${aws_db_instance.cell.master_user_secret[0].secret_arn}:password::"
    }]
    image                  = var.image_uri
    logConfiguration       = local.log_configuration
    name                   = "bootstrap"
    readonlyRootFilesystem = true
  })])
  cpu                      = 256
  execution_role_arn       = aws_iam_role.bootstrap_execution.arn
  family                   = "${local.name}-bootstrap"
  memory                   = 512
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  task_role_arn            = aws_iam_role.bootstrap.arn

  runtime_platform {
    cpu_architecture        = "ARM64"
    operating_system_family = "LINUX"
  }
}

resource "aws_ecs_service" "api" {
  name             = "${local.name}-api"
  cluster          = aws_ecs_cluster.cell.id
  desired_count    = var.api_desired_count
  launch_type      = "FARGATE"
  platform_version = "1.4.0"
  task_definition  = aws_ecs_task_definition.api.arn

  deployment_circuit_breaker {
    enable   = true
    rollback = true
  }

  load_balancer {
    container_name   = "api"
    container_port   = local.api_port
    target_group_arn = aws_lb_target_group.api.arn
  }

  network_configuration {
    assign_public_ip = false
    security_groups  = [aws_security_group.api.id]
    subnets          = [for subnet in aws_subnet.application : subnet.id]
  }

  lifecycle {
    postcondition {
      condition     = !self.network_configuration[0].assign_public_ip
      error_message = "The API service must not assign public IP addresses."
    }
  }

  depends_on = [aws_lb_listener.api]
}
