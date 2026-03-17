# ---------------------------------------------------------------------------------------------------------------------
# SET TERRAFORM AND PROVIDER REQUIREMENTS FOR RUNNING THIS MODULE
# ---------------------------------------------------------------------------------------------------------------------

terraform {
  required_version = "~> 1.0"

  required_providers {
    github = {
      source  = "G-Research-Forks/github"
      version = "6.9.0-gr.3"
    }
  }
}
