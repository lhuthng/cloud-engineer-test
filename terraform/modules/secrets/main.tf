resource "aws_secretsmanager_secret" "db" {
  name = "media-${var.environment}-db"

  tags = {
    Environment = var.environment
  }
}

resource "aws_secretsmanager_secret_version" "db" {
  secret_id = aws_secretsmanager_secret.db.id

  secret_string = jsonencode({
    DB_HOST     = var.db_address
    DB_PORT     = var.db_port
    DB_NAME     = var.db_name
    DB_USERNAME = var.db_username
    DB_PASSWORD = var.db_password
  })
}
