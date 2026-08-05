resource "aws_ecs_task_definition" "api" {
  container_definitions = jsonencode([{
    essential = true
    image     = "453722413624.dkr.ecr.eu-west-2.amazonaws.com/accountable:main"
    name      = "api"
  }])
}
