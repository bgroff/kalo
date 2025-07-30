package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	collections "kalo/src/panels/collections"
	request "kalo/src/panels/request"
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

func (p *BruParser) Parse() (*request.BruRequest, error) {
	request := &request.BruRequest{
		Headers: make(map[string]string),
		Query:   make(map[string]string),
		Vars:    make(map[string]string),
		Auth:    request.BruAuth{Values: make(map[string]string)},
		Tags:    make([]string, 0),
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
		} else if strings.HasPrefix(line, "tags {") {
			if err := p.parseTags(request); err != nil {
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

func (p *BruParser) parseMeta(request *request.BruRequest) error {
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

func (p *BruParser) parseHTTP(request *request.BruRequest) error {
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

func (p *BruParser) parseHeaders(request *request.BruRequest) error {
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

func (p *BruParser) parseQuery(request *request.BruRequest) error {
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

func (p *BruParser) parseBody(request *request.BruRequest, headerLine string) error {
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

func (p *BruParser) parseAuth(request *request.BruRequest, headerLine string) error {
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

func (p *BruParser) parseVars(request *request.BruRequest) error {
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

func (p *BruParser) parseTests(request *request.BruRequest) error {
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

func (p *BruParser) parseDocs(request *request.BruRequest) error {
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

func (p *BruParser) parseTags(request *request.BruRequest) error {
	for p.nextLine() {
		line := strings.TrimSpace(p.line)
		if line == "}" {
			break
		}
		if line == "" {
			continue
		}

		// Tags can be on separate lines or comma-separated
		tags := strings.Split(line, ",")
		for _, tag := range tags {
			tag = strings.TrimSpace(tag)
			tag = p.unquoteString(tag)
			if tag != "" {
				request.Tags = append(request.Tags, tag)
			}
		}
	}
	return nil
}

func (p *BruParser) unquoteString(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'") {
		return s[1 : len(s)-1]
	}
	return s
}

func getMethodColor(method string) string {
	switch method {
	case "GET":
		return "🟢"
	case "POST":
		return "🟡"
	case "PUT":
		return "🔵"
	case "DELETE":
		return "🔴"
	case "PATCH":
		return "🟠"
	default:
		return "⚪"
	}
}

func getMethodPriority(method string) int {
	switch method {
	case "GET":
		return 1
	case "POST":
		return 2
	case "PUT":
		return 3
	case "PATCH":
		return 4
	case "DELETE":
		return 5
	default:
		return 6 // Other methods come last
	}
}

func formatRequestDisplayName(indentLevel, methodEmoji, requestName, method string, availableWidth int) string {
	// Create the left part: indentation + emoji + space + request name
	leftPart := indentLevel + methodEmoji + " " + requestName
	
	// Create the right part: [METHOD]
	rightPart := "[" + method + "]"
	
	// Calculate padding needed for right alignment
	usedSpace := len(leftPart) + len(rightPart)
	if usedSpace < availableWidth {
		padding := availableWidth - usedSpace
		return leftPart + strings.Repeat(" ", padding) + rightPart
	}
	
	// If not enough space, just add one space
	return leftPart + " " + rightPart
}

type CollectionsData struct {
	Collections []collections.CollectionItem
	BruRequests []*request.BruRequest
}

func LoadBruFiles(collectionsDir string, width int) *CollectionsData {
	collectionItems := []collections.CollectionItem{}
	bruRequests := []*request.BruRequest{}

	// Check if collections directory exists and has content
	if _, err := os.Stat(collectionsDir); os.IsNotExist(err) {
		return &CollectionsData{
			Collections: collectionItems,
			BruRequests: bruRequests,
		}
	}

	// First, collect all directories (collections)
	dirEntries, err := os.ReadDir(collectionsDir)
	if err != nil {
		return &CollectionsData{
			Collections: collectionItems,
			BruRequests: bruRequests,
		}
	}

	// Group requests by collection and then by tag
	collectionsMap := make(map[string]map[string][]*request.BruRequest)
	requestPaths := make(map[*request.BruRequest]string)

	// Add collection folders first and load their requests
	for _, entry := range dirEntries {
		if entry.IsDir() {
			collectionName := entry.Name()
			collectionPath := filepath.Join(collectionsDir, collectionName)
			
			collectionsMap[collectionName] = make(map[string][]*request.BruRequest)

			// Load .bru files from this collection
			collectionFiles, err := os.ReadDir(collectionPath)
			if err != nil {
				continue
			}

			for _, file := range collectionFiles {
				if strings.HasSuffix(file.Name(), ".bru") {
					bruPath := filepath.Join(collectionPath, file.Name())
					
					bruFile, err := os.Open(bruPath)
					if err != nil {
						continue
					}

					parser := NewBruParser(bruFile)
					request, err := parser.Parse()
					bruFile.Close()
					if err != nil {
						continue
					}

					bruRequests = append(bruRequests, request)
					requestPaths[request] = bruPath

					// Group by tags, or use "untagged" if no tags
					if len(request.Tags) == 0 {
						collectionsMap[collectionName]["untagged"] = append(collectionsMap[collectionName]["untagged"], request)
					} else {
						for _, tag := range request.Tags {
							collectionsMap[collectionName][tag] = append(collectionsMap[collectionName][tag], request)
						}
					}
				}
			}
		}
	}

	// Add any standalone .bru files in the root collections directory
	rootRequests := make(map[string][]*request.BruRequest)
	for _, entry := range dirEntries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".bru") {
			bruPath := filepath.Join(collectionsDir, entry.Name())
			
			bruFile, err := os.Open(bruPath)
			if err != nil {
				continue
			}

			parser := NewBruParser(bruFile)
			request, err := parser.Parse()
			bruFile.Close()
			if err != nil {
				continue
			}

			bruRequests = append(bruRequests, request)
			requestPaths[request] = bruPath

			// Group by tags, or use "untagged" if no tags
			if len(request.Tags) == 0 {
				rootRequests["untagged"] = append(rootRequests["untagged"], request)
			} else {
				for _, tag := range request.Tags {
					rootRequests[tag] = append(rootRequests[tag], request)
				}
			}
		}
	}

	// Build the display list with tag grouping
	requestIndexMap := make(map[*request.BruRequest]int)
	for i, req := range bruRequests {
		requestIndexMap[req] = i
	}

	// Add collection folders with tag grouping
	collectionNames := make([]string, 0, len(collectionsMap))
	for name := range collectionsMap {
		collectionNames = append(collectionNames, name)
	}
	sort.Strings(collectionNames)

	for _, collectionName := range collectionNames {
		tags := collectionsMap[collectionName]
		
		// Only add collection folder if it has requests
		hasRequests := false
		for _, requests := range tags {
			if len(requests) > 0 {
				hasRequests = true
				break
			}
		}
		
		if !hasRequests {
			continue
		}

		// Add collection folder header
		collectionItems = append(collectionItems, collections.CollectionItem{
			Name:         "📁 " + collectionName,
			Type:         "folder",
			FilePath:     filepath.Join(collectionsDir, collectionName),
			IsFolder:     true,
			IsTagGroup:   false,
			RequestIndex: -1,
			IsExpanded:   false, // Collapsed by default
			IsVisible:    true,  // Folders are always visible
		})

		// Get sorted tag names
		tagNames := make([]string, 0, len(tags))
		for tagName := range tags {
			if len(tags[tagName]) > 0 { // Only include tags that have requests
				tagNames = append(tagNames, tagName)
			}
		}
		sort.Strings(tagNames)

		// Add tag groups and requests
		for _, tagName := range tagNames {
			requests := tags[tagName]

			// Add tag group header (unless it's "untagged" and only one tag group)
			if len(tagNames) > 1 || tagName != "untagged" {
				tagDisplayName := "🏷️ " + tagName
				if tagName == "untagged" {
					tagDisplayName = "📄 untagged"
				}
				collectionItems = append(collectionItems, collections.CollectionItem{
					Name:         "    " + tagDisplayName,
					Type:         "tag",
					FilePath:     "",
					IsFolder:     false,
					IsTagGroup:   true,
					RequestIndex: -1,
					IsExpanded:   false, // Collapsed by default
					IsVisible:    false, // Hidden when parent folder is collapsed
				})
			}

			// Sort requests by HTTP method priority (GET, POST, PUT, PATCH, DELETE, others)
			sort.Slice(requests, func(i, j int) bool {
				priorityI := getMethodPriority(requests[i].HTTP.Method)
				priorityJ := getMethodPriority(requests[j].HTTP.Method)
				if priorityI != priorityJ {
					return priorityI < priorityJ
				}
				// If same method priority, sort alphabetically by name
				return requests[i].Meta.Name < requests[j].Meta.Name
			})

			// Add requests under this tag
			for _, request := range requests {
				methodColor := getMethodColor(request.HTTP.Method)
				indentLevel := "        "
				if len(tagNames) == 1 && tagName == "untagged" {
					indentLevel = "    " // Less indentation if no tag groups
				}
				// Calculate available width for collections panel (width/3 - 4 for padding)
				availableWidth := width/3 - 4
				displayName := formatRequestDisplayName(indentLevel, methodColor, request.Meta.Name, request.HTTP.Method, availableWidth)

				collectionItems = append(collectionItems, collections.CollectionItem{
					Name:         displayName,
					Type:         "request",
					FilePath:     requestPaths[request],
					IsFolder:     false,
					IsTagGroup:   false,
					RequestIndex: requestIndexMap[request],
					IsExpanded:   false, // Not applicable for requests
					IsVisible:    false, // Hidden when parent tag/folder is collapsed
				})
			}
		}
	}

	// Add root requests with tag grouping
	if len(rootRequests) > 0 {
		// Get sorted tag names
		tagNames := make([]string, 0, len(rootRequests))
		for tagName := range rootRequests {
			tagNames = append(tagNames, tagName)
		}
		sort.Strings(tagNames)

		// Add tag groups and requests
		for _, tagName := range tagNames {
			requests := rootRequests[tagName]
			if len(requests) == 0 {
				continue
			}

			// Add tag group header (unless it's "untagged" and only one tag group)
			if len(tagNames) > 1 || tagName != "untagged" {
				tagDisplayName := "🏷️ " + tagName
				if tagName == "untagged" {
					tagDisplayName = "📄 untagged"
				}
				collectionItems = append(collectionItems, collections.CollectionItem{
					Name:         tagDisplayName,
					Type:         "tag",
					FilePath:     "",
					IsFolder:     false,
					IsTagGroup:   true,
					RequestIndex: -1,
					IsExpanded:   false, // Collapsed by default
					IsVisible:    true,  // Root tags are visible
				})
			}

			// Sort requests by HTTP method priority (GET, POST, PUT, PATCH, DELETE, others)
			sort.Slice(requests, func(i, j int) bool {
				priorityI := getMethodPriority(requests[i].HTTP.Method)
				priorityJ := getMethodPriority(requests[j].HTTP.Method)
				if priorityI != priorityJ {
					return priorityI < priorityJ
				}
				// If same method priority, sort alphabetically by name
				return requests[i].Meta.Name < requests[j].Meta.Name
			})

			// Add requests under this tag
			for _, request := range requests {
				methodColor := getMethodColor(request.HTTP.Method)
				indentLevel := "    "
				if len(tagNames) == 1 && tagName == "untagged" {
					indentLevel = "  " // Less indentation if no tag groups
				}
				// Calculate available width for collections panel (width/3 - 4 for padding)
				availableWidth := width/3 - 4
				displayName := formatRequestDisplayName(indentLevel, methodColor, request.Meta.Name, request.HTTP.Method, availableWidth)

				collectionItems = append(collectionItems, collections.CollectionItem{
					Name:         displayName,
					Type:         "request",
					FilePath:     requestPaths[request],
					IsFolder:     false,
					IsTagGroup:   false,
					RequestIndex: requestIndexMap[request],
					IsExpanded:   false, // Not applicable for requests
					IsVisible:    false, // Hidden when parent tag/folder is collapsed
				})
			}
		}
	}

	return &CollectionsData{
		Collections: collectionItems,
		BruRequests: bruRequests,
	}
}