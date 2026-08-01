variable "environment" {
  type = string
}

resource "aws_s3_bucket" "media" {
  bucket = "media-${var.environment}-files"

  tags = {
    Name        = "media-${var.environment}-files"
    Environment = var.environment
  }
}

resource "aws_s3_bucket_versioning" "media" {
  bucket = aws_s3_bucket.media.id

  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "media" {
  bucket = aws_s3_bucket.media.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

resource "aws_s3_bucket_public_access_block" "media" {
  bucket = aws_s3_bucket.media.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

output "media_bucket" {
  value = aws_s3_bucket.media.id
}

output "media_bucket_arn" {
  value = aws_s3_bucket.media.arn
}
