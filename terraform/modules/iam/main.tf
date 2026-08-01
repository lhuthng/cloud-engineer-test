variable "environment" {
  type = string
}

variable "media_bucket_arn" {
  type = string
}

variable "db_secret_arn" {
  type = string
}

data "aws_iam_policy_document" "ecs_assume" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["ecs-tasks.amazonaws.com"]
    }
  }
}

data "aws_iam_policy_document" "task_execution" {
  statement {
    actions = [
      "ecr:GetAuthorizationToken",
      "ecr:BatchCheckLayerAvailability",
      "ecr:GetDownloadUrlForLayer",
      "ecr:BatchGetImage",
    ]
    resources = ["*"]
  }

  statement {
    actions = [
      "logs:CreateLogStream",
      "logs:PutLogEvents",
    ]
    resources = ["*"]
  }

  statement {
    actions   = ["secretsmanager:GetSecretValue"]
    resources = [var.db_secret_arn]
  }
}

data "aws_iam_policy_document" "task" {
  statement {
    actions = [
      "s3:GetObject",
      "s3:PutObject",
      "s3:DeleteObject",
      "s3:ListBucket",
    ]
    resources = [
      var.media_bucket_arn,
      "${var.media_bucket_arn}/*",
    ]
  }

  statement {
    actions   = ["secretsmanager:GetSecretValue"]
    resources = [var.db_secret_arn]
  }
}

resource "aws_iam_role" "task_execution" {
  name               = "media-${var.environment}-task-execution"
  assume_role_policy = data.aws_iam_policy_document.ecs_assume.json

  tags = {
    Environment = var.environment
  }
}

resource "aws_iam_role_policy" "task_execution" {
  name   = "media-${var.environment}-task-execution"
  role   = aws_iam_role.task_execution.id
  policy = data.aws_iam_policy_document.task_execution.json
}

resource "aws_iam_role" "task" {
  name               = "media-${var.environment}-task"
  assume_role_policy = data.aws_iam_policy_document.ecs_assume.json

  tags = {
    Environment = var.environment
  }
}

resource "aws_iam_role_policy" "task" {
  name   = "media-${var.environment}-task"
  role   = aws_iam_role.task.id
  policy = data.aws_iam_policy_document.task.json
}

output "task_execution_role_arn" {
  value = aws_iam_role.task_execution.arn
}

output "task_role_arn" {
  value = aws_iam_role.task.arn
}
