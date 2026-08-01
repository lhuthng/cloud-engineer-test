variable "environment" {
  type = string
}

variable "name" {
  description = "Service name (api/worker)"
  type        = string
}

variable "cluster_id" {
  type = string
}

variable "image" {
  description = "Full image URI incl. tag"
  type        = string
}

variable "cpu" {
  description = "Task CPU units"
  type        = string
}

variable "memory" {
  description = "Task memory (MB)"
  type        = string
}

variable "desired_count" {
  type = number
}

variable "command" {
  description = "Container command override"
  type        = list(string)
  default     = null
}

variable "environment_vars" {
  description = "Plain env vars for the container"
  type        = map(string)
  default     = {}
}

variable "secrets" {
  description = "Secrets Manager refs (name -> valueFrom)"
  type        = map(string)
  default     = {}
}

variable "subnet_ids" {
  type = list(string)
}

variable "security_group_ids" {
  type = list(string)
}

variable "target_group_arn" {
  description = "ALB target group for the API (null for worker)"
  type        = string
  default     = null
}

variable "container_port" {
  description = "Container port for ALB routing"
  type        = number
  default     = 8080
}

variable "task_execution_role_arn" {
  type = string
}

variable "task_role_arn" {
  type = string
}

locals {
  container_definitions = jsonencode([
    {
      name         = var.name
      image        = var.image
      cpu          = tonumber(var.cpu)
      memory       = tonumber(var.memory)
      essential    = true
      command      = var.command
      environment  = [for k, v in var.environment_vars : { name = k, value = v }]
      secrets      = [for k, v in var.secrets : { name = k, valueFrom = v }]
      portMappings = var.target_group_arn != null ? [{ containerPort = var.container_port, protocol = "tcp" }] : []
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = aws_cloudwatch_log_group.this.name
          "awslogs-region"        = data.aws_region.current.name
          "awslogs-stream-prefix" = var.name
        }
      }
    }
  ])
}

data "aws_region" "current" {}

resource "aws_cloudwatch_log_group" "this" {
  name              = "/ecs/media-${var.environment}-${var.name}"
  retention_in_days = 14

  tags = {
    Environment = var.environment
  }
}

resource "aws_ecs_task_definition" "this" {
  family                   = "media-${var.environment}-${var.name}"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = var.cpu
  memory                   = var.memory
  execution_role_arn       = var.task_execution_role_arn
  task_role_arn            = var.task_role_arn
  container_definitions    = local.container_definitions

  tags = {
    Environment = var.environment
  }
}

resource "aws_ecs_service" "this" {
  name            = "media-${var.environment}-${var.name}"
  cluster         = var.cluster_id
  task_definition = aws_ecs_task_definition.this.arn
  desired_count   = var.desired_count
  launch_type     = "FARGATE"

  network_configuration {
    subnets          = var.subnet_ids
    security_groups  = var.security_group_ids
    assign_public_ip = false
  }

  dynamic "load_balancer" {
    for_each = var.target_group_arn != null ? [1] : []
    content {
      target_group_arn = var.target_group_arn
      container_name   = var.name
      container_port   = var.container_port
    }
  }

  depends_on = [aws_cloudwatch_log_group.this]

  tags = {
    Environment = var.environment
  }
}
