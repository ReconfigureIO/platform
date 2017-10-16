// +build integration

package queue

import (
	"context"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/deployment"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/jinzhu/gorm"
)

func startDeploymentQueue() (Queue, []Job) {
	db := connectDB()
	jobs, err := genDeploymentsJobs(db, 5)
	if err != nil {
		log.Fatal(err)
	}

	runner := DeploymentRunner{
		Hostname:     "test.reconfigure.io",
		DB:           db,
		Service:      &fakeDepService{db: db},
		waitInterval: time.Second * 1,
	}
	deploymentQueue := &dbQueue{
		jobType:      "deployment",
		runner:       runner,
		concurrent:   2,
		service:      QueueService{db: db},
		waitInterval: time.Second * 1,
	}

	go deploymentQueue.Start()
	return deploymentQueue, jobs
}

func genDeploymentsJobs(db *gorm.DB, n int) ([]Job, error) {
	var jobs []Job
	for i := 0; i < n; i++ {
		dep := models.Deployment{
			Build: models.Build{
				Project: models.Project{
					UserID: "user1",
				},
			},
			Command: "test",
		}
		if err := db.Create(&dep).Error; err != nil {
			return jobs, err
		}
		jobs = append(jobs, Job{ID: dep.ID, Weight: 2})
	}
	return jobs, nil
}

func TestDeploymentRunner(t *testing.T) {
	queue, jobs := startDeploymentQueue()
	for _, job := range jobs {
		queue.Push(job)
	}
	service := queue.(*dbQueue).runner.(DeploymentRunner).Service.(*fakeDepService)
	for {
		time.Sleep(time.Second * 1)

		service.Lock()
		count := service.count
		service.Unlock()

		if count >= len(jobs) {
			queue.Halt()
			break
		}
	}
}

var _ deployment.Service = &fakeDepService{}

type fakeDepService struct {
	db    *gorm.DB
	count int
	sync.Mutex
}

func (f *fakeDepService) RunDeployment(_ context.Context, dep models.Deployment, callbackUrl string) (string, error) {
	time.Sleep(time.Second * 1)
	f.Lock()
	f.count++
	f.Unlock()

	log.Println("starting deployment", dep.ID)

	go func() {
		// ensure deployment is queued before completing it
		for {
			time.Sleep(time.Second * 1)

			var dep1 models.Deployment
			err := f.db.Preload("Events", func(db *gorm.DB) *gorm.DB {
				return db.Order("timestamp")
			}).First(&dep1, "id = ?", dep.ID).Error

			if err != nil {
				log.Fatal(err)
			}

			if len(dep1.Events) == 1 && dep1.Events[0].Status == models.StatusQueued {
				break
			}
		}

		finishEvent := models.DeploymentEvent{Timestamp: time.Now(), Status: models.StatusCompleted}
		err := f.db.Model(&dep).Association("Events").Append(finishEvent).Error
		if err != nil {
			log.Fatal(err)
		}
		log.Println("finished deployment", dep.ID)
	}()

	return "fakeDeployment" + dep.ID, nil
}
func (f *fakeDepService) StopDeployment(ctx context.Context, deployment models.Deployment) error {
	return nil
}
func (f *fakeDepService) GetDepDetail(id int) (detail string, err error) { return }
func (f *fakeDepService) GetDeploymentStream(ctx context.Context, deployment models.Deployment) (*cloudwatchlogs.LogStream, error) {
	return nil, nil
}
func (f *fakeDepService) DescribeInstanceStatus(ctx context.Context, deployments []models.Deployment) (map[string]string, error) {
	return nil, nil
}
func (f *fakeDepService) GetServiceConfig() deployment.ServiceConfig {
	return deployment.ServiceConfig{}
}
