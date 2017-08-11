// +build integration

package models

import (
	"reflect"
	"testing"

	"github.com/jinzhu/gorm"
)

func TestGetBuildsWithStatus(t *testing.T) {
	RunTransaction(func(db *gorm.DB) {
		d := BuildDataSource(db)
		//create a build in the DB
		build := Build{
			BatchJob: BatchJob{
				Events: []BatchJobEvent{
					BatchJobEvent{
						Status: "COMPLETED",
					},
				},
			},
		}
		db.Create(&build)
		//run the get with status function
		builds, err := d.GetBuildsWithStatus([]string{"COMPLETED"}, 10)
		if err != nil {
			t.Error(err)
			return
		}

		ids := []string{}
		for _, returnedBuild := range builds {
			ids = append(ids, returnedBuild.ID)
		}
		//return from get with status should match the build we made at the start
		expected := []string{build.ID}
		if !reflect.DeepEqual(expected, ids) {
			t.Fatalf("\nExpected: %+v\nGot:      %+v\n", expected, builds)
			return
		}
	})
}

func TestCreateBuildReport(t *testing.T) {
	RunTransaction(func(db *gorm.DB) {
		d := BuildDataSource(db)
		//create a build in the DB
		build := Build{}
		db.Create(&build)

		reportContents := "{\"moduleName\":\"fooModule\",\"partName\":\"barPart\",\"lutSummary\":{\"description\":\"CLB LUTs\",\"used\":70,\"available\":600577,\"utilisation\":0.01,\"detail\":{\"lutLogic\":{\"description\":\"LUT as Logic\",\"used\":3,\"available\":600577,\"utilisation\":0.01},\"lutMemory\":{\"description\":\"LUT as Memory\",\"used\":67,\"available\":394560,\"utilisation\":0.02}}},\"regSummary\":{\"description\":\"CLB Registers\",\"used\":38,\"available\":1201154,\"utilisation\":0.01,\"detail\":{\"regFlipFlop\":{\"description\":\"Register as Flip Flop\",\"used\":38,\"available\":1201154,\"utilisation\":0.01},\"regLatch\":{\"description\":\"Register as Latch\",\"used\":0,\"available\":1201154,\"utilisation\":0}}},\"blockRamSummary\":{\"description\":\"Block RAM Tile\",\"used\":0,\"available\":1024,\"utilisation\":0,\"detail\":{\"blockRamB36\":{\"description\":\"RAMB36/FIFO\",\"used\":0,\"available\":1024,\"utilisation\":0},\"blockRamB18\":{\"description\":\"RAMB18\",\"used\":0,\"available\":2048,\"utilisation\":0}}},\"ultraRamSummary\":{\"description\":\"URAM\",\"used\":0,\"available\":470,\"utilisation\":0},\"dspBlockSummary\":{\"description\":\"DSPs\",\"used\":0,\"available\":3474,\"utilisation\":0},\"weightedAverage\":{\"description\":\"Weighted Average\",\"used\":318,\"available\":4569222,\"utilisation\":0.01}}"
		//run the get with status function
		buildReport, err := d.CreateBuildReport(build, "1", reportContents)
		if err != nil {
			t.Error(err)
			return
		}

		//return from get with status should match the build we made at the start
		if buildReport.Report != (reportContents) {
			t.Fatalf("\nExpected: %+v\nGot:      %+v\n", reportContents, buildReport.Report)
			return
		}
	})
}
