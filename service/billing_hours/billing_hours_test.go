package billing_hours

import (
	"testing"

	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/mock_deployment"
	"github.com/golang/mock/gomock"
)

type fake_SubscriptionRepo struct{}

type fake_Deployment struct{}

// provide a bunch of users who are active
func (repo fake_SubscriptionRepo) ActiveUsers() ([]models.User, error) {
	user := models.User{}
	return []models.User{user}, nil
}

func (b billingHours) Net() (int, error) {
	return 0, nil
}

func TestCheckUserHours(t *testing.T) {
	d := fake_SubscriptionRepo{}

	deployments := []models.Deployment{models.Deployment{}}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockDeploymentRepo := models.NewMockDeploymentRepo(mockCtrl)
	mockDeploymentRepo.EXPECT().HoursUsedSince(gomock.Any(), gomock.Any()).Return(5, nil)
	mockDeploymentRepo.EXPECT().ListUserDeployments(gomock.Any(), gomock.Any()).Return(deployments, nil)

	mockDeployments := mock_deployment.NewMockService(mockCtrl)
	mockDeployments.EXPECT().StopDeployment(gomock.Any(), deployments[0]).Return(nil)

	err := CheckUserHours(d, mockDeploymentRepo, mockDeployments)
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
