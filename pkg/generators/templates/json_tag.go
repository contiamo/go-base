package templates

import "fmt"

// JSONTag is generates a struct json tag with the given name.
// Set omit = true to add `omitempty` to the json tag.
func JSONTag(name string, omit bool) string {
	if omit {
		return fmt.Sprintf("`json:\"%s\"`", name)
	}
	return fmt.Sprintf("`json:\"%s,omitempty\"`", name)
}
