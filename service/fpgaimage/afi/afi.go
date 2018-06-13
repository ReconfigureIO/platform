package afi

import (
	"context"
	"sync"

	"github.com/ReconfigureIO/platform/pkg/models"
	"github.com/ReconfigureIO/platform/service/fpgaimage"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

// Service implements DescribeAFIStatus.
type Service struct {
	EC2API interface {
		DescribeFpgaImagesWithContext(aws.Context, *ec2.DescribeFpgaImagesInput, ...request.Option) (*ec2.DescribeFpgaImagesOutput, error)
	}

	once sync.Once
}

func (s *Service) ensureInit() {
	s.once.Do(func() {
		if s.EC2API == nil {
			s.EC2API = ec2.New(session.Must(session.NewSession()))
		}
	})
}

func (s *Service) DescribeAFIStatus(ctx context.Context, builds []models.Build) (map[string]fpgaimage.Status, error) {
	s.ensureInit()

	ret := make(map[string]fpgaimage.Status)

	var afiids []string
	for _, build := range builds {
		afiids = append(afiids, build.FPGAImage)
	}

	cfg := ec2.DescribeFpgaImagesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("fpga-image-global-id"),
				Values: aws.StringSlice(afiids),
			},
		},
	}

	results, err := s.EC2API.DescribeFpgaImagesWithContext(ctx, &cfg)
	if err != nil {
		return ret, err
	}

	for _, image := range results.FpgaImages {
		ret[*image.FpgaImageGlobalId] = fpgaimage.Status{
			Status:    *image.State.Code,
			UpdatedAt: *image.UpdateTime,
		}
	}

	return ret, nil
}
