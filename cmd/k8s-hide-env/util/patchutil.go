package util

func CreateRemovePatch(path string) map[string]interface{} {
	return map[string]interface{}{
		"op":   "remove",
		"path": path,
	}
}

func CreatePatch(op string, path string, value interface{}) map[string]interface{} {
	return map[string]interface{}{
		"op":    op,
		"path":  path,
		"value": value,
	}
}
