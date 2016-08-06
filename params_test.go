package httprouter

import (
	"net/http"
	"testing"
)

func TestParams(t *testing.T) {
	ps := Params{
		Param{"param1", "value1"},
		Param{"param2", "value2"},
		Param{"param3", "value3"},
	}
	for i := range ps {
		if val := ps.ByName(ps[i].Key); val != ps[i].Value {
			t.Errorf("Wrong value for %s: Got %s; Want %s", ps[i].Key, val, ps[i].Value)
		}
	}
	if val := ps.ByName("noKey"); val != "" {
		t.Errorf("Expected empty string for not found key; got: %s", val)
	}
}

func TestContextParams(t *testing.T) {
	ps := Params{
		Param{"param1", "value1"},
		Param{"param2", "value2"},
		Param{"param3", "value3"},
	}
	req, _ := http.NewRequest("GET", "/whatever", nil)
	req = req.WithContext(WithParams(req.Context(), ps))
	for i := range ps {
		if val := GetParam(req, ps[i].Key); val != ps[i].Value {
			t.Errorf("Wrong value for %s: Got %s; Want %s", ps[i].Key, val, ps[i].Value)
		}
	}
	if val := GetParam(req, "noKey"); val != "" {
		t.Errorf("Expected empty string for not found key; got: %s", val)
	}
}

func TestMissingContextParams(t *testing.T) {
	req, _ := http.NewRequest("GET", "/whatever", nil)
	if value := GetParam(req, "whatever"); value != "" {
		t.Fatalf("wrong empty parameter value: got %v", value)
	}
	ps := GetParams(req)
	if value := ps.ByName("whatever"); value != "" {
		t.Fatalf("wrong empty parameter value: got %v", value)
	}
}
