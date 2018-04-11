package admin

import (
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
	select j.id as build_id, event.status as build_status, users.id as user_id, users.github_name as user_github_name, users.name as user_name, users.company as user_company, projects.name as project_name 
	from builds j
	left join batch_job_events event
	on j.batch_job_id = event.batch_job_id
		and event.timestamp = (
			select max(timestamp)
			from batch_job_events e1
			where j.batch_job_id = e1.batch_job_id
		)
	left join projects
	on j.project_id = projects.id
	left join users
	on projects.user_id = users.id
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
		var build BuildData
		rows.Scan(&build)
		builds = append(builds, build)
	}
	rows.Close()

	output := [len(builds)]BuildInfo{}
	for i, build := range builds {
		output[i].FromBuildData(build)
	}

	sugar.SuccessResponse(c, 200, output)
}
