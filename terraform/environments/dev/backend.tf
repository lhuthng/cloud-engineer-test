terraform {
  backend "s3" {
    bucket         = "cloud-engineer-tfstate-790139457078"
    key            = "environments/dev/terraform.tfstate"
    region         = "eu-central-1"
    use_lockfile   = true
    encrypt        = true
  }
}
