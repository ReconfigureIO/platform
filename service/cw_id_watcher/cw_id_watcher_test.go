package cw_id_watcher

import (
	"context"
	"testing"

	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/aws"
	"github.com/golang/mock/gomock"
)

func TestFindLogNames(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	b := models.NewMockBatchRepo(mockCtrl)
	a := aws.NewMockService(mockCtrl)

	watcher := NewLogWatcher(a, b)

	batchJobIDs := []string{"foobar"}
	cwLogNames := map[string]string{"foobar": "cwLogName"}

	ctx := context.Background()
	limit := 100

	a.EXPECT().ListBatchJobs(ctx, limit).Return(batchJobIDs, nil)
	a.EXPECT().GetCwLogNames(ctx, batchJobIDs).Return(cwLogNames, nil)
	b.EXPECT().SetCwLogName("foobar", "cwLogName").Return(nil)

	err := watcher.FindLogNames(ctx, limit)

	if err != nil {
		t.Error(err)
	}
}
