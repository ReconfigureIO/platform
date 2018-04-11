package admin

import (
	"github.com/ReconfigureIO/platform/models"
)

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

func (b BuildInfo) FromUser(user models.User) {
	p.ID = user.ID
	p.Name = user.Name
	p.Email = user.Email
	p.PhoneNumber = user.PhoneNumber
	p.Company = user.Company
	p.Token = user.LoginToken()
	p.BillingPlan = sub.Identifier
	p.CreatedAt = user.CreatedAt
	p.Landing = user.Landing
	p.MainGoal = user.MainGoal
	p.Employees = user.Employees
	p.MarketVerticals = user.MarketVerticals
	p.JobTitle = user.JobTitle
}
