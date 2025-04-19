# defines the variables used in the Terraform configuration

variable "region" {
  description = "AWS region"
  type        = string
  default     = "us-west-2"
}

variable "key_name" {
  description = "SSH key name"
  type        = string
  default     = "cs6650hw1b"
}

# AWS credentials
variable "access_key" {
  description = "AWS access key"
  type        = string
  sensitive   = true
}

variable "secret_key" {
  description = "AWS secret key"
  type        = string
  sensitive   = true
}

variable "session_token" {
  description = "AWS session token"
  type        = string
  sensitive   = true
  default     = ""
}