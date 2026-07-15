variable "owner" {
  description = "Org name"
  type        = string
}

variable "app_id" {
  description = "Github app id"
  type        = string
}

variable "app_installation_id" {
  description = "Github app installation id"
  type        = string
}

variable "app_private_key" {
  description = "Github app pem file as string"
  type        = string
}

variable "environment_directory" {
  description = "Environment directory"
  type        = string
}

variable "bootstrap" {
  description = "When true, import existing organisation teams and members into state so the first apply is a no-op. Set only for the one-time bootstrap run."
  type        = bool
  default     = false
}