# Terraform Provider for CoderForge Cloud Builder

Manage CoderForge Cloud Builder infrastructure as code: virtual machines,
container services, disks, parameters, secrets and port forwarding rules.

```
terraform  ──►  terraform-provider-coderforge  ──►  terraform-api  ──►  Cloud Builder
   HCL                    gRPC plugin                 REST/JSON        srv12.net.coderforge.org:8000
```

The provider does not talk to Cloud Builder directly. It talks to
[`terraform-api`](../terraform-api), a service that gives Cloud Builder's API
the four properties Terraform requires and it does not have: synchronous
create semantics over background provisioning, a stable identifier for every
resource, JSON in place of multipart uploads, and one error shape carrying a
`not_found` the provider can act on. Putting that translation in a service
rather than in the plugin means it is testable on its own and shared by
anything else that needs it.

## Requirements

| | |
|---|---|
| Terraform | >= 1.5 |
| Go (to build) | >= 1.21 |
| terraform-api | >= 1.0.0, reachable from wherever Terraform runs |

## Authentication

The provider uses the same OAuth2 bearer token as the Cloud Builder console.
Generate one with:

```bash
curl -s -X POST https://auth.coderforge.org/api/1.3/auth/oauth2/token \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'grant_type=password&username=YOUR_USERNAME&password=YOUR_PASSWORD'
```

Take `access_token` from the response and export it. Supplying the token
through the environment rather than in HCL keeps it out of version control:

```bash
export CODERFORGE_TOKEN='eyJhbGciOi...'
export CODERFORGE_ENDPOINT='http://srv12.net.coderforge.org:8080'
```

Every entitlement Cloud Builder enforces applies unchanged: `terraform apply`
can do exactly what the person running it can do in the console, no more.

## Usage

```hcl
terraform {
  required_providers {
    coderforge = {
      source  = "coderforge/coderforge"
      version = "~> 2.0"
    }
  }
}

provider "coderforge" {
  endpoint = "http://srv12.net.coderforge.org:8080"
  # token comes from CODERFORGE_TOKEN
}

resource "coderforge_machine" "web" {
  name    = "web01"
  cpu     = 2
  ram     = 4096
  network = "hostonly"
}

resource "coderforge_network_rule" "http" {
  src_port  = "8080"
  dest_ip   = coderforge_machine.web.ip
  dest_port = "80"
}
```

See [`examples/complete`](examples/complete) for a configuration that exercises
most of the provider at once.

## Resources and data sources

| Resource | Manages |
|---|---|
| `coderforge_machine` | A virtual machine, its hardware and its power state |
| `coderforge_container_service` | A Docker Swarm cluster and the compose stack on it |
| `coderforge_storage` | A virtual disk, and where it is attached |
| `coderforge_param` | A value in the parameter store |
| `coderforge_secret` | Material in the vault |
| `coderforge_network_rule` | A host-to-guest port forward |

| Data source | Reads |
|---|---|
| `coderforge_machine` | One machine, by name |
| `coderforge_machines` | Every visible machine, filterable by state |
| `coderforge_container_service` | One service, by id or name |
| `coderforge_storage` | One disk, by id or name |
| `coderforge_param` | One parameter, value included |
| `coderforge_secret` | One secret's metadata — never its material |
| `coderforge_vm_templates` | Templates available for `template` |
| `coderforge_container_networks` | Swarm overlay networks, for `network_rid` |

Full reference in [`docs/`](docs), generated from the schemas.

## Behaviour worth knowing before the first apply

**Applies are synchronous.** When `terraform apply` finishes creating a
machine, the guest has booted and `ip` is populated; when it finishes creating
a container service, the Swarm has formed and the stack has converged. That is
why the default timeouts are in tens of minutes and why every long-running
resource takes a `timeouts` block.

**Hardware changes restart a machine.** Cloud Builder's modify operation powers
the guest off, applies the change and starts it again — the hypervisor cannot
do it live. The provider restores `desired_state` afterwards, so a machine
declared `stopped` does not silently come back up.

**A failed create rolls back.** Terraform never records a machine whose
creation failed, so anything left behind would be invisible to state. The
half-built machine is deleted; set `rollback_on_failure = false` to keep it for
a post-mortem.

**Machines owned by a container service are off limits.** A service provisions
its own nodes and owns their whole lifecycle. They carry `managed_by`, the
`coderforge_machines` data source hides them by default, and declaring one as a
`coderforge_machine` will fail.

**Secret material is write-only.** The provider never reads it back, so a value
changed in the vault by hand is not reported as drift; what is tracked is the
set of key names. The values you write are still stored in Terraform state —
protect the state file.

**Machine passwords are returned once.** Cloud Builder generates one at
creation and returns it exactly once. It is kept in state and never re-read,
which also means an imported machine has no password.

## Import

Every resource supports `terraform import`. The identifier is the natural one:

```bash
terraform import coderforge_machine.web web01
terraform import coderforge_container_service.app 0001
terraform import coderforge_storage.data store:0003
terraform import coderforge_param.settings param:0012
terraform import coderforge_secret.creds secret:0004
terraform import coderforge_network_rule.http tcp_8080_192.168.56.10_80
```

Network rule ids are derived from the rule itself
(`<protocol>_<src_port>_<dest_ip>_<dest_port>`), so importing one needs no
lookup. Machine and secret imports warn about the credential they cannot
recover.

## Development

```bash
make build          # build the binary
make test           # unit tests, no API needed
make docs           # regenerate docs/ from the schemas
make install        # build and install into the local plugin mirror
```

On Windows, `build_provider.bat` builds and installs into
`%APPDATA%\terraform.d\plugins` in one step.

Acceptance tests create real machines and are gated behind `TF_ACC`:

```bash
export TF_ACC=1
export CODERFORGE_ENDPOINT=http://srv12.net.coderforge.org:8080
export CODERFORGE_TOKEN=...
make testacc
```

### Layout

```
internal/client/      HTTP transport: retries, error decoding, one method per API operation
internal/provider/    Schemas and CRUD for every resource and data source
examples/             Configurations, also the source for the generated docs
docs/                 Generated - edit the schema descriptions, not these files
```

## Troubleshooting

Set `TF_LOG=DEBUG` to see every API call. The token is masked in that output.

| Symptom | Cause |
|---|---|
| `could not reach the CoderForge API` | Wrong `endpoint`, or terraform-api is down. Check `curl $CODERFORGE_ENDPOINT/healthz`. |
| `The token is invalid, expired or revoked` | Generate a new token; they are short-lived. |
| `Missing entitlement: cb:...` | The account lacks that permission in Cloud Builder. |
| `unrecognised body ... check that the endpoint points at terraform-api` | The endpoint is pointing at Cloud Builder itself, or at a proxy. |
| `Timed out after ...s waiting for provisioning` | Raise the `timeouts` block; check the machine's provisioning log in the console. |

Every error carries a request id that matches a line in the terraform-api log.

## License

MIT. See [LICENSE](LICENSE).
