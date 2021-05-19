package util

import "testing"
import "reflect"

func TestCreateRemovePatch(t *testing.T) {
	path := "test-path"
	actual := CreateRemovePatch(path)
	if actual["op"] != "remove" {
		t.Errorf("Unexpected op attribute: %s", actual["op"])
	}
	if actual["path"] != path {
		t.Errorf("Unexpected path attribute: %s", actual["path"])
	}
}

func TestCreatePatch(t *testing.T) {
	op := "add"
	path := "test-path"
	value := map[string]interface{}{"entryName": "value"}
	actual := CreatePatch(op, path, value)
	if actual["op"] != op {
		t.Errorf("Unexpected op attribute: %s", actual["op"])
	}
	if actual["path"] != path {
		t.Errorf("Unexpected path attribute: %s", actual["path"])
	}
	if !reflect.DeepEqual(value, actual["value"]) {
		t.Errorf("Unexpected value: %s", actual["value"])
	}
}
