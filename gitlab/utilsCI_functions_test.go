package gitlab

import (
	"reflect"
	"testing"

	"gopkg.in/yaml.v2"
)

// mustUnmarshalRun parses a `run:` YAML fragment the same way
// GitlabCIConf.GitlabJobs entries are decoded (yaml.v2, so nested maps come
// back as map[interface{}]interface{}), and returns the resulting value
// ready to feed into extractGitLabFunctionRefs.
func mustUnmarshalRun(t *testing.T, raw string) interface{} {
	t.Helper()
	var doc struct {
		Run interface{} `yaml:"run"`
	}
	if err := yaml.Unmarshal([]byte(raw), &doc); err != nil {
		t.Fatalf("unmarshal run fragment: %v", err)
	}
	return doc.Run
}

func TestExtractGitLabFunctionRefs(t *testing.T) {
	cases := []struct {
		name string
		yaml string
		want []string
	}{
		{
			name: "func_keyword",
			yaml: `
run:
  - name: say_hi
    func: registry.gitlab.com/group/proj/echo:1
`,
			want: []string{"registry.gitlab.com/group/proj/echo:1"},
		},
		{
			name: "step_keyword_deprecated",
			yaml: `
run:
  - name: legacy
    step: gitlab.com/group/proj/step@v1
`,
			want: []string{"gitlab.com/group/proj/step@v1"},
		},
		{
			name: "script_only_step_ignored",
			yaml: `
run:
  - name: plain
    script: echo hello
`,
			want: nil,
		},
		{
			name: "mixed_steps",
			yaml: `
run:
  - name: a
    script: echo hi
  - name: b
    func: registry.gitlab.com/group/proj/a:1
  - name: c
    step: gitlab.com/group/proj/b@v2
`,
			want: []string{"registry.gitlab.com/group/proj/a:1", "gitlab.com/group/proj/b@v2"},
		},
		{
			name: "no_run_block",
			yaml: `image: alpine`,
			want: nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			run := mustUnmarshalRun(t, tc.yaml)
			got := extractGitLabFunctionRefs(run)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("extractGitLabFunctionRefs() = %#v, want %#v", got, tc.want)
			}
		})
	}
}
