# function-authorized-sources — flag `run:` block function references
# (the current `func:` keyword, or the deprecated-but-still-supported
# `step:` keyword) whose source doesn't match
# functionMustComeFromAuthorizedSources.trustedUrls. GitLab Functions
# execute arbitrary third-party code with the job's full context, the
# same supply-chain exposure as CI/CD Components. Local functions
# (`./…`, `/…`) run from the project's own repository and are always
# exempt.
#
# `$VAR` / `${VAR}` references in both the function's source and each
# trustedUrls pattern are resolved against input.pipeline.globalVariables
# before comparison, so the shipped default
# `$CI_TEMPLATE_REGISTRY_HOST/$CI_PROJECT_PATH/*` implicitly trusts
# functions the scanned project publishes for itself.
package function_authorized_sources

import rego.v1

deny contains finding if {
	input.config.functionAuthorizedSources
	some i, j
	job := input.pipeline.jobs[i]
	fn := job.functions[j]
	fn.uses != ""
	not _is_local(fn.uses)
	not _is_authorized(fn.uses)
	finding := {
		"code":     "ISSUE-415",
		"severity": "high",
		"message":  sprintf("job %q references function %q from an unauthorized source", [job.name, fn.uses]),
		"job":      job.name,
		"link":     fn.uses,
		"status":   "unauthorized",
	}
}

_is_local(uses) if startswith(uses, "./")

_is_local(uses) if startswith(uses, "/")

_is_authorized(uses) if {
	pattern := input.config.functionAuthorizedSources.trustedUrls[_]
	glob.match(_resolve_vars(pattern), null, _resolve_vars(uses))
}

# _resolve_vars replaces every `$VAR` / `${VAR}` reference in s with its
# value from input.pipeline.globalVariables, in a single pass via
# strings.replace_n so multiple distinct variables in the same string are
# each substituted correctly.
_resolve_vars(s) := strings.replace_n(_var_patterns(s), s)

_var_patterns(s) := {p: v |
	some m in regex.find_all_string_submatch_n(`\$\{?([a-zA-Z_][a-zA-Z0-9_]*)\}?`, s, -1)
	name := m[1]
	v := input.pipeline.globalVariables[name]
	some p in {sprintf("$%s", [name]), sprintf("${%s}", [name])}
}
