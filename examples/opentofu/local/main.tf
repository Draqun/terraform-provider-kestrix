terraform {
  required_providers {
    kestrix = { source = "kestrix/kestrix" }
  }
}

provider "kestrix" { endpoint = var.kestrix_endpoint }

resource "kestrix_project" "example" {
  name                = "provider-example"
  repo_url            = "https://gitlab.com/kestrix/kestrix.git"
  repo_ref            = "main"
  repo_credential_ref = var.repo_credential_ref
  pipeline_dir        = "ci"
  pipeline_file       = "pipeline.tf"
}

variable "kestrix_endpoint" {
  type    = string
  default = "http://localhost:8080"
}

variable "repo_credential_ref" {
  type      = string
  sensitive = true
  default   = ""
}
