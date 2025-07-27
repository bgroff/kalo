package main

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"kalo/src/panels"
)


type BruParser struct {
	scanner *bufio.Scanner
	line    string
	lineNum int
}

func NewBruParser(reader io.Reader) *BruParser {
	return &BruParser{
		scanner: bufio.NewScanner(reader),
		lineNum: 0,
	}
}

func (p *BruParser) Parse() (*panels.BruRequest, error) {
	request := &panels.BruRequest{
		Headers: make(map[string]string),
		Query:   make(map[string]string),
		Vars:    make(map[string]string),
		Auth:    panels.BruAuth{Values: make(map[string]string)},
	}

	for p.nextLine() {
		line := strings.TrimSpace(p.line)
		
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "meta {") {
			if err := p.parseMeta(request); err != nil {
				return nil, fmt.Errorf("line %d: %v", p.lineNum, err)
			}
		} else if strings.HasPrefix(line, "get {") || strings.HasPrefix(line, "post {") || strings.HasPrefix(line, "put {") || strings.HasPrefix(line, "delete {") || strings.HasPrefix(line, "patch {") {
			method := strings.ToUpper(strings.TrimSuffix(line, " {"))
			request.HTTP.Method = method
			if err := p.parseHTTP(request); err != nil {
				return nil, fmt.Errorf("line %d: %v", p.lineNum, err)
			}
		} else if strings.HasPrefix(line, "headers {") {
			if err := p.parseHeaders(request); err != nil {
				return nil, fmt.Errorf("line %d: %v", p.lineNum, err)
			}
		} else if strings.HasPrefix(line, "query {") {
			if err := p.parseQuery(request); err != nil {
				return nil, fmt.Errorf("line %d: %v", p.lineNum, err)
			}
		} else if strings.HasPrefix(line, "body:") {
			if err := p.parseBody(request, line); err != nil {
				return nil, fmt.Errorf("line %d: %v", p.lineNum, err)
			}
		} else if strings.HasPrefix(line, "auth:") {
			if err := p.parseAuth(request, line); err != nil {
				return nil, fmt.Errorf("line %d: %v", p.lineNum, err)
			}
		} else if strings.HasPrefix(line, "vars {") {
			if err := p.parseVars(request); err != nil {
				return nil, fmt.Errorf("line %d: %v", p.lineNum, err)
			}
		} else if strings.HasPrefix(line, "tests {") {
			if err := p.parseTests(request); err != nil {
				return nil, fmt.Errorf("line %d: %v", p.lineNum, err)
			}
		} else if strings.HasPrefix(line, "docs {") {
			if err := p.parseDocs(request); err != nil {
				return nil, fmt.Errorf("line %d: %v", p.lineNum, err)
			}
		}
	}

	return request, nil
}

func (p *BruParser) nextLine() bool {
	if p.scanner.Scan() {
		p.line = p.scanner.Text()
		p.lineNum++
		return true
	}
	return false
}

func (p *BruParser) parseMeta(request *panels.BruRequest) error {
	for p.nextLine() {
		line := strings.TrimSpace(p.line)
		if line == "}" {
			break
		}
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = p.unquoteString(value)

		switch key {
		case "name":
			request.Meta.Name = value
		case "type":
			request.Meta.Type = value
		case "seq":
			if seq, err := strconv.Atoi(value); err == nil {
				request.Meta.Seq = seq
			}
		}
	}
	return nil
}

func (p *BruParser) parseHTTP(request *panels.BruRequest) error {
	for p.nextLine() {
		line := strings.TrimSpace(p.line)
		if line == "}" {
			break
		}
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = p.unquoteString(value)

		switch key {
		case "url":
			request.HTTP.URL = value
		}
	}
	return nil
}

func (p *BruParser) parseHeaders(request *panels.BruRequest) error {
	for p.nextLine() {
		line := strings.TrimSpace(p.line)
		if line == "}" {
			break
		}
		if line == "" || strings.HasPrefix(line, "@") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = p.unquoteString(value)

		request.Headers[key] = value
	}
	return nil
}

func (p *BruParser) parseQuery(request *panels.BruRequest) error {
	for p.nextLine() {
		line := strings.TrimSpace(p.line)
		if line == "}" {
			break
		}
		if line == "" || strings.HasPrefix(line, "@") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = p.unquoteString(value)

		request.Query[key] = value
	}
	return nil
}

func (p *BruParser) parseBody(request *panels.BruRequest, headerLine string) error {
	bodyTypeRegex := regexp.MustCompile(`body:(\w+)`)
	matches := bodyTypeRegex.FindStringSubmatch(headerLine)
	if len(matches) > 1 {
		request.Body.Type = matches[1]
	}

	if !p.nextLine() {
		return nil
	}

	line := strings.TrimSpace(p.line)
	if line != "{" {
		return fmt.Errorf("expected '{' after body declaration")
	}

	var bodyContent strings.Builder
	braceCount := 1

	for p.nextLine() && braceCount > 0 {
		line := p.line
		
		for _, char := range line {
			if char == '{' {
				braceCount++
			} else if char == '}' {
				braceCount--
			}
		}

		if braceCount > 0 {
			bodyContent.WriteString(line)
			bodyContent.WriteString("\n")
		}
	}

	content := strings.TrimSpace(bodyContent.String())
	request.Body.Data = content
	return nil
}

func (p *BruParser) parseAuth(request *panels.BruRequest, headerLine string) error {
	authTypeRegex := regexp.MustCompile(`auth:(\w+)`)
	matches := authTypeRegex.FindStringSubmatch(headerLine)
	if len(matches) > 1 {
		request.Auth.Type = matches[1]
	}

	if !p.nextLine() {
		return nil
	}

	line := strings.TrimSpace(p.line)
	if line != "{" {
		return nil
	}

	for p.nextLine() {
		line := strings.TrimSpace(p.line)
		if line == "}" {
			break
		}
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = p.unquoteString(value)

		request.Auth.Values[key] = value
	}
	return nil
}

func (p *BruParser) parseVars(request *panels.BruRequest) error {
	for p.nextLine() {
		line := strings.TrimSpace(p.line)
		if line == "}" {
			break
		}
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = p.unquoteString(value)

		request.Vars[key] = value
	}
	return nil
}

func (p *BruParser) parseTests(request *panels.BruRequest) error {
	var content strings.Builder
	braceCount := 1

	for p.nextLine() && braceCount > 0 {
		line := p.line
		
		for _, char := range line {
			if char == '{' {
				braceCount++
			} else if char == '}' {
				braceCount--
			}
		}

		if braceCount > 0 {
			content.WriteString(line)
			content.WriteString("\n")
		}
	}

	request.Tests = strings.TrimSpace(content.String())
	return nil
}

func (p *BruParser) parseDocs(request *panels.BruRequest) error {
	var content strings.Builder
	braceCount := 1

	for p.nextLine() && braceCount > 0 {
		line := p.line
		
		for _, char := range line {
			if char == '{' {
				braceCount++
			} else if char == '}' {
				braceCount--
			}
		}

		if braceCount > 0 {
			content.WriteString(line)
			content.WriteString("\n")
		}
	}

	request.Docs = strings.TrimSpace(content.String())
	return nil
}

func (p *BruParser) unquoteString(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'") {
		return s[1 : len(s)-1]
	}
	return s
}