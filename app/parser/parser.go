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

// NewDirParser creates the parser and detects labels from the template
func NewDirParser(template string) (*DirParser, error) {
	reTags := regexp.MustCompile(`\{([a-zA-Z0-9_]+)}`)
	matches := reTags.FindAllStringSubmatch(template, -1)

	var labelNames []string
	for _, match := range matches {
		labelNames = append(labelNames, match[1])
	}

	regexStr := template
	regexStr = strings.ReplaceAll(regexStr, "*", `(?:.+?)`)

	for _, label := range labelNames {
		placeholder := fmt.Sprintf("{%s}", label)
		replacement := fmt.Sprintf("(?P<%s>.+?)", label)
		regexStr = strings.Replace(regexStr, placeholder, replacement, 1)
	}

	regexStr = "^" + regexStr + "$"

	r, err := regexp.Compile(regexStr)
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
