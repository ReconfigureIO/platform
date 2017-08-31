// +build integration

package models

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestSerialiseDeserialise(t *testing.T) {
	report := &ReportV1{}
	reportcontents := `{"moduleName":"fooModule","partName":"barPart","lutSummary":{"description":"CLB LUTs","used":70,"available":600577,"utilisation":0.01,"detail":{"lutLogic":{"description":"LUT as Logic","used":3,"available":600577,"utilisation":0.01},"lutMemory":{"description":"LUT as Memory","used":67,"available":394560,"utilisation":0.02}}},"regSummary":{"description":"CLB Registers","used":38,"available":1201154,"utilisation":0.01,"detail":{"regFlipFlop":{"description":"Register as Flip Flop","used":38,"available":1201154,"utilisation":0.01},"regLatch":{"description":"Register as Latch","used":0,"available":1201154,"utilisation":0}}},"blockRamSummary":{"description":"Block RAM Tile","used":0,"available":1024,"utilisation":0,"detail":{"blockRamB36":{"description":"RAMB36/FIFO","used":0,"available":1024,"utilisation":0},"blockRamB18":{"description":"RAMB18","used":0,"available":2048,"utilisation":0}}},"ultraRamSummary":{"description":"URAM","used":0,"available":470,"utilisation":0},"dspBlockSummary":{"description":"DSPs","used":0,"available":3474,"utilisation":0},"weightedAverage":{"description":"Weighted Average","used":318,"available":4569222,"utilisation":0.01}}`
	reportbytes := []byte(reportcontents)
	err := json.Unmarshal(reportbytes, report)
	if err != nil {
		t.Error(err)
		return
	}

	expectedReport := &ReportV1{
		ModuleName: "fooModule",
		PartName:   "barPart",
		LutSummary: GroupSummary{
			Description: "CLB LUTs",
			Used:        70,
			Available:   600577,
			Utilisation: 0.01,
			Detail: PartDetails{
				"lutLogic": PartDetail{
					Description: "LUT as Logic",
					Used:        3,
					Available:   600577,
					Utilisation: 0.01,
				},
				"lutMemory": PartDetail{
					Description: "LUT as Memory",
					Used:        67,
					Available:   394560,
					Utilisation: 0.02,
				},
			},
		},
		RegSummary: GroupSummary{
			Description: "CLB Registers",
			Used:        38,
			Available:   1201154,
			Utilisation: 0.01,
			Detail: PartDetails{
				"regFlipFlop": PartDetail{
					Description: "Register as Flip Flop",
					Used:        38,
					Available:   1201154,
					Utilisation: 0.01,
				},
				"regLatch": PartDetail{
					Description: "Register as Latch",
					Used:        0,
					Available:   1201154,
					Utilisation: 0,
				},
			},
		},
		BlockRamSummary: GroupSummary{
			Description: "Block RAM Tile",
			Used:        0,
			Available:   1024,
			Utilisation: 0,
			Detail: PartDetails{
				"blockRamB36": PartDetail{
					Description: "RAMB36/FIFO",
					Used:        0,
					Available:   1024,
					Utilisation: 0,
				},
				"blockRamB18": PartDetail{
					Description: "RAMB18",
					Used:        0,
					Available:   2048,
					Utilisation: 0,
				},
			},
		},
		UltraRamSummary: PartDetail{
			Description: "URAM",
			Used:        0,
			Available:   470,
			Utilisation: 0,
		},
		DspBlockSummary: PartDetail{
			Description: "DSPs",
			Used:        0,
			Available:   3474,
			Utilisation: 0,
		},
		WeightedAverage: PartDetail{
			Description: "Weighted Average",
			Used:        318,
			Available:   4569222,
			Utilisation: 0.01,
		},
	}

	//return from get with status should match the build we made at the start
	if !reflect.DeepEqual(expectedReport, report) {
		t.Fatalf("\nExpected: %+v\nGot:      %+v\n", expectedReport, report)
		return
	}
}
