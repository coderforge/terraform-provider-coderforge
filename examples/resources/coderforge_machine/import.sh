# A machine is identified by its name.
terraform import coderforge_machine.web web01

# The generated password cannot be recovered on import: Cloud Builder returns
# it only when the machine is created. The provider warns about this, and
# `password` will be null in state.
