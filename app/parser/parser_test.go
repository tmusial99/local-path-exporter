package parser

import "testing"

func TestDirParser_DefaultTemplate(t *testing.T) {
	p, err := NewDirParser("pvc-*_{pvc_namespace}_{pvc_name}")
	if err != nil {
		t.Fatalf("NewDirParser() error = %v", err)
	}

	wantLabels := []string{"pvc_namespace", "pvc_name"}
	if len(p.LabelNames) != len(wantLabels) {
		t.Fatalf("LabelNames = %v, want %v", p.LabelNames, wantLabels)
	}
	for i, name := range wantLabels {
		if p.LabelNames[i] != name {
			t.Errorf("LabelNames[%d] = %q, want %q", i, p.LabelNames[i], name)
		}
	}

	values, ok := p.Parse("pvc-1234abcd-5678-90ef-ghij-klmnopqrstuv_team-a_data-vol")
	if !ok {
		t.Fatalf("Parse() ok = false, want true")
	}
	if want := []string{"team-a", "data-vol"}; values[0] != want[0] || values[1] != want[1] {
		t.Errorf("Parse() = %v, want %v", values, want)
	}
}

func TestDirParser_WildcardIgnoresSegment(t *testing.T) {
	p, err := NewDirParser("pvc-*_{pvc_namespace}_{pvc_name}")
	if err != nil {
		t.Fatalf("NewDirParser() error = %v", err)
	}

	// Different UUID-like segments should still match and the wildcard
	// content itself must not leak into the extracted label values.
	values, ok := p.Parse("pvc-aaaa-bbbb_team-b_other-vol")
	if !ok {
		t.Fatalf("Parse() ok = false, want true")
	}
	if values[0] != "team-b" || values[1] != "other-vol" {
		t.Errorf("Parse() = %v, want [team-b other-vol]", values)
	}
}

func TestDirParser_NoMatch(t *testing.T) {
	p, err := NewDirParser("pvc-*_{pvc_namespace}_{pvc_name}")
	if err != nil {
		t.Fatalf("NewDirParser() error = %v", err)
	}

	values, ok := p.Parse("not-a-matching-directory")
	if ok {
		t.Fatalf("Parse() ok = true, want false")
	}
	if values != nil {
		t.Errorf("Parse() values = %v, want nil", values)
	}
}

func TestDirParser_EscapesRegexLiterals(t *testing.T) {
	p, err := NewDirParser("a.b-{x}")
	if err != nil {
		t.Fatalf("NewDirParser() error = %v", err)
	}

	// "." in the template is literal, so "axb-1" must NOT match.
	if _, ok := p.Parse("axb-1"); ok {
		t.Errorf("Parse(%q) ok = true, want false (literal '.' must not match any char)", "axb-1")
	}

	// "a.b-1" matches because "." is literal.
	values, ok := p.Parse("a.b-1")
	if !ok {
		t.Fatalf("Parse(%q) ok = false, want true", "a.b-1")
	}
	if len(values) != 1 || values[0] != "1" {
		t.Errorf("Parse(%q) = %v, want [1]", "a.b-1", values)
	}
}

func TestDirParser_DuplicateLabelName(t *testing.T) {
	_, err := NewDirParser("{x}_{x}")
	if err == nil {
		t.Fatalf("NewDirParser() error = nil, want error for duplicate label name")
	}
}
