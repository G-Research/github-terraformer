# ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
# OUTPUTS
# ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

output "environment" {
  description = "The github_repository_environment resource."
  value       = github_repository_environment.environment
}

output "deployment_policies" {
  description = "Map of github_repository_environment_deployment_policy resources, keyed by branch/tag pattern."
  value       = github_repository_environment_deployment_policy.branch_pattern
}
