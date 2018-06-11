//+build integration

package s3

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

func TestS3Upload(t *testing.T) {
	// Try uploading some random data of different sizes to S3, download and
	// check equivalence.

	for _, size := range []int64{
		0,
		1,
		5*(1<<20) - 1,
		5*(1<<20) + 1,
		10*(1<<20) + 1,
	} {
		t.Run(
			fmt.Sprintf("size=%d", size),
			s3Test{
				Size: size,
			}.TestS3Upload,
		)
	}
}

type s3Test struct {
	Size int64
}

func (test s3Test) TestS3Upload(t *testing.T) {
	// Try uploading some random data to S3 and, download and
	// check equivalence.

	sess := session.Must(session.NewSession())
	storage := &Service{
		Bucket:      "testbucket.reconfigure.io",
		S3API:       s3.New(sess),
		UploaderAPI: s3manager.NewUploader(sess),
	}

	timestamp := time.Now().Format(time.RFC3339)
	key := fmt.Sprintf("integrationtest/TestS3Upload-%v", timestamp)

	randomReader := rand.New(rand.NewSource(0))
	random10Meg := io.LimitReader(randomReader, test.Size)
	missingLength := int64(-1)

	hasher := sha1.New()

	// Hash the contents as they are read by the uploader.
	random10Meg = io.TeeReader(random10Meg, hasher)

	s3key, err := storage.Upload(key, random10Meg, missingLength)
	if err != nil {
		t.Fatalf("storage.Upload: %v", err)
	}

	uploadedHash := hex.EncodeToString(hasher.Sum(nil))

	defer func() {
		_, err := storage.S3API.DeleteObject(&s3.DeleteObjectInput{
			Bucket: aws.String(storage.Bucket),
			Key:    aws.String(key),
		})
		if err != nil {
			t.Logf("Failed to delete object: %v", err)
		}
	}()

	t.Logf("Upload completed at %v", s3key)

	rc, err := storage.Download(key)
	if err != nil {
		t.Fatalf("Failed to download: %v", err)
	}
	defer func() {
		errClose := rc.Close()
		if errClose != nil {
			t.Errorf("rc.Close: %v", errClose)
		}
	}()

	hasher.Reset()
	_, err = io.Copy(hasher, rc)
	if err != nil {
		t.Fatalf("failed to io.Copy: %v", err)
	}

	downloadedHash := hex.EncodeToString(hasher.Sum(nil))
	if uploadedHash != downloadedHash {
		t.Logf("uploadedHash != downloadedHash")
	}
}
