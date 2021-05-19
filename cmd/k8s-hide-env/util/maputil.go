package util

func ExtractMapList(container map[string]interface{}, key string) []map[string]interface{} {
	value := container[key]
	if value == nil {
		return nil
	}

	castValue, ok := value.([]map[string]interface{})
	if ok {
		return castValue
	}

	castValue2, ok2 := value.([]interface{})
	if ok2 {
		var retVal []map[string]interface{}
		for _, entry := range castValue2 {
			retVal = append(retVal, entry.(map[string]interface{}))
		}
		return retVal
	}

	return nil
}

func ExtractStringList(container map[string]interface{}, key string) []string {
	value := container[key]
	if value == nil {
		return nil
	}

	castValue, ok := value.([]string)
	if ok {
		return castValue
	}

	castValue2, ok2 := value.([]interface{})
	if ok2 {
		var retVal []string
		for _, e := range castValue2 {
			retVal = append(retVal, e.(string))
		}
		return retVal
	}

	return nil
}

func ExtractMap(container map[string]interface{}, key string) map[string]interface{} {
	value := container[key]
	if value == nil {
		return nil
	}

	castValue, ok := value.(map[string]interface{})
	if !ok {
		return nil
	}

	return castValue
}

func GetArrayIndex(namedMapList []map[string]interface{}, name string) int {
	for index, entry := range namedMapList {
		if entry["name"] == name {
			return index
		}
	}
	return -1
}
