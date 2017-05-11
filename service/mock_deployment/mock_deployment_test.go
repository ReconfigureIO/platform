package mock_deployment

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestDeploymentJSONDecodes(t *testing.T) {
	expected := Deployment{
		Container: ContainerConfig{
			Image:   "ubuntu",
			Command: "echo \"hello world\"",
		},
		Logs: LogsConfig{
			Group: "silly",
		},
	}
	actual := Deployment{}
	reference := "{\"container\":{\"image\":\"ubuntu\",\"command\":\"echo \\\"hello world\\\"\"},\"logs\":{\"group\":\"silly\"}}"

	err := json.Unmarshal([]byte(reference), &actual)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(expected, actual) {
		t.Fail()
	}
}
