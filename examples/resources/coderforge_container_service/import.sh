# A container service is identified by its four-digit platform id.
terraform import coderforge_container_service.edge 0001

# Cloud Builder stores the deployed bundle, not the `compose` value that
# produced it, so the first plan after an import proposes redeploying it.
# Review that plan before applying.
