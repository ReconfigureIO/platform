// +build

package api

import (
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
)

func TestJSONValidation(t *testing.T) {
	body := "{\"moduleName\":\"fooModule\",\"partName\":\"barPart\",\"lutSummary\":{\"description\":\"CLB LUTs\",\"used\":70,\"available\":600577,\"utilisation\":0.01,\"detail\":{\"lutLogic\":{\"description\":\"LUT as Logic\",\"used\":3,\"available\":600577,\"utilisation\":0.01},\"lutMemory\":{\"description\":\"LUT as Memory\",\"used\":67,\"available\":394560,\"utilisation\":0.02}}},\"regSummary\":{\"description\":\"CLB Registers\",\"used\":38,\"available\":1201154,\"utilisation\":0.01,\"detail\":{\"regFlipFlop\":{\"description\":\"Register as Flip Flop\",\"used\":38,\"available\":1201154,\"utilisation\":0.01},\"regLatch\":{\"description\":\"Register as Latch\",\"used\":0,\"available\":1201154,\"utilisation\":0}}},\"blockRamSummary\":{\"description\":\"Block RAM Tile\",\"used\":0,\"available\":1024,\"utilisation\":0,\"detail\":{\"blockRamB36\":{\"description\":\"RAMB36/FIFO\",\"used\":0,\"available\":1024,\"utilisation\":0},\"blockRamB18\":{\"description\":\"RAMB18\",\"used\":0,\"available\":2048,\"utilisation\":0}}},\"ultraRamSummary\":{\"description\":\"URAM\",\"used\":0,\"available\":470,\"utilisation\":0},\"dspBlockSummary\":{\"description\":\"DSPs\",\"used\":0,\"available\":3474,\"utilisation\":0},\"weightedAverage\":{\"description\":\"Weighted Average\",\"used\":318,\"available\":4569222,\"utilisation\":0.01}}"

	report, err := ValidateJson(c)

	if report != body {
		t.Errorf("\nExpected: %+v\nGot:      %+v\n", body, report)
	}
}

func TestInvalidJSONValidation(t *testing.T) {
	body := "{foo:bar}"

	_, err := ValidateJson(c)

	if err == nil {
		t.Error("Expected an error to be returned")
	}
}
