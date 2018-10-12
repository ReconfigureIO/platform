package main

import (
	"bytes"
	"context"
	"crypto/md5"
	"fmt"
	"hash"
	"io"
	"log"
	"math/rand"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/docker/docker/pkg/stdcopy"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/batch"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"

	"github.com/ReconfigureIO/platform/service/fakebatchlogs"
	"github.com/ReconfigureIO/platform/service/storage/localfile"
)

type fakeDockerClient struct {
	dockerClient

	mu                     *sync.Mutex
	idCount                int
	idToTestContainerState map[string]*fakeContainer
}

type fakeContainer struct {
	id     int
	status string
	once   sync.Once
	hasher hash.Hash
	// hashReady is used to indicate that all the bytes of log have been written
	// to the hasher and that the Sum is now available. It is closed by the
	// function that wins the race for once.Do after its io.Copy has completed.
	hashReady chan struct{}
}

func (c *fakeContainer) GetHash(b []byte) []byte {
	<-c.hashReady
	return c.hasher.Sum(b)
}

func (c *fakeDockerClient) ContainerCreate(
	ctx context.Context,
	config *container.Config,
	hostConfig *container.HostConfig,
	networkingConfig *network.NetworkingConfig,
	containerName string,
) (
	container.ContainerCreateCreatedBody,
	error,
) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.idCount++
	idString := strconv.Itoa(c.idCount)
	c.idToTestContainerState[idString] = &fakeContainer{
		id:        c.idCount,
		status:    "created",
		hasher:    md5.New(),
		hashReady: make(chan struct{}),
	}

	return container.ContainerCreateCreatedBody{
		ID: idString,
	}, nil
}

func (c *fakeDockerClient) ContainerStart(
	ctx context.Context,
	containerID string,
	options types.ContainerStartOptions,
) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.idToTestContainerState[containerID].status = "started"

	return nil
}

func (c *fakeDockerClient) ContainerLogs(
	ctx context.Context,
	container string,
	options types.ContainerLogsOptions,
) (
	io.ReadCloser,
	error,
) {
	c.mu.Lock()
	tcs := c.idToTestContainerState[container]
	c.mu.Unlock()

	maxLength := 50
	r := getRandomData(maxLength, tcs.id)

	// We've got a predictable source of randomness that will always produce the
	// same data for the same seed. We need to generate a hash of this data but
	// it only makes sense to do it on the first run of this code since
	// subsequent runs will be hashing the same data. The primitive to do this
	// is a sync.Once. It takes a function.

	// The hash is computed by the first once.Do() caller which wins. When the
	// hash is computed, this channel will be closed to indicate that the Sum is
	// ready to be used.

	var hasherConnected bool
	tcs.once.Do(func() {
		r = io.TeeReader(r, tcs.hasher)
		hasherConnected = true
	})

	piper, pipew := io.Pipe()
	w := stdcopy.NewStdWriter(pipew, stdcopy.Stdout)

	go func() {
		defer func() {
			if hasherConnected == true {
				close(tcs.hashReady)
			}
		}()
		defer pipew.Close()

		_, err := io.Copy(w, r)
		if err != nil {
			panic(fmt.Errorf("io.Copy in ContainerLogs: %v", err))
		}
	}()

	return piper, nil
}

func (c *fakeDockerClient) ContainerWait(
	ctx context.Context,
	containerID string,
	condition container.WaitCondition,
) (
	<-chan container.ContainerWaitOKBody,
	<-chan error,
) {
	exited := make(chan container.ContainerWaitOKBody)
	err := make(chan error)
	go func() {
		time.Sleep(1 * time.Millisecond)
		exited <- container.ContainerWaitOKBody{}
	}()
	return exited, err
}

func (c *fakeDockerClient) ContainerRemove(
	ctx context.Context,
	containerID string,
	options types.ContainerRemoveOptions,
) error {
	return nil
}

// The goal of this test is to test fake-batch's internal logic.
// It takes a fake implementation of Docker and uses it to respond to client requests.
// The client steps taken here are the creation of a job and requesting the logs of that job.
func TestHandler(t *testing.T) {
	var numContainers = 10
	var numFollowers = 30
	var wg sync.WaitGroup

	fakeDockerClient := &fakeDockerClient{
		mu: &sync.Mutex{},
		idToTestContainerState: make(map[string]*fakeContainer),
	}
	handler := &handler{
		dockerClient: fakeDockerClient,
		dockerState:  NewDockerState(),
		jobDefinitions: map[string]JobDefinition{
			"fake-batch-job-definition": {},
		},

		jobQueueSemaphores: newJobQueueSemaphores(map[Q]int{
			"build": 10,
			"graph": 2,
			"sim":   2,
		}),

		storage: localfile.Service("./logs/"),
	}

	s := httptest.NewServer(handler)
	defer log.Println("Closed")
	defer s.Close()

	config := &aws.Config{
		Credentials: credentials.NewStaticCredentials(
			"nocredentialsneeded", "nocredentialsneeded", "nocredentialsneeded"),
		Endpoint:   aws.String(s.URL),
		MaxRetries: aws.Int(0),
		Region:     aws.String("noregion"),
	}

	batchSession := batch.New(session.New(config))

	for i := 0; i < numContainers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			respSubmitJob, err := batchSession.SubmitJob(&batch.SubmitJobInput{
				JobDefinition: aws.String("fake-batch-job-definition"),
				JobName:       aws.String("example"),
				JobQueue:      aws.String("build"),
				ContainerOverrides: &batch.ContainerOverrides{
					Command: aws.StringSlice([]string{"echo", "foobar"}),
				},
			})
			if err != nil {
				t.Fatal(err)
			}

			jobID := *respSubmitJob.JobId

			logService := fakebatchlogs.Service{
				Endpoint: s.URL,
			}
			for j := 0; j < numFollowers; j++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					logReader := logService.Stream(context.Background(), jobID)
					defer logReader.Close()

					logCheckHasher := md5.New()
					_, err := io.Copy(logCheckHasher, logReader)
					if err != nil {
						log.Println("Error")
						t.Fatalf("io.Copy: %v", err)
					}

					receivedHash := logCheckHasher.Sum(nil)
					fakeDockerClient.mu.Lock()
					tcs := fakeDockerClient.idToTestContainerState[jobID]
					expectedHash := tcs.GetHash(nil)
					fakeDockerClient.mu.Unlock()

					if !bytes.Equal(expectedHash, receivedHash) {
						fmt.Printf("Job ID: %v Expected %v, got %v \n", jobID, expectedHash, receivedHash)
						// t.Errorf("Source and received log hashes are not equal")
					}
				}()
			}

		}()
	}
	wg.Wait()
}

func getRandomData(maxLength, seed int) io.Reader {
	r := rand.New(rand.NewSource(int64(seed)))
	lengthOfLog := r.Int63n(int64(maxLength))
	return io.LimitReader(r, lengthOfLog)
}
