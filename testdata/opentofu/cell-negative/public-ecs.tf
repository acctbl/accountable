resource "aws_ecs_service" "api" {
  network_configuration {
    assign_public_ip = true
    security_groups  = [aws_security_group.api.id]
    subnets          = [for subnet in aws_subnet.application : subnet.id]
  }
}
