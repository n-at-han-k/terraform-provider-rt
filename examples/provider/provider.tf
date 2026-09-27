terraform {
  required_providers {
    rt = {
      source = "ghcr.io/n-at-han-k/rt"
    }
  }
}

provider "rt" {
  endpoint = "/REST/2.0"
  # api_key  = "your-api-key"
  # token    = "your-bearer-token"
}
