terraform {
  backend "s3" {
    bucket       = "cloud-engineer-tfstate-790139457078"
    key          = "environments/prod/terraform.tfstate"
    region       = "eu-central-1"
    use_lockfile = true
    encrypt      = true
  }
}
