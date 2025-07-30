package panels

import (
	"encoding/json"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/itchyny/gojq"
)

type FilterType string

const (
	JQFilter          FilterType = "jq"
	CollectionsFilter FilterType = "collections"
)

type filterMsg struct {
	filterType FilterType
	result     string
	err        error
}

type FilterManager struct {
	Mode                  bool
	FilterType            FilterType
	Input                 string
	CursorPos             int
	ShowSuggestions       bool
	SelectedSuggestion    int
	JqSuggestions         []string
	LastJQFilter          string
	LastCollectionsFilter string
	AppliedJQFilter       string
	OriginalCollections   []CollectionItem
}

func NewFilterManager() *FilterManager {
	return &FilterManager{}
}

func (f *FilterManager) StartFilter(filterType FilterType) {
	f.Mode = true
	f.FilterType = filterType
	
	// Restore previous filter input
	switch filterType {
	case JQFilter:
		f.Input = f.LastJQFilter
		f.ShowSuggestions = true
		f.SelectedSuggestion = 0
	case CollectionsFilter:
		f.Input = f.LastCollectionsFilter
	default:
		f.Input = ""
	}
	
	// Set cursor to end of input
	f.CursorPos = len(f.Input)
}

func (f *FilterManager) ExitFilter() []CollectionItem {
	// Save the current filter input before exiting
	switch f.FilterType {
	case JQFilter:
		f.LastJQFilter = f.Input
	case CollectionsFilter:
		f.LastCollectionsFilter = f.Input
	}
	
	f.Mode = false
	f.Input = ""
	f.ShowSuggestions = false
	f.SelectedSuggestion = 0
	f.CursorPos = 0
	
	// Return original collections if needed for restoration
	var restore []CollectionItem
	if f.FilterType == CollectionsFilter && len(f.OriginalCollections) > 0 {
		restore = make([]CollectionItem, len(f.OriginalCollections))
		copy(restore, f.OriginalCollections)
		f.OriginalCollections = nil
	}
	
	return restore
}

func (f *FilterManager) ApplyJQFilter(originalResponse string) tea.Cmd {
	if originalResponse == "" || f.Input == "" {
		return nil
	}

	filter := f.Input
	originalData := originalResponse

	return func() tea.Msg {
		// Parse the jq query
		query, err := gojq.Parse(filter)
		if err != nil {
			return filterMsg{filterType: JQFilter, err: fmt.Errorf("jq parse error: %v", err)}
		}

		// Parse the JSON
		var jsonData interface{}
		if err := json.Unmarshal([]byte(originalData), &jsonData); err != nil {
			return filterMsg{filterType: JQFilter, err: fmt.Errorf("JSON parse error: %v", err)}
		}

		// Apply the filter
		iter := query.Run(jsonData)
		var results []interface{}
		for {
			v, ok := iter.Next()
			if !ok {
				break
			}
			if err, ok := v.(error); ok {
				return filterMsg{filterType: JQFilter, err: fmt.Errorf("jq filter error: %v", err)}
			}
			results = append(results, v)
		}

		// Format the result
		var resultData interface{}
		if len(results) == 0 {
			resultData = nil
		} else if len(results) == 1 {
			resultData = results[0]
		} else {
			resultData = results
		}

		// Convert back to pretty JSON
		resultBytes, err := json.MarshalIndent(resultData, "", "  ")
		if err != nil {
			return filterMsg{filterType: JQFilter, err: fmt.Errorf("JSON marshal error: %v", err)}
		}

		return filterMsg{filterType: JQFilter, result: string(resultBytes)}
	}
}

func (f *FilterManager) ApplyCollectionsFilter(originalCollections []CollectionItem) []CollectionItem {
	// Store original if not already stored
	if len(f.OriginalCollections) == 0 {
		f.OriginalCollections = make([]CollectionItem, len(originalCollections))
		copy(f.OriginalCollections, originalCollections)
	}

	// If filter is empty, restore all collections and collapse them
	if f.Input == "" {
		collections := make([]CollectionItem, len(f.OriginalCollections))
		copy(collections, f.OriginalCollections)
		
		// Reset expansion state to collapsed
		for i := range collections {
			if collections[i].IsFolder || collections[i].IsTagGroup {
				collections[i].IsExpanded = false
			}
		}
		
		return collections
	}

	filter := strings.ToLower(f.Input)
	var filteredCollections []CollectionItem
	var currentFolder *CollectionItem
	var currentTagGroup *CollectionItem
	var hasMatchingRequests bool
	
	for _, item := range f.OriginalCollections {
		if item.IsFolder {
			// Store current folder, add it later if it has matching requests
			currentFolder = &item
			currentTagGroup = nil
			hasMatchingRequests = false
		} else if item.IsTagGroup {
			// Store current tag group, add it later if it has matching requests  
			currentTagGroup = &item
			hasMatchingRequests = false
		} else {
			// This is a request - check if it matches the filter
			if strings.Contains(strings.ToLower(item.Name), filter) {
				// Add the folder if we haven't added it yet and expand it
				if currentFolder != nil && !hasMatchingRequests {
					expandedFolder := *currentFolder
					expandedFolder.IsExpanded = true
					filteredCollections = append(filteredCollections, expandedFolder)
				}
				// Add the tag group if we haven't added it yet and expand it
				if currentTagGroup != nil && !hasMatchingRequests {
					expandedTagGroup := *currentTagGroup
					expandedTagGroup.IsExpanded = true
					filteredCollections = append(filteredCollections, expandedTagGroup)
				}
				// Add the matching request
				filteredCollections = append(filteredCollections, item)
				hasMatchingRequests = true
			}
		}
	}

	return filteredCollections
}

func (f *FilterManager) GenerateJQSuggestions(originalResponse string) {
	f.JqSuggestions = []string{}
	
	// Basic jq operations
	basicSuggestions := []string{
		".",
		".[]",
		".[0]",
		"length",
		"keys",
		"keys[]",
		"type",
		"empty",
		"map(.)",
		"select(.)",
		"sort",
		"reverse",
		"unique",
		"group_by(.)",
		"min",
		"max",
		"add",
	}
	
	f.JqSuggestions = append(f.JqSuggestions, basicSuggestions...)
	
	// Extract field suggestions from current JSON
	if originalResponse != "" {
		fieldSuggestions := f.extractJSONFields(originalResponse, "")
		f.JqSuggestions = append(f.JqSuggestions, fieldSuggestions...)
	}
}

func (f *FilterManager) extractJSONFields(jsonStr string, prefix string) []string {
	var suggestions []string
	var data interface{}
	
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return suggestions
	}
	
	suggestions = f.extractFieldsFromValue(data, prefix, 0)
	return suggestions
}

func (f *FilterManager) extractFieldsFromValue(value interface{}, prefix string, depth int) []string {
	var suggestions []string
	
	// Limit depth to avoid infinite recursion and too many suggestions
	if depth > 3 {
		return suggestions
	}
	
	switch v := value.(type) {
	case map[string]interface{}:
		for key, subValue := range v {
			fieldPath := prefix + "." + key
			if prefix == "" {
				fieldPath = "." + key
			}
			suggestions = append(suggestions, fieldPath)
			
			// Add array access for arrays
			if _, isArray := subValue.([]interface{}); isArray {
				suggestions = append(suggestions, fieldPath+"[]")
				suggestions = append(suggestions, fieldPath+"[0]")
			}
			
			// Recursively get nested fields (limited depth)
			if depth < 2 {
				nestedSuggestions := f.extractFieldsFromValue(subValue, fieldPath, depth+1)
				suggestions = append(suggestions, nestedSuggestions...)
			}
		}
	case []interface{}:
		if len(v) > 0 {
			// Get fields from first array element
			if depth < 2 {
				nestedSuggestions := f.extractFieldsFromValue(v[0], prefix+"[]", depth+1)
				suggestions = append(suggestions, nestedSuggestions...)
			}
		}
	}
	
	return suggestions
}

func (f *FilterManager) GetFilteredSuggestions(originalResponse string) []string {
	// Regenerate suggestions based on current input context
	f.generateContextualSuggestions(originalResponse)
	
	if f.Input == "" {
		return f.JqSuggestions
	}
	
	var filtered []string
	input := strings.ToLower(f.Input)
	
	for _, suggestion := range f.JqSuggestions {
		if strings.Contains(strings.ToLower(suggestion), input) ||
		   strings.HasPrefix(strings.ToLower(suggestion), input) {
			filtered = append(filtered, suggestion)
		}
	}
	
	return filtered
}

func (f *FilterManager) generateContextualSuggestions(originalResponse string) {
	// Start with basic suggestions
	f.GenerateJQSuggestions(originalResponse)
	
	// Add contextual suggestions based on current input
	if originalResponse != "" && f.Input != "" {
		contextSuggestions := f.getContextualCompletions(f.Input, originalResponse)
		f.JqSuggestions = append(f.JqSuggestions, contextSuggestions...)
	}
}

func (f *FilterManager) getContextualCompletions(input string, originalResponse string) []string {
	var suggestions []string
	
	// If input ends with a dot, suggest fields for that path
	if strings.HasSuffix(input, ".") {
		pathSuggestions := f.getFieldsForPath(input[:len(input)-1], originalResponse)
		for _, field := range pathSuggestions {
			suggestions = append(suggestions, input+field)
		}
	}
	
	// If input looks like a partial field path, suggest completions
	if strings.Contains(input, ".") && !strings.HasSuffix(input, ".") {
		lastDotIndex := strings.LastIndex(input, ".")
		if lastDotIndex >= 0 {
			basePath := input[:lastDotIndex]
			partialField := input[lastDotIndex+1:]
			
			pathFields := f.getFieldsForPath(basePath, originalResponse)
			for _, field := range pathFields {
				if strings.HasPrefix(strings.ToLower(field), strings.ToLower(partialField)) {
					suggestions = append(suggestions, basePath+"."+field)
				}
			}
		}
	}
	
	return suggestions
}

func (f *FilterManager) getFieldsForPath(path string, originalResponse string) []string {
	var fields []string
	var data interface{}
	
	if err := json.Unmarshal([]byte(originalResponse), &data); err != nil {
		return fields
	}
	
	// Navigate to the specified path
	current := data
	if path != "" && path != "." {
		pathParts := strings.Split(strings.TrimPrefix(path, "."), ".")
		for _, part := range pathParts {
			if part == "" {
				continue
			}
			
			// Handle array notation
			if strings.HasSuffix(part, "[]") {
				part = part[:len(part)-2]
				if obj, ok := current.(map[string]interface{}); ok {
					if arr, ok := obj[part].([]interface{}); ok && len(arr) > 0 {
						current = arr[0] // Use first element as template
					} else {
						return fields
					}
				} else {
					return fields
				}
			} else {
				if obj, ok := current.(map[string]interface{}); ok {
					if val, exists := obj[part]; exists {
						current = val
					} else {
						return fields
					}
				} else {
					return fields
				}
			}
		}
	}
	
	// Extract fields from current object
	if obj, ok := current.(map[string]interface{}); ok {
		for key, value := range obj {
			fields = append(fields, key)
			
			// Add array notation for arrays
			if _, isArray := value.([]interface{}); isArray {
				fields = append(fields, key+"[]")
				fields = append(fields, key+"[0]")
			}
		}
	}
	
	return fields
}

func (f *FilterManager) FindPreviousWordBoundary(input string, pos int) int {
	if pos <= 0 {
		return 0
	}
	
	// Word boundaries for jq expressions: ., [, ]
	wordBoundaries := []rune{'.', '[', ']'}
	
	// Start from position before cursor
	for i := pos - 1; i >= 0; i-- {
		char := rune(input[i])
		for _, boundary := range wordBoundaries {
			if char == boundary {
				// Found a boundary, position cursor after it (unless at start)
				if i == 0 {
					return 0
				}
				return i + 1
			}
		}
	}
	
	// No boundary found, go to start
	return 0
}

func (f *FilterManager) FindNextWordBoundary(input string, pos int) int {
	if pos >= len(input) {
		return len(input)
	}
	
	// Word boundaries for jq expressions: ., [, ]
	wordBoundaries := []rune{'.', '[', ']'}
	
	// Start from current position
	for i := pos; i < len(input); i++ {
		char := rune(input[i])
		for _, boundary := range wordBoundaries {
			if char == boundary {
				// Found a boundary, position cursor at it
				return i
			}
		}
	}
	
	// No boundary found, go to end
	return len(input)
}

func (f *FilterManager) UpdateVisibility(collections []CollectionItem) []CollectionItem {
	for i := range collections {
		item := &collections[i]
		
		if item.IsFolder {
			// Folders are always visible
			item.IsVisible = true
		} else if item.IsTagGroup {
			// Tag groups are visible if their parent folder is expanded (or if they're root tags)
			item.IsVisible = true
			
			// Find parent folder
			for j := i - 1; j >= 0; j-- {
				if collections[j].IsFolder {
					item.IsVisible = collections[j].IsExpanded
					break
				}
			}
		} else {
			// Requests are visible if their parent tag group is expanded
			item.IsVisible = false
			
			// Find parent tag group
			for j := i - 1; j >= 0; j-- {
				parentItem := &collections[j]
				if parentItem.IsTagGroup {
					item.IsVisible = parentItem.IsExpanded
					break
				}
				if parentItem.IsFolder {
					// If we hit a folder before finding a tag group, this is a direct child of folder
					item.IsVisible = parentItem.IsExpanded
					break
				}
			}
		}
	}
	
	return collections
}

func (f *FilterManager) Reset(filterType FilterType) {
	switch filterType {
	case JQFilter:
		f.LastJQFilter = ""
		f.AppliedJQFilter = ""
	case CollectionsFilter:
		f.LastCollectionsFilter = ""
		f.OriginalCollections = nil
	}
}