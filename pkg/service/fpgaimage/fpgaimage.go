package fpgaimage

import (
	"context"
	"time"

	"github.com/ReconfigureIO/platform/pkg/models"
)

// Status contains information about an FPGA image.
type Status struct {
	Status    string
	UpdatedAt time.Time
}

//go:generate mockgen -source=fpgaimage.go -package=fpgaimage -destination=fpgaimage_mock.go

// The Service interface provides DescribeAFIStatus().
type Service interface {
	DescribeAFIStatus(ctx context.Context, builds []models.Build) (map[string]Status, error)
}
