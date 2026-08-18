# A secret is identified by its resource id.
terraform import coderforge_secret.database secret:0004

# The material cannot be recovered on import: the vault does not return it to
# this provider. Declare the same values in your configuration; the first apply
# writes them.
