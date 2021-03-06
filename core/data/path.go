package data

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

func PathGetValue(value interface{}, path string) (interface{}, error) {

	if path == "" {
		return value, nil
	}

	var newVal interface{}
	var err error
	var newPath string

	if strings.HasPrefix(path, ".") {
		if objVal, ok := value.(map[string]interface{}); ok {
			newVal, newPath, err = pathGetSetObjValue(objVal, path, nil, false)
		} else if paramsVal, ok := value.(map[string]string); ok {
			newVal, newPath, err = pathGetSetParamsValue(paramsVal, path, nil, false)
		} else {
			return nil, fmt.Errorf("Unable to evaluate path: %s", path)
		}
	} else if strings.HasPrefix(path, "[") {
		newVal, newPath, err = pathGetSetArrayValue(value, path, nil, false)
	} else {
		return nil, fmt.Errorf("Unable to evaluate path: %s", path)
	}

	if err != nil {
		return nil, err
	}
	return PathGetValue(newVal, newPath)
}

func PathSetValue(attrValue interface{}, path string, value interface{}) error {
	if path == "" || attrValue == nil {
		return nil
	}

	var newVal interface{}
	var err error
	var newPath string

	if strings.HasPrefix(path, ".") {

		if objVal, ok := attrValue.(map[string]interface{}); ok {
			newVal, newPath, err = pathGetSetObjValue(objVal, path, value, true)
		} else if paramsVal, ok := attrValue.(map[string]string); ok {
			newVal, newPath, err = pathGetSetParamsValue(paramsVal, path, value, true)
		} else {
			return fmt.Errorf("Unable to evaluate path: %s", path)
		}
	} else if strings.HasPrefix(path, "[") {
		newVal, newPath, err = pathGetSetArrayValue(attrValue, path, value, true)
	} else {
		return fmt.Errorf("Unable to evaluate path: %s", path)
	}

	if err != nil {
		return err
	}
	return PathSetValue(newVal, newPath, value)
}

func getMapKey(s string) (string, int) {
	i := 0

	for i < len(s) {

		if s[i] == '.' || s[i] == '[' {
			return s[:i], i + 1
		}

		i += 1
	}

	return s, len(s) + 1
}

func pathGetSetArrayValue(obj interface{}, path string, value interface{}, set bool) (interface{}, string, error) {

	arrValue, valid := obj.([]interface{})
	if !valid {
		return nil, path, errors.New("'" + path + "' not an array")
	}

	closeIdx := strings.Index(path, "]")

	if closeIdx == -1 {
		return nil, path, errors.New("'" + path + "' not an array")
	}

	arrayIdx, err := strconv.Atoi(path[1:closeIdx])
	if err != nil {
		return nil, path, errors.New("Invalid array index: " + path[1:closeIdx])
	}

	if arrayIdx >= len(arrValue) {
		return nil, path, errors.New("Array index '" + path + "' out of range.")
	}

	if set && closeIdx == len(path)-1 {
		arrValue[arrayIdx] = value
		return nil, "", nil
	}

	return arrValue[arrayIdx], path[closeIdx+1:], nil
}

func pathGetSetObjValue(objValue map[string]interface{}, path string, value interface{}, set bool) (interface{}, string, error) {

	key, npIdx := getMapKey(path[1:])
	if set && key == path[1:] {
		//end of path so set the value
		objValue[key] = value
		return nil, "", nil
	}

	val, found := objValue[key]

	if !found {
		if path == "." + key {
			return nil, "", nil
		}

		return nil, "", errors.New("Invalid path '" + path + "'. path not found.")
	}

	return val, path[npIdx:], nil
}

func pathGetSetParamsValue(params map[string]string, path string, value interface{}, set bool) (interface{}, string, error) {

	key, _ := getMapKey(path[1:])
	if set && key == path[1:] {
		//end of path so set the value
		paramVal, err := CoerceToString(value)

		if err != nil {
			return nil, "", err
		}
		params[key] = paramVal
		return nil, "", nil
	}

	val, found := params[key]

	if !found {
		return nil, "", errors.New("Invalid path '" + path + "'. path not found.")
	}

	return val, "", nil
}

func PathDeconstruct(fullPath string) (attrName string, path string, err error) {

	idx := strings.IndexFunc(fullPath, isSep)

	if idx == -1 {
		return fullPath, "", nil
	}

	return fullPath[:idx], fullPath[idx:], nil
}
