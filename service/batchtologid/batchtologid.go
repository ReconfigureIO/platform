// package batchtologid takes an AWS Batch Job ID and returns the associated AWS CloudWatch Log Name
package batchtologid

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/ReconfigureIO/platform/models"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/batch"
)

type Adapter struct {
	BatchRepo models.BatchRepo
	AWS       interface {
		DescribeJobs(
			*batch.DescribeJobsInput,
		) (
			*batch.DescribeJobsOutput,
			error,
		)
	}
	PollingPeriod time.Duration
}

// Do takes a batch job ID and returns the log name associated with that job. It
// attempts to do this by querying batchRepo. It first waits for the batch job
// to become started, which is a blocking operation. It then queries the batch
// repo for the log name. If this is not available, it queries AWS for the log
// name.
func (a *Adapter) Do(ctx context.Context, batchID string) (string, error) {
	err := models.BatchAwaitStarted(ctx, a.BatchRepo, batchID, a.PollingPeriod)
	if err != nil {
		return "", err
	}

	logname, err := a.BatchRepo.GetLogName(batchID)
	if err != nil {
		log.Printf("bidToLid: BatchRepo.GetLogName: %v \n", err)
		return "", err
	}
	if logname != "" {
		return logname, nil
	}

	resp, err := a.AWS.DescribeJobs(&batch.DescribeJobsInput{
		Jobs: aws.StringSlice([]string{batchID}),
	})
	if err != nil {
		return "", err
	}
	if len(resp.Jobs) == 0 {
		return "", fmt.Errorf("bidToLid: There is no AWS Batch Job with ID %v", batchID)
	}

	if resp.Jobs[0].Container.LogStreamName == nil {
		return "", errors.New("BatchToLogID: Adapter.Do: Got nil LogStreamName from AWS Batch")
	}
	err = a.BatchRepo.SetLogName(batchID, *resp.Jobs[0].Container.LogStreamName)
	if err != nil {
		log.Printf("bidToLid: BatchRepo.SetLogName: %v \n", err)
	}
	return *resp.Jobs[0].Container.LogStreamName, nil

}
