package models

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestSerialiseDeserialise(t *testing.T) {
	report := &ReportV1{}
	reportContents := `{"moduleName":"fooModule","partName":"barPart","lutSummary":{"description":"CLB LUTs","used":70,"available":600577,"utilisation":0.01,"detail":{"lutLogic":{"description":"LUT as Logic","used":3,"available":600577,"utilisation":0.01},"lutMemory":{"description":"LUT as Memory","used":67,"available":394560,"utilisation":0.02}}},"regSummary":{"description":"CLB Registers","used":38,"available":1201154,"utilisation":0.01,"detail":{"regFlipFlop":{"description":"Register as Flip Flop","used":38,"available":1201154,"utilisation":0.01},"regLatch":{"description":"Register as Latch","used":0,"available":1201154,"utilisation":0}}},"blockRamSummary":{"description":"Block RAM Tile","used":0,"available":1024,"utilisation":0,"detail":{"blockRamB18":{"description":"RAMB18","used":0,"available":2048,"utilisation":0},"blockRamB36":{"description":"RAMB36/FIFO","used":0,"available":1024,"utilisation":0}}},"ultraRamSummary":{"description":"URAM","used":0,"available":470,"utilisation":0},"dspBlockSummary":{"description":"DSPs","used":0,"available":3474,"utilisation":0},"weightedAverage":{"description":"Weighted Average","used":318,"available":4569222,"utilisation":0.01}}`
	reportBytes := []byte(reportContents)
	err := json.Unmarshal(reportBytes, report)
	if err != nil {
		t.Error(err)
		return
	}

	var input interface{}
	err = json.Unmarshal(reportBytes, &input)

	newBytes, err := json.Marshal(report)
	if err != nil {
		t.Error(err)
		return
	}

	var output interface{}
	err = json.Unmarshal(newBytes, &output)

	// return from get with status should match the build we made at the start
	if !reflect.DeepEqual(input, output) {
		t.Fatalf("\nExpected: %+v\nGot: %+v\n", input, output)
		return
	}
}
