package admin

import (
	"fmt"

	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/sugar"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

type Build struct {
	DB *gorm.DB
}

type BuildData struct {
	BuildID        string
	BuildStatus    string
	UserID         string
	UserGithubName string
	UserName       string
	UserCompany    string
	ProjectName    string
}

const (
	SQL_BUILD_DATA = `
	select j.id as buildid, users.id as user_id, users.github_name as user_github_name, users.name as user_name, users.company as user_company, projects.name as project_name, event.status as build_status
	from builds j
	left join projects
	on j.project_id = projects.id
	left join users
	on projects.user_id = users.id
	left join batch_job_events event
	on j.batch_job_id = event.batch_job_id
		and event.timestamp = (
			select max(timestamp)
			from batch_job_events e1
			where j.batch_job_id = e1.batch_job_id
		)
	`
)

// Lists BuildInfo for all builds (big reply)
func (b Build) List(c *gin.Context) {
	//Ask the database for a big join of all these objects, put into an array
	rows, err := b.DB.Raw(SQL_BUILD_DATA).Rows()

	if err != nil {
		sugar.InternalError(c, err)
		return
	}

	builds := []BuildData{}
	for rows.Next() {
		var temp1 BuildData
		rows.Scan(&temp1.BuildID, &temp1.UserID, &temp1.UserGithubName, &temp1.UserName, &temp1.UserCompany, &temp1.ProjectName, &temp1.BuildStatus)
		fmt.Printf("resulting object: %v", temp1)
		fmt.Println("")
		fmt.Println(temp1.BuildID)
		fmt.Println(temp1.BuildStatus)
		builds = append(builds, temp1)
	}
	rows.Close()
	output := []BuildInfo{}
	for _, build := range builds {
		var user models.User
		b.DB.Where("id = ?", build.UserID).First(&user)
		sub, err := models.SubscriptionDataSource(b.DB).CurrentSubscription(user)
		if err != nil {
			fmt.Println("error :(")
			sugar.InternalError(c, err)
			return
		}
		var temp BuildInfo
		temp.FromBuildData(build, sub)
		fmt.Printf("resulting object: %v", temp)
		output = append(output, temp)
	}

	sugar.SuccessResponse(c, 200, output)
}

type BuildInfo struct {
	BuildID        string `json:"id"`
	BuildStatus    string `json:"status"`
	UserID         string `json:"user_id"`
	UserGithubName string `json:"github_name"`
	UserName       string `json:"name"`
	UserCompany    string `json:"company"`
	ProjectName    string `json:"project"`
	BuildInput     string `json:"input_artifact"`
}

func (b *BuildInfo) FromBuildData(bd BuildData, sub models.SubscriptionInfo) {
	fmt.Println("bd.BuildID = " + bd.BuildID)
	b.BuildID = bd.BuildID
	b.BuildStatus = bd.BuildStatus
	b.UserID = bd.UserID
	b.UserGithubName = bd.UserGithubName
	b.UserName = bd.UserName
	b.UserCompany = bd.UserCompany
	b.ProjectName = bd.ProjectName
	//is this user a paying customer?
	if sub.Identifier == models.PlanOpenSource {
		b.BuildInput = "s3://reconfigureio-builds/builds/" + b.BuildID + "/build.tar.gz"
	}
}
