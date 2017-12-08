package credits

import (
	"testing"
	"time"

	"github.com/ReconfigureIO/platform/models"
	"github.com/golang/mock/gomock"
	"github.com/jinzhu/gorm"
)

func TestUpdateDebits(t *testing.T) {
	models.RunTransaction(func(db *gorm.DB) {
		user := models.User{
			ID:         "josh",
			Email:      "josh@joshbohde.com",
			GithubName: "joshbohde",
			// important part
			StripeToken: "cus_AgZQTeZbnY6AE4",
		}

		timeNow := time.Now()

		subscriptionInfo := models.SubscriptionInfo{
			UserID:    user.ID,
			StartTime: timeNow.AddDate(0, -1, -1),
			EndTime:   timeNow.AddDate(0, 0, -1),
			Hours:     1,
		}

		deploymentHours := models.DeploymentHours{
			Started:    timeNow.AddDate(0, 0, -5),
			Terminated: timeNow.AddDate(0, 0, -4),
		}

		joshCredit := models.Credit{
			Hours: 1000,
			User:  user,
		}

		joshDebit := models.Debit{
			Hours:     0,
			User:      user,
			InvoiceID: "foobar",
		}

		joshBalance := models.UserBalance{
			Subscription: subscriptionInfo,
			Credits:      []models.Credit{joshCredit},
			Debits:       []models.Debit{joshDebit},
		}

		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		ds := models.NewMockUserBalanceRepo(mockCtrl)
		deployments := models.NewMockDeploymentRepo(mockCtrl)

		ds.EXPECT().ActiveUsers().Return([]models.User{user}, nil)
		ds.EXPECT().CurrentSubscription(user).Return(subscriptionInfo, nil)
		ds.EXPECT().AddDebit(user, 23, "foobar").Return(nil)
		//get list of deployments along with their start and end times
		deployments.EXPECT().DeploymentHours(subscriptionInfo.UserID, subscriptionInfo.StartTime, subscriptionInfo.EndTime).Return([]models.DeploymentHours{deploymentHours}, nil)
		ds.EXPECT().GetUserBalance(user).Return(joshBalance, nil)

		err := UpdateDebits(ds, deployments, timeNow)
		if err != nil {
			t.Error(err)
			return
		}
	})
}
