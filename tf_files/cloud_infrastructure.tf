terraform {

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 3.27"
    }
  }

  required_version = ">= 0.14.9"
}

provider "aws" {
  profile = "default"
  region  = "us-east-1"
}

resource "aws_dynamodb_table" "sdcc_table" {
  name           = "HeavyValues"
  write_capacity = 10
  read_capacity  = 10
  hash_key       = "Key"

  attribute {
    name = "Key"
    type = "S"
  }
}
