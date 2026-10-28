package profile

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"

	"agentic/issue_scout/internal/github"
	"go.yaml.in/yaml/v4"
)

type Profile struct {
	About            string              `yaml:"about"`
	Languages        LanguagePreferences `yaml:"languages"`
	LabelsWanted     []string            `yaml:"labels_wanted"`
	SkipRepos        []string            `yaml:"skip_repos"`
	Interests        []string            `yaml:"interests"`
	Avoid            []string            `yaml:"avoid"`
	MaxHoursPerIssue int                 `yaml:"max_hours_per_issue"`
	MaxIssueAgeHours int                 `yaml:"max_issue_age_hours"`
	MinRepoStars     int                 `yaml:"min_repo_stars"`
}

type LanguagePreferences struct {
	Strong []string `yaml:"strong"`
	Ok     []string `yaml:"ok"`
	Reject []string `yaml:"reject"`
}

// Render returns an XML encoded version
// of the profile
func (p *Profile) Render() string {
	var sb strings.Builder

	sb.WriteString("<profile>")

	if p.About != "" {
		fmt.Fprintf(&sb, "<about>%v</about>", p.About)
	}

	sb.WriteString("<languages>")
	if len(p.Languages.Strong) > 0 {
		strong := strings.Join(p.Languages.Strong, ", ")
		fmt.Fprintf(&sb, "<strong>%v</strong>", strong)
	}
	if len(p.Languages.Ok) > 0 {
		ok := strings.Join(p.Languages.Ok, ", ")
		fmt.Fprintf(&sb, "<ok>%v</ok>", ok)
	}
	if len(p.Languages.Reject) > 0 {
		reject := strings.Join(p.Languages.Reject, ", ")
		fmt.Fprintf(&sb, "<reject>%v</reject>", reject)
	}
	sb.WriteString("</languages>")

	if len(p.Interests) != 0 {
		sb.WriteString("<interests>")
		for _, i := range p.Interests {
			fmt.Fprintf(&sb, "<item>%s</item>", i)
		}
		sb.WriteString("</interests>")
	}

	if len(p.Avoid) != 0 {
		sb.WriteString("<avoid>")
		for _, a := range p.Avoid {
			fmt.Fprintf(&sb, "<item>%s</item>", a)
		}
		sb.WriteString("</avoid>")
	}

	sb.WriteString("</profile>")

	return sb.String()
}

// Load reads an YAML defining a profile
// and returns the populated struct
func Load(path string) (*Profile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read profile: %w", err)
	}

	var profile Profile

	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&profile); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	profile.normalize()

	err = profile.Validate()
	if err != nil {
		return nil, fmt.Errorf("invalid profile %s: %w", path, err)
	}

	return &profile, nil
}

// Validate reports every problem at once. It assumes normalize has already run.
func (p *Profile) Validate() error {
	var errs []error

	if len(p.Languages.Strong) == 0 {
		errs = append(errs, errors.New("provide at least one strong language"))
	}

	if p.MaxHoursPerIssue <= 0 {
		errs = append(errs, errors.New("`max_hours_per_issue` must be a positive integer"))
	}

	if p.MaxIssueAgeHours <= 0 {
		errs = append(errs, errors.New("`max_issue_age_hours` must be a positive integer"))
	}

	if p.MinRepoStars < 0 {
		errs = append(errs, errors.New("`min_repo_stars` must be a positive integer"))
	}

	if p.About == "" {
		errs = append(errs, errors.New("`about` must not be empty"))
	}

	seen := map[string]string{}
	check := func(tier string, langs []string) {
		for _, l := range langs {
			if _, ok := github.CanonicalLanguage(l); !ok {
				errs = append(errs, fmt.Errorf("unknown language %q in languages.%s", l, tier))
				continue
			}
			if prev, dup := seen[l]; dup {
				errs = append(errs, fmt.Errorf("language %q is in both %s and %s", l, prev, tier))
				continue
			}
			seen[l] = tier
		}
	}
	check("strong", p.Languages.Strong)
	check("ok", p.Languages.Ok)
	check("reject", p.Languages.Reject)

	return errors.Join(errs...)
}

// `normalize` rewrites the fields code compares against GitHub data
// (languages, labels, repos) into canonical form and deduplicates them.
// Fields the model reads are only trimmed, never lowercased.
// It never fails: unknown languages are kept so `Validate` can report them.
// Alters `Profile` in place.
func (p *Profile) normalize() {
	p.Languages.Strong = normalizeList(p.Languages.Strong, normalizeLanguage)
	p.Languages.Ok = normalizeList(p.Languages.Ok, normalizeLanguage)
	p.Languages.Reject = normalizeList(p.Languages.Reject, normalizeLanguage)

	p.LabelsWanted = normalizeList(p.LabelsWanted, NormalizeLabel)
	p.SkipRepos = normalizeList(p.SkipRepos, func(s string) string {
		return strings.ToLower(strings.TrimSpace(s))
	})

	p.About = strings.TrimSpace(p.About)
	p.Interests = normalizeList(p.Interests, strings.TrimSpace)
	p.Avoid = normalizeList(p.Avoid, strings.TrimSpace)
}

// `normalizeLanguage` maps a language or alias to its canonical name.
// Unknown languages come back lowercased and trimmed.
func normalizeLanguage(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if canonical, ok := github.CanonicalLanguage(s); ok {
		return canonical
	}
	return s
}

// `NormalizeLabel` lowercases a label and treats '-', '_' and runs of
// spaces as a single space, so "Good-First-Issue" == "good first issue".
// Apply it to issue labels too, or the comparison is one-sided.
func NormalizeLabel(s string) string {
	s = strings.ToLower(s)
	s = strings.NewReplacer("-", " ", "_", " ").Replace(s)
	return strings.Join(strings.Fields(s), " ")
}

// `normalizeList` applies `f` to each item, then drops empty results
// and duplicates, keeping first-seen order.
func normalizeList(items []string, f func(string) string) []string {
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		v := f(item)
		if v == "" {
			continue
		}
		if _, dup := seen[v]; dup {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

// `Prettify` returns a normalized, formatted version of
// `profile.yaml`.
func (p *Profile) Prettify() {
	panic("unimplementeed")
}
