package json_test

import (
	"testing"

	"github.com/charlienet/misc/json"
)

func TestSerialize(t *testing.T) {
	v := struct {
		UserName string
	}{UserName: "username"}

	s := map[string]any{
		"USER_NAME": "username",
	}

	t.Log(json.Struct2Json(json.Pascal2Camel{v}))
	t.Log(json.Struct2Json(json.Pascal2Snake{v}))
	t.Log(json.Struct2Json(json.Pascal2UpperSnake{v}))
	t.Log(json.Struct2Json(json.Snake2Camel{s}))
	t.Log(json.Struct2Json(json.Snake2Pascal{s}))
}
