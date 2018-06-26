package deployment

import (
	"encoding/base64"
	"encoding/json"
	"reflect"
	"testing"
	"testing/quick"
)

func TestDeploymentJSONDecodes(t *testing.T) {
	expected := Deployment{
		Container: ContainerConfig{
			Image:   "ubuntu",
			Command: "echo \"hello world\"",
		},
		Logs: LogsConfig{
			Group:  "silly",
			Prefix: "deployment-256",
		},
	}
	actual := Deployment{}
	reference := "{\"container\":{\"image\":\"ubuntu\",\"command\":\"echo \\\"hello world\\\"\"},\"logs\":{\"group\":\"silly\", \"prefix\": \"deployment-256\"}}"

	err := json.Unmarshal([]byte(reference), &actual)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(expected, actual) {
		t.Fail()
	}
}

func TestDeploymentEncodes(t *testing.T) {
	expected := Deployment{
		CallbackUrl: "https://example.com/",
		Container: ContainerConfig{
			Image:   "398048034572.dkr.ecr.us-east-1.amazonaws.com/reconfigureio/platform/deployment:latest",
			Command: "echo wat",
		},
		Logs: LogsConfig{
			Group:  "josh-test-sdaccel",
			Prefix: "deployment-1",
		},
		Build: BuildConfig{
			ArtifactUrl: "Bar",
			Agfi:        "agfi-0e3d5b71759a2da10",
		},
	}
	s, err := expected.String()
	if err != nil {
		t.Fatal(err)
	}

	// if you change this, verify this is well formed JSON the command
	// line w/ `echo <string> | base64 -d`
	if !reflect.DeepEqual(s, "eyJjb250YWluZXIiOnsiaW1hZ2UiOiIzOTgwNDgwMzQ1NzIuZGtyLmVjci51cy1lYXN0LTEuYW1hem9uYXdzLmNvbS9yZWNvbmZpZ3VyZWlvL3BsYXRmb3JtL2RlcGxveW1lbnQ6bGF0ZXN0IiwiY29tbWFuZCI6ImVjaG8gd2F0In0sImxvZ3MiOnsiZ3JvdXAiOiJqb3NoLXRlc3Qtc2RhY2NlbCIsInByZWZpeCI6ImRlcGxveW1lbnQtMSJ9LCJjYWxsYmFja191cmwiOiJodHRwczovL2V4YW1wbGUuY29tLyIsImJ1aWxkIjp7ImFydGlmYWN0X3VybCI6IkJhciIsImFnZmkiOiJhZ2ZpLTBlM2Q1YjcxNzU5YTJkYTEwIn19Cg==") {
		t.Fail()
	}
}

func TestRoundTripEncode(t *testing.T) {
	identity := func(dep Deployment) Deployment {
		return dep
	}

	roundTrip := func(dep Deployment) Deployment {
		s, err := dep.String()
		if err != nil {
			t.Fatal(err)
		}

		ret := Deployment{}
		bytes, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			t.Fatal(err)
		}

		err = json.Unmarshal(bytes, &ret)
		if err != nil {
			t.Fatal(err)
		}
		return ret
	}

	if err := quick.CheckEqual(identity, roundTrip, nil); err != nil {
		t.Error(err)
	}
}
