package output

import (
	"encoding/json"
)

// JSONFormatter formats output as JSON.
type JSONFormatter struct {
	indent bool
}

// NewJSONFormatter creates a new JSON formatter.
func NewJSONFormatter(indent bool) *JSONFormatter {
	return &JSONFormatter{
		indent: indent,
	}
}

// Format formats the output as JSON.
func (f *JSONFormatter) Format(output StandardOutput, options FormatterOptions) (string, error) {
	// Apply filters if deep analysis is present
	if output.DeepAnalysis != nil && len(options.Filters) > 0 {
		for category, section := range output.DeepAnalysis.Sections {
			if section != nil {
				filteredItems := []AnalysisItem{}
				for _, item := range section.Items {
					include := true
					for _, filter := range options.Filters {
						if !filter(item) {
							include = false
							break
						}
					}
					if include {
						filteredItems = append(filteredItems, item)
					}
				}
				section.Items = filteredItems
				section.ItemCount = len(filteredItems)
				output.DeepAnalysis.Sections[category] = section
			}
		}
	}

	var data []byte
	var err error

	if f.indent {
		data, err = json.MarshalIndent(output, "", "  ")
	} else {
		data, err = json.Marshal(output)
	}

	if err != nil {
		return "", err
	}

	return string(data), nil
}
