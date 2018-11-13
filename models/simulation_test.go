// +build integration

package models

import (
	"testing"

	"github.com/jinzhu/gorm"
)

func TestStoreSimulationReport(t *testing.T) {
	RunTransaction(func(db *gorm.DB) {
		d := SimulationDataSource(db)

		var sim Simulation
		db.Create(&sim)

		var report Report
		err := d.StoreReport(sim.ID, report)
		if err != nil {
			t.Error(err)
			return
		}
		return
	})
}

func TestGetSimulationReport(t *testing.T) {
	RunTransaction(func(db *gorm.DB) {
		d := SimulationDataSource(db)

		report := SimulationReport{
			SimulationID: "foobar",
			Report:       "{}",
		}

		db.Create(&report)
		ret, err := d.GetReport(report.SimulationID)
		if err != nil {
			t.Error(err)
			return
		}
		if ret.ID != report.ID {
			t.Errorf("Expected: %v Got: %v", report, ret)
			return
		}
		return
	})
}

func TestSimulationByIDForUser(t *testing.T) {
	RunTransaction(func(db *gorm.DB) {
		users := []User{
			User{Email: "foo@bar.baz", GithubID: 1},
			User{Email: "baz@bar.foo", GithubID: 2},
		}
		for _, user := range users {
			db.Save(&user)
		}

		projects := []Project{
			Project{UserID: users[0].ID},
			Project{UserID: users[1].ID},
		}
		for _, project := range projects {
			db.Save(&project)
		}

		simulations := []Simulation{
			Simulation{ProjectID: projects[0].ID},
			Simulation{ProjectID: projects[0].ID},
			Simulation{ProjectID: projects[1].ID},
			Simulation{ProjectID: projects[1].ID},
		}
		for _, simulation := range simulations {
			db.Save(&simulation)
		}

		var testCombos = []struct {
			simID  string
			userID string
			ret    Simulation
		}{
			{simulations[0].ID, users[0].ID, simulations[0]},
			{simulations[0].ID, users[1].ID, Simulation{}},
			{simulations[1].ID, users[0].ID, simulations[1]},
			{simulations[1].ID, users[1].ID, Simulation{}},
			{simulations[2].ID, users[0].ID, Simulation{}},
			{simulations[2].ID, users[1].ID, simulations[2]},
			{simulations[3].ID, users[0].ID, Simulation{}},
			{simulations[3].ID, users[1].ID, simulations[3]},
		}
		d := SimulationDataSource(db)
		for _, testCombo := range testCombos {
			var sim Simulation
			sim, err := d.ByIDForUser(testCombo.simID, testCombo.userID)
			if sim.ID != testCombo.ret.ID {
				t.Errorf("Error during testcombo: %v, %v", testCombo, err)
			}
			if sim.Project.UserID != testCombo.userID {
				t.Error("Preload error during testcombo")
			}
		}
	})
}
