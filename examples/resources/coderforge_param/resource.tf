# Parameters hold configuration. They are stored and read back in clear text,
# so anything that must stay hidden belongs in a coderforge_secret instead.
resource "coderforge_param" "notes" {
  name  = "deploy-notes"
  value = "Owned by the platform team. Rebuilt nightly.\n"
}

# A JSON object, which is the shape that can be used as an environment source
# for a container service: the platform turns its keys into environment
# variables. Anything else - a JSON array, or KEY=value lines - is stored
# perfectly well but refused as a source, so check `is_env_capable` before
# wiring a parameter into a service's `env_param_rid`.
resource "coderforge_param" "app_settings" {
  name = "app-settings"
  value = jsonencode({
    LOG_LEVEL     = "info"
    WORKERS       = "4"
    FEATURE_FLAGS = "beta"
  })
}

# true for app_settings, false for notes.
output "settings_can_be_used_as_env" {
  value = coderforge_param.app_settings.is_env_capable
}
