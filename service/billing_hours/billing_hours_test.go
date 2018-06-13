package billing_hours

import (
	"testing"
	"time"

	"github.com/ReconfigureIO/platform/pkg/models"
	"github.com/ReconfigureIO/platform/service/deployment"
	"github.com/golang/mock/gomock"
)

type fake_SubscriptionRepo struct{}

type fake_Deployment struct{}

// provide a bunch of users who are active
func (repo fake_SubscriptionRepo) ActiveUsers() ([]models.User, error) {
	user := models.User{ID: "fake-user"}
	return []models.User{user}, nil
}

func (b billingHours) Net() (int, error) {
	return 0, nil
}

func TestCheckUserHours(t *testing.T) {
	d := fake_SubscriptionRepo{}

	deployments := []models.Deployment{models.Deployment{}}
	// Add 7 days to date, over 100 hours. Replace if better solution found.
	now := time.Now()
	timeInFuture := now.AddDate(0, 0, 7)

	deploymentHours := []models.DeploymentHours{models.DeploymentHours{
		Id:         "1",
		Started:    now,
		Terminated: timeInFuture,
	}}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	deploymentRepo := models.NewMockDeploymentRepo(mockCtrl)
	deploymentRepo.EXPECT().GetWithStatusForUser("fake-user", []string{"started"}).Return(deployments, nil)
	deploymentRepo.EXPECT().DeploymentHours("fake-user", gomock.Any(), gomock.Any()).Return(deploymentHours, nil)

	deploymentService := deployment.NewMockService(mockCtrl)
	deploymentService.EXPECT().StopDeployment(gomock.Any(), deployments[0]).Return(nil)

	err := CheckUserHours(d, deploymentRepo, deploymentService)
	if err != nil {
		t.Fatalf("Error in TestCheckUserHours function: %s", err)
	}
}

type billingHours struct {
}

func (b billingHours) Available() (int, error) {
	return 80, nil
}

func (b billingHours) Used() (int, error) {
	return 100, nil
}

func (s fake_SubscriptionRepo) Current(user models.User) (sub models.SubscriptionInfo, err error) {
	sub = models.SubscriptionInfo{}
	return sub, nil
}

func (s fake_SubscriptionRepo) CurrentSubscription(user models.User) (sub models.SubscriptionInfo, err error) {
	sub = models.SubscriptionInfo{}
	return sub, nil
}

func (s fake_SubscriptionRepo) UpdatePlan(user models.User, plan string) (sub models.SubscriptionInfo, err error) {
	sub = models.SubscriptionInfo{}
	return sub, nil
}
