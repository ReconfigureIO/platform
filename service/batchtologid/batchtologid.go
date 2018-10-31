// package batchtologid takes an AWS Batch Job ID and returns the associated AWS CloudWatch Log Name
package batchtologid

import (
	"fmt"
	"log"

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

// bidToLid takes a batch job ID and returns the log name associated with that
// job. It attempts to do this by querying our Database. If the log name is not
// available yet, perhaps because we're using AWS Batch which only presents a
// log name once the job has started running, then we wait for the DB to state
// the job has started when queried before asking AWS Batch for the log name. If
// we get the log name from AWS Batch as part of this process we also write it
// back to the batch job so that cron doesn't have to continue to poll AWS Batch
// for that particular job.
func (a *adapter) bidToLid(batchID string) (string, error) {
	started, err := a.batchRepo.AwaitStarted(batchID)
	if err != nil {
		return "", err
	}
	<-started

	logname, err := a.batchRepo.GetLogName(batchID)
	if err != nil {
		log.Printf("bidToLid: batchRepo.GetLogName: %v \n", err)
		return "", err
	}
	if logname != "" {
		return logname, nil
	}

	resp, err := a.aws.DescribeJobs(&batch.DescribeJobsInput{
		Jobs: aws.StringSlice([]string{batchID}),
	})
	if err != nil {
		return "", err
	}
	if len(resp.Jobs) == 0 {
		return "", fmt.Errorf("bidToLid: There is no AWS Batch Job with ID %v", batchID)
	}

	err = a.batchRepo.SetLogName(batchID, *resp.Jobs[0].Container.LogStreamName)
	if err != nil {
		log.Printf("bidToLid: batchRepo.SetLogName: %v \n", err)
	}
	return *resp.Jobs[0].Container.LogStreamName, nil

}
