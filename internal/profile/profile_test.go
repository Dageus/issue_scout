package profile

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// validProfile returns a fresh profile that passes validation.
// Each test breaks one thing on its own copy.
func validProfile() *Profile {
	return &Profile{
		About: "Backend dev, strongest in Go and Rust.",
		Languages: LanguagePreferences{
			Strong: []string{"go", "rust"},
			Ok:     []string{"python"},
		},
		LabelsWanted:     []string{"good first issue"},
		Interests:        []string{"CLI tools"},
		Avoid:            []string{"frontend/CSS work"},
		MaxHoursPerIssue: 6,
		MaxIssueAgeHours: 48,
		MinRepoStars:     50,
	}
}

// check runs the same steps Load does after decoding.
func check(p *Profile) error {
	p.normalize()
	return p.Validate()
}

// wantErr fails unless err mentions every substring in want.
func wantErr(t *testing.T, err error, want ...string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected an error mentioning %q, got nil", want)
	}
	for _, w := range want {
		if !strings.Contains(err.Error(), w) {
			t.Errorf("error %q does not mention %q", err, w)
		}
	}
}

func wantStrings(t *testing.T, got, want []string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

// writeProfile writes yaml to a temp file and returns its path.
func writeProfile(t *testing.T, yaml string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "profile.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestValidProfileResolves(t *testing.T) {
	if err := check(validProfile()); err != nil {
		t.Fatalf("valid profile failed validation: %v", err)
	}
}

func TestUppercaseLanguageResolves(t *testing.T) {
	p := validProfile()
	p.Languages.Strong = []string{"Go", "RUST"}
	if err := check(p); err != nil {
		t.Fatal(err)
	}
	wantStrings(t, p.Languages.Strong, []string{"go", "rust"})
}

func TestSpacedLanguageResolves(t *testing.T) {
	p := validProfile()
	p.Languages.Ok = []string{"VIM SCRIPT"}
	if err := check(p); err != nil {
		t.Fatal(err)
	}
	wantStrings(t, p.Languages.Ok, []string{"vim script"})
}

func TestTrailingSpacedLanguageResolves(t *testing.T) {
	p := validProfile()
	p.Languages.Strong = []string{"  go ", "rust\t"}
	if err := check(p); err != nil {
		t.Fatal(err)
	}
	wantStrings(t, p.Languages.Strong, []string{"go", "rust"})
}

func TestDuplicateLanguageResolves(t *testing.T) {
	p := validProfile()
	p.Languages.Strong = []string{"rust", "go", "Rust", "golang"}
	if err := check(p); err != nil {
		t.Fatal(err)
	}
	// Duplicates and aliases collapse; first-seen order is kept.
	wantStrings(t, p.Languages.Strong, []string{"rust", "go"})
}

func TestMistypedLanguageErrors(t *testing.T) {
	p := validProfile()
	p.Languages.Ok = []string{"pyhton"}
	wantErr(t, check(p), `unknown language "pyhton"`)
}

func TestLanguageInMoreThanOneLanguagePreferenceErrors(t *testing.T) {
	p := validProfile()
	p.Languages.Reject = []string{"go"}
	wantErr(t, check(p), `"go" is in both strong and reject`)
}

func TestSameLanguageAliasInMoreThanOneLanguagePreferenceErrors(t *testing.T) {
	p := validProfile()
	p.Languages.Strong = []string{"bash"}
	p.Languages.Reject = []string{"sh"}
	wantErr(t, check(p), `"shell" is in both strong and reject`)
}

func TestNoLanguagesErrors(t *testing.T) {
	tests := map[string]LanguagePreferences{
		"missing section": LanguagePreferences{},
		"empty strong":    {Ok: []string{"python"}},
	}
	for name, langs := range tests {
		t.Run(name, func(t *testing.T) {
			p := validProfile()
			p.Languages = langs
			wantErr(t, check(p), "strong language")
		})
	}
}

func TestEmptyAboutErrors(t *testing.T) {
	p := validProfile()
	p.About = "  \n "
	wantErr(t, check(p), "about")
}

func TestZeroOrNegativeLimitsError(t *testing.T) {
	tests := map[string]struct {
		breakIt func(*Profile)
		want    string
	}{
		"hours missing":  {func(p *Profile) { p.MaxHoursPerIssue = 0 }, "max_hours_per_issue"},
		"hours negative": {func(p *Profile) { p.MaxHoursPerIssue = -1 }, "max_hours_per_issue"},
		"age missing":    {func(p *Profile) { p.MaxIssueAgeHours = 0 }, "max_issue_age_hours"},
		"stars negative": {func(p *Profile) { p.MinRepoStars = -1 }, "min_repo_stars"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			p := validProfile()
			tc.breakIt(p)
			wantErr(t, check(p), tc.want)
		})
	}
}

func TestAllErrorsReportedAtOnce(t *testing.T) {
	p := validProfile()
	p.About = ""
	p.Languages.Ok = []string{"pyhton"}
	wantErr(t, check(p), "about", "pyhton")
}

func TestLabelsGetNormalized(t *testing.T) {
	p := validProfile()
	p.LabelsWanted = []string{"Good-First-Issue", " good_first  issue", "Help Wanted"}
	if err := check(p); err != nil {
		t.Fatal(err)
	}
	wantStrings(t, p.LabelsWanted, []string{"good first issue", "help wanted"})
}

func TestProseFieldsKeepCase(t *testing.T) {
	p := validProfile()
	p.About = "  Prefer Go CLIs.  "
	p.Avoid = []string{" React components "}
	if err := check(p); err != nil {
		t.Fatal(err)
	}
	if p.About != "Prefer Go CLIs." {
		t.Errorf("About = %q, want trimmed with case kept", p.About)
	}
	wantStrings(t, p.Avoid, []string{"React components"})
}

func TestUnknownAttributeErrors(t *testing.T) {
	path := writeProfile(t, `
about: Backend dev.
languages:
  strong: [go]
max_hour_per_issue: 6
max_issue_age_hours: 48
`)
	_, err := Load(path)
	wantErr(t, err, "max_hour_per_issue")
}

// Guards the real config: editing profile.yaml into an invalid state fails here.
func TestValidProfileLoads(t *testing.T) {
	p, err := Load(filepath.Join("..", "..", "profile.yaml"))
	if err != nil {
		t.Fatalf("profile.yaml does not load: %v", err)
	}
	if !slices.Contains(p.Languages.Strong, "shell") {
		t.Errorf("expected bash to normalize to shell, got %q", p.Languages.Strong)
	}
}
