package gitlab

import (
	"testing"

	"github.com/getplumber/plumber/internal/ir"
)

func TestToNormalizedPipeline_Empty(t *testing.T) {
	pipeline := ToNormalizedPipeline("group/project", "main", "", "", nil, nil, nil)
	if pipeline.Provider != ir.ProviderGitLab {
		t.Fatalf("expected provider gitlab, got %q", pipeline.Provider)
	}
	if pipeline.ProjectPath != "group/project" {
		t.Fatalf("expected project path propagated, got %q", pipeline.ProjectPath)
	}
	if pipeline.DefaultBranch != "main" {
		t.Fatalf("expected default branch propagated, got %q", pipeline.DefaultBranch)
	}
	if len(pipeline.Jobs) != 0 {
		t.Fatalf("expected no jobs, got %d", len(pipeline.Jobs))
	}
}

func TestToNormalizedPipeline_JobsAndImages(t *testing.T) {
	origin := &GitlabPipelineOriginData{
		JobMap: map[string]*GitlabPipelineJobData{
			"build":  {Name: "build"},
			"deploy": {Name: "deploy"},
			"lint":   {Name: "lint"},
		},
	}
	images := &GitlabPipelineImageData{
		Images: []GitlabPipelineImageInfo{
			{Job: "build", Link: "docker.io/alpine:3.20", Name: "alpine", Tag: "3.20"},
			{Job: "deploy", Link: "registry.example.com/deployer@sha256:abcdef", Name: "deployer"},
		},
	}

	pipeline := ToNormalizedPipeline("grp/proj", "main", "", "", origin, images, nil)

	if got := len(pipeline.Jobs); got != 3 {
		t.Fatalf("expected 3 jobs, got %d", got)
	}

	// Sorted alphabetically: build, deploy, lint
	names := []string{pipeline.Jobs[0].Name, pipeline.Jobs[1].Name, pipeline.Jobs[2].Name}
	expected := []string{"build", "deploy", "lint"}
	for i := range names {
		if names[i] != expected[i] {
			t.Fatalf("jobs[%d]: expected %q, got %q", i, expected[i], names[i])
		}
	}

	if pipeline.Jobs[0].Image == nil || pipeline.Jobs[0].Image.Tag != "3.20" {
		t.Fatalf("build job image: expected tag 3.20, got %+v", pipeline.Jobs[0].Image)
	}
	if pipeline.Jobs[1].Image == nil || pipeline.Jobs[1].Image.Digest != "sha256:abcdef" {
		t.Fatalf("deploy job image: expected digest sha256:abcdef, got %+v", pipeline.Jobs[1].Image)
	}
	if pipeline.Jobs[2].Image != nil {
		t.Fatalf("lint job: expected no image, got %+v", pipeline.Jobs[2].Image)
	}
}

func TestToNormalizedPipeline_NilJobInMap(t *testing.T) {
	origin := &GitlabPipelineOriginData{
		JobMap: map[string]*GitlabPipelineJobData{
			"valid":     {Name: "valid"},
			"corrupted": nil,
		},
	}

	pipeline := ToNormalizedPipeline("grp/proj", "main", "", "", origin, nil, nil)
	if got := len(pipeline.Jobs); got != 1 {
		t.Fatalf("expected 1 job (nil entry skipped), got %d", got)
	}
	if pipeline.Jobs[0].Name != "valid" {
		t.Fatalf("expected valid job kept, got %q", pipeline.Jobs[0].Name)
	}
}

func TestPredefinedGitLabVariables(t *testing.T) {
	got := predefinedGitLabVariables("my-group/my-subgroup/my-project", "https://gitlab.example.com")
	want := map[string]string{
		"CI_TEMPLATE_REGISTRY_HOST": "registry.gitlab.com",
		"CI_PROJECT_PATH":           "my-group/my-subgroup/my-project",
		"CI_PROJECT_NAMESPACE":      "my-group/my-subgroup",
		"CI_PROJECT_NAME":           "my-project",
		"CI_PROJECT_ROOT_NAMESPACE": "my-group",
		"CI_SERVER_FQDN":            "gitlab.example.com",
		"CI_SERVER_HOST":            "gitlab.example.com",
		"CI_SERVER_URL":             "https://gitlab.example.com",
		"CI_SERVER_PROTOCOL":        "https",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("predefinedGitLabVariables()[%q] = %q, want %q", k, got[k], v)
		}
	}
}

func TestPredefinedGitLabVariables_EmptyInputs(t *testing.T) {
	got := predefinedGitLabVariables("", "")
	if _, ok := got["CI_PROJECT_PATH"]; ok {
		t.Error("expected no CI_PROJECT_PATH when projectPath is empty")
	}
	if _, ok := got["CI_SERVER_FQDN"]; ok {
		t.Error("expected no CI_SERVER_FQDN when instanceURL is empty")
	}
	if got["CI_TEMPLATE_REGISTRY_HOST"] != "registry.gitlab.com" {
		t.Errorf("CI_TEMPLATE_REGISTRY_HOST should always be set, got %q", got["CI_TEMPLATE_REGISTRY_HOST"])
	}
}

// TestToNormalizedPipeline_GlobalVariablesMerge checks the precedence
// order documented on buildGlobalVariables: predefined < pipeline
// `variables:` block < instance < group < project. Each source
// overrides the same key so the test can tell which one actually won.
func TestToNormalizedPipeline_GlobalVariablesMerge(t *testing.T) {
	origin := &GitlabPipelineOriginData{
		JobMap: map[string]*GitlabPipelineJobData{},
		MergedConf: &GitlabCIConf{
			GlobalVariables: map[string]interface{}{
				"CI_PROJECT_PATH": "overridden-by-pipeline-vars",
				"PIPELINE_ONLY":   "pipeline-value",
			},
		},
	}
	images := &GitlabPipelineImageData{
		InstanceVars: map[string]string{"CI_PROJECT_PATH": "overridden-by-instance", "INSTANCE_ONLY": "instance-value"},
		GroupVars:    map[string]string{"CI_PROJECT_PATH": "overridden-by-group", "GROUP_ONLY": "group-value"},
		ProjectVars:  map[string]string{"CI_PROJECT_PATH": "overridden-by-project", "PROJECT_ONLY": "project-value"},
	}

	pipeline := ToNormalizedPipeline("grp/proj", "main", "", "https://gitlab.example.com", origin, images, nil)

	if got := pipeline.GlobalVariables["CI_PROJECT_PATH"]; got != "overridden-by-project" {
		t.Errorf("CI_PROJECT_PATH: expected project-level value to win, got %q", got)
	}
	for k, want := range map[string]string{
		"PIPELINE_ONLY": "pipeline-value",
		"INSTANCE_ONLY": "instance-value",
		"GROUP_ONLY":    "group-value",
		"PROJECT_ONLY":  "project-value",
	} {
		if got := pipeline.GlobalVariables[k]; got != want {
			t.Errorf("%s: expected %q, got %q", k, want, got)
		}
	}
	if got := pipeline.GlobalVariables["CI_SERVER_FQDN"]; got != "gitlab.example.com" {
		t.Errorf("CI_SERVER_FQDN: expected predefined value gitlab.example.com, got %q", got)
	}
}

func TestToNormalizedPipeline_JobFunctions(t *testing.T) {
	origin := &GitlabPipelineOriginData{
		JobMap: map[string]*GitlabPipelineJobData{
			"build": {Name: "build"},
			"lint":  {Name: "lint"},
		},
	}
	images := &GitlabPipelineImageData{
		Functions: []GitlabPipelineFunctionInfo{
			{Job: "build", Link: "registry.gitlab.com/group/proj/step:1"},
			{Job: "build", Link: "./local-step"},
		},
	}

	pipeline := ToNormalizedPipeline("grp/proj", "main", "", "", origin, images, nil)

	var build, lint *ir.Job
	for i := range pipeline.Jobs {
		switch pipeline.Jobs[i].Name {
		case "build":
			build = &pipeline.Jobs[i]
		case "lint":
			lint = &pipeline.Jobs[i]
		}
	}
	if build == nil {
		t.Fatal("build job not found")
	}
	if got := len(build.Functions); got != 2 {
		t.Fatalf("build.Functions: expected 2, got %d (%+v)", got, build.Functions)
	}
	if lint != nil && len(lint.Functions) != 0 {
		t.Fatalf("lint.Functions: expected none, got %+v", lint.Functions)
	}
}
