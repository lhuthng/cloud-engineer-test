variable "environment" {
  type = string
}

resource "aws_ecs_cluster" "this" {
  name = "media-${var.environment}-cluster"

  setting {
    name  = "containerInsights"
    value = "enabled"
  }

  tags = {
    Name        = "media-${var.environment}-cluster"
    Environment = var.environment
  }
}

output "cluster_id" {
  value = aws_ecs_cluster.this.id
}

output "cluster_name" {
  value = aws_ecs_cluster.this.name
}
