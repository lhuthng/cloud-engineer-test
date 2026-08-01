module "network" {
  source = "../../modules/network"

  environment          = var.environment
  vpc_cidr             = var.vpc_cidr
  azs                  = var.azs
  public_subnet_cidrs  = var.public_subnet_cidrs
  private_subnet_cidrs = var.private_subnet_cidrs
}

module "storage" {
  source = "../../modules/storage"

  environment = var.environment
}

module "ecr" {
  source = "../../modules/ecr"

  environment = var.environment
}

resource "random_password" "db" {
  length  = 24
  special = false
}

module "database" {
  source = "../../modules/database"

  environment        = var.environment
  vpc_id             = module.network.vpc_id
  private_subnet_ids = module.network.private_subnet_ids
  rds_sg_id          = module.network.rds_sg_id
  db_username        = var.db_username
  db_password        = random_password.db.result
  db_name            = var.db_name
  db_instance_class  = var.db_instance_class
}

module "secrets" {
  source = "../../modules/secrets"

  environment = var.environment
  db_address  = module.database.db_address
  db_port     = tostring(module.database.db_port)
  db_name     = var.db_name
  db_username = var.db_username
  db_password = random_password.db.result
}

module "iam" {
  source = "../../modules/iam"

  environment      = var.environment
  media_bucket_arn = module.storage.media_bucket_arn
  db_secret_arn    = module.secrets.db_secret_arn
}

module "cluster" {
  source = "../../modules/cluster"

  environment = var.environment
}

module "alb" {
  source = "../../modules/alb"

  environment       = var.environment
  vpc_id            = module.network.vpc_id
  public_subnet_ids = module.network.public_subnet_ids
  alb_sg_id         = module.network.alb_sg_id
}

module "api_service" {
  source = "../../modules/service"

  environment             = var.environment
  name                    = "api"
  cluster_id              = module.cluster.cluster_id
  image                   = "${module.ecr.api_repository_url}:${var.api_image_tag}"
  cpu                     = "256"
  memory                  = "512"
  desired_count           = 1
  container_port          = 8080
  target_group_arn        = module.alb.target_group_arn
  subnet_ids              = module.network.private_subnet_ids
  security_group_ids      = [module.network.ecs_sg_id]
  task_execution_role_arn = module.iam.task_execution_role_arn
  task_role_arn           = module.iam.task_role_arn
  environment_vars = {
    MEDIA_BUCKET = module.storage.media_bucket
  }
  secrets = {
    DB_HOST     = "${module.secrets.db_secret_arn}:DB_HOST::"
    DB_PORT     = "${module.secrets.db_secret_arn}:DB_PORT::"
    DB_NAME     = "${module.secrets.db_secret_arn}:DB_NAME::"
    DB_USERNAME = "${module.secrets.db_secret_arn}:DB_USERNAME::"
    DB_PASSWORD = "${module.secrets.db_secret_arn}:DB_PASSWORD::"
  }
}

module "worker_service" {
  source = "../../modules/service"

  environment             = var.environment
  name                    = "worker"
  cluster_id              = module.cluster.cluster_id
  image                   = "${module.ecr.worker_repository_url}:${var.worker_image_tag}"
  cpu                     = "1024"
  memory                  = "2048"
  desired_count           = 2
  subnet_ids              = module.network.private_subnet_ids
  security_group_ids      = [module.network.ecs_sg_id]
  task_execution_role_arn = module.iam.task_execution_role_arn
  task_role_arn           = module.iam.task_role_arn
  environment_vars = {
    MEDIA_BUCKET = module.storage.media_bucket
  }
  secrets = {
    DB_HOST     = "${module.secrets.db_secret_arn}:DB_HOST::"
    DB_PORT     = "${module.secrets.db_secret_arn}:DB_PORT::"
    DB_NAME     = "${module.secrets.db_secret_arn}:DB_NAME::"
    DB_USERNAME = "${module.secrets.db_secret_arn}:DB_USERNAME::"
    DB_PASSWORD = "${module.secrets.db_secret_arn}:DB_PASSWORD::"
  }
}
