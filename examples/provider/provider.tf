terraform {
  required_providers {
    coderforge = {
      source  = "coderforge/coderforge"
      version = "~> 2.0"
    }
  }
}

# The token is the same OAuth2 bearer token the Cloud Builder console uses.
# Supply it through CODERFORGE_TOKEN rather than in the configuration, so it
# does not end up in version control:
#
#   export CODERFORGE_TOKEN=$(curl -s -X POST \
#     https://auth.coderforge.org/api/1.3/auth/oauth2/token \
#     -H 'Content-Type: application/x-www-form-urlencoded' \
#     -d "grant_type=password&username=$USER&password=$PASSWORD" | jq -r .access_token)
#
provider "coderforge" {
  endpoint = "http://srv12.net.coderforge.org:8080"
}
