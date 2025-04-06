
terraform {
  backend "http" {
    address        = "http://localhost:9080/tfstate/myapp"
    lock_address   = "http://localhost:9080/tfstate/myapp/lock"
    unlock_address = "http://localhost:9080/tfstate/myapp/unlock"

    username = "my-access"
    password = "really-long-key"
  }
}

provider "aws" {
  shared_config_files      = ["${path.root}/../aws-config"]
  profile                  = "home-fern"

  s3_use_path_style = true

  # Skip AWS related checks and validations
  skip_credentials_validation = true
  skip_requesting_account_id = true
  skip_metadata_api_check = true
  skip_region_validation = true

  default_tags {
    tags = {
      application = var.stack_prefix
    }
  }
}
