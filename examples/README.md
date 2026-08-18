# Examples

`tfplugindocs` reads specific filenames from this tree to build the registry
documentation, so the layout is fixed:

| Path | Renders into |
|---|---|
| `provider/provider.tf` | The provider index page |
| `resources/<name>/resource.tf` | The example on that resource's page |
| `resources/<name>/import.sh` | The import section on that page |
| `data-sources/<name>/data-source.tf` | The example on that data source's page |

`complete/` is not part of the generated docs. It is a working configuration
that exercises most of the provider at once, and the best starting point for
trying things out:

```bash
export CODERFORGE_TOKEN=$(curl -s -X POST \
  https://auth.coderforge.org/api/1.3/auth/oauth2/token \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d "grant_type=password&username=$USER&password=$PASSWORD" | jq -r .access_token)

cd complete
terraform init
terraform apply -var database_password=...
```

Applying it provisions real machines, which takes real minutes. `terraform
destroy` removes everything it created.
