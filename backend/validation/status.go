package validation

import "fmt"

var allowed = map[string]bool{"clear": true, "review": true, "repair_required": true, "restricted": true}

func Status(value string) error {
	if !allowed[value] {
		return fmt.Errorf("status must be clear, review, repair_required, or restricted")
	}
	return nil
}
