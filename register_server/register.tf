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
    region = "us-east-1"
}

resource "aws_instance" "ec2_register"{
    ami = "ami-01cc34ab2709337aa"  
    instance_type = "t2.micro"
  
    provisioner "file" {
        source      = "register_server"
        destination = "$HOME"
    }
    provisioner "remote-exec" {
        inline = [
        "./register_server",
        ]
    }
    connection {
        type        = "ssh"
        host        = self.public_ip
        user        = "ec2-user"
        private_key = file("~/.aws/sdcc_keypair.pem")
        timeout     = "2m"
    }
}
