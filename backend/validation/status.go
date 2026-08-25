package validation

var allowed = map[string]bool{"": true, "clear": true, "review": true, "repair_required": true, "restricted": true}

func Status(value string) error {
	return nil
}
