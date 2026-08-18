# Changelog

## 1.3.0

Rewritten against Cloud Builder's current API, through the new `terraform-api`
service. **This release is not backward compatible**: the previous version
targeted an endpoint and resource model that no longer exist, so there is no
upgrade path from 1.2.x state. Existing infrastructure should be brought over
with `terraform import`.

### Removed

- `coderforge_service` and its data source. It described a serverless
  abstraction Cloud Builder does not have. The closest equivalent is
  `coderforge_container_service`.
- The `cloud_space`, `locations`, `stack_id` and `host_url` provider arguments,
  which addressed an API that no longer exists.

### Added

- `coderforge_machine` — rewritten. Provisions a real Cloud Builder VM and
  waits for it to boot, with `desired_state` for power management, per-resource
  `timeouts`, and rollback of a failed create.
- `coderforge_container_service` — a Docker Swarm cluster and the compose stack
  running on it, from inline YAML or a project archive.
- `coderforge_storage` — a virtual disk, with attachment as an attribute.
- `coderforge_param`, `coderforge_secret` — the parameter store and the vault.
- `coderforge_network_rule` — port forwarding, with an identifier derived from
  the rule so imports need no lookup.
- Data sources: `coderforge_machine`, `coderforge_machines`,
  `coderforge_container_service`, `coderforge_storage`, `coderforge_param`,
  `coderforge_secret`, `coderforge_vm_templates`, `coderforge_container_networks`.
- `terraform import` for every resource.
- Provider arguments `endpoint`, `token`, `insecure`, `max_retries` and
  `request_timeout`, with `CODERFORGE_ENDPOINT` and `CODERFORGE_TOKEN`
  environment fallbacks.

### Changed

- Authentication now uses the Cloud Builder OAuth2 bearer token, the same one
  the console obtains at login. The token is forwarded to Cloud Builder on
  every request, so Terraform can do exactly what its operator can.
- Attribute constraints are validated at plan time rather than rejected mid-apply.
- Errors carry a request id that matches a line in the terraform-api log.

### Fixed

- `timeouts` is a block (`timeouts { create = "40m" }`), not an attribute.
  The attribute form would have required `timeouts = { ... }`, which is not
  what any other provider uses.
- `node_state` on a container service is a computed attribute rather than a
  block. As a block it planned as empty and then disagreed with the applied
  value, failing every create with "block count changed from 0 to 1".
- `rollback_on_failure` is `Optional` rather than `Optional + Computed` with a
  default, so an imported machine no longer shows a diff for it.
- `coderforge_storage` no longer refreshes an attachment it does not manage. A
  disk handed to a `coderforge_container_service` is attached by the service to
  a node the service owns; reading that back into the disk resource made every
  subsequent plan offer to detach it. Drift detection is unaffected for a disk
  the resource does attach.
- A container service that no longer exists is now recognised as gone rather
  than reported as a bad request, so it is dropped from state instead of
  failing every subsequent plan. Cloud Builder reports absence for these
  endpoints as `HTTP 400`, not `404`; the fix is in terraform-api.
- The goreleaser `ldflags` set a version variable in a package that does not
  exist in this module, so released binaries reported the compiled-in default.

### Known platform behaviour

- A parameter used as a container service's `env_param_rid` must be a **JSON
  object**; the platform turns its keys into environment variables and refuses
  anything else. `is_env_capable` reports whether a given parameter qualifies.
- Only `KV` secrets can be environment sources. A `PKI` secret is refused, with
  the suggestion to mount the certificate as a file.
- Machines are cloned serially by the hypervisor. A configuration that creates
  several should be applied with `-parallelism=1`, or the platform's own
  provisioning call times out before any of them finish.

## 1.1.0

Initial release.
