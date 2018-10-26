// package bidtolidadapter takes an AWS Batch Job ID and returns the associated AWS CloudWatch Log Name
package bidtolidadapter

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ReconfigureIO/platform/models"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/batch"
)

type Adapter interface {
	bidToLid(string) (string, error)
}

type adapter struct {
	batchRepo models.BatchRepo
	aws       interface {
		DescribeJobs(
			*batch.DescribeJobsInput,
		) (
			*batch.DescribeJobsOutput,
			error,
		)
	}
}

func (a *adapter) bidToLid(ctx context.Context, batchID string) (string, error) {
	// take batch ID
	// try not to hammer aws.

	// // Let's ask the repo what the logname attached to this batch ID is
	// if it's a value we're done, return that value
	// if it's nothing we need to go further
	logname, err := a.batchRepo.GetLogName(batchID)
	if err != nil {
		log.Printf("bidToLid: batchRepo.GetLogName: %v \n", err)
		return "", err
	}
	if logname != "" {
		return logname, nil
	}

	// poll DB until we think batch job is started
	var started bool
	select {
    case <-ctx.Done():
        return "", ctx.Err()
    case err := <-c:
        return err
    }

	// for started != true {
	// 	started, err = a.batchRepo.HasStarted(batchID) // TODO campgareth: find a better way to implement this, maybe get the batchRepo to return a channel we can block on while it handles stuff? idk
	// 	if err != nil {
	// 		return "", err
	// 	}
	// 	time.Sleep(10 * time.Second)
	// }

	// Once batch job is started query the database one more time in case cron got there first
	logname, err = a.batchRepo.GetLogName(batchID)
	if err != nil {
		log.Printf("bidToLid: batchRepo.GetLogName: %v \n", err)
		return "", err
	}
	if logname != "" {
		return logname, nil
	}

	// If it didn't, it's time to ask AWS what's going on
	// ask AWS Batch for the JobDetail of that job
	// pull the log name out of that
	inp := &batch.DescribeJobsInput{
		Jobs: aws.StringSlice([]string{batchID}),
	}
	resp, err := a.aws.DescribeJobs(inp)
	if err != nil {
		return "", err
	}
	if len(resp.Jobs) == 0 {
		return "", fmt.Errorf("bidToLid: There is no AWS Batch Job with ID %v", batchID)
	}

	// while we're at it submit the logname back to DB, do cron's work for it
	// then return the log name
	err = a.batchRepo.SetLogName(batchID, *resp.Jobs[0].Container.LogStreamName)
	if err != nil {
		// well no biggie, we were only doing a kindness to cron.
		log.Printf("bidToLid: batchRepo.SetLogName: %v \n", err)
	}
	return *resp.Jobs[0].Container.LogStreamName, nil

}
