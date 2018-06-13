package api

import (
	"context"

	"github.com/gin-gonic/gin"
)

func WithClose(c *gin.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(c)

	w := c.Writer

	// cancel whenever we get a close
	go func() {
		<-w.CloseNotify()
		cancel()
	}()

	return ctx, cancel
}
