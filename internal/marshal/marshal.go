package marshal

import (
	"encoding/json"
	"os"
)

// Parses a JSON file
func UnmarshalJsonFile(filename string, out interface{}) error {
	
	// Attempt to read the contents of the JSON file
	jsonData, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	
	// Attempt to parse the JSON data
	if err := json.Unmarshal(jsonData, out); err != nil {
		return err
	}
	
	return nil
}
