package parser

import (
	"fmt"
	"regexp"
	"strings"
)

type DirParser struct {
	regex      *regexp.Regexp
	LabelNames []string
}

// reToken matches the special tokens in a template: a {label} placeholder or
// a * wildcard. Everything else is treated as a literal run.
var reToken = regexp.MustCompile(`\{([a-zA-Z0-9_]+)}|\*`)

// NewDirParser creates the parser and detects labels from the template.
// The template is tokenized into literal runs, "*" wildcards and "{label}"
// placeholders; literal runs are regex-escaped so that regex metacharacters
// in the template (e.g. ".") are matched literally rather than as regex
// syntax.
func NewDirParser(template string) (*DirParser, error) {
	var labelNames []string
	seenLabels := make(map[string]bool)

	var sb strings.Builder
	sb.WriteString("^")

	lastEnd := 0
	for _, loc := range reToken.FindAllStringSubmatchIndex(template, -1) {
		start, end := loc[0], loc[1]
		sb.WriteString(regexp.QuoteMeta(template[lastEnd:start]))

		if loc[2] >= 0 {
			// {label} placeholder
			label := template[loc[2]:loc[3]]
			if seenLabels[label] {
				return nil, fmt.Errorf("duplicate label name %q in template", label)
			}
			seenLabels[label] = true
			labelNames = append(labelNames, label)
			fmt.Fprintf(&sb, "(?P<%s>.+?)", label)
		} else {
			// * wildcard
			sb.WriteString(`(?:.+?)`)
		}

		lastEnd = end
	}
	sb.WriteString(regexp.QuoteMeta(template[lastEnd:]))
	sb.WriteString("$")

	r, err := regexp.Compile(sb.String())
	if err != nil {
		return nil, fmt.Errorf("failed to compile regex from template: %w", err)
	}

	return &DirParser{
		regex:      r,
		LabelNames: labelNames,
	}, nil
}

// Parse extracts label values from a directory name based on the template
func (p *DirParser) Parse(dirName string) ([]string, bool) {
	match := p.regex.FindStringSubmatch(dirName)
	if match == nil {
		return nil, false
	}

	values := make([]string, len(p.LabelNames))
	for i, labelName := range p.LabelNames {
		idx := p.regex.SubexpIndex(labelName)
		if idx >= 0 && idx < len(match) {
			values[i] = match[idx]
		}
	}
	return values, true
}
