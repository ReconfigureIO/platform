package stream

import (
	"bytes"
	"github.com/ReconfigureIO/platform/service/aws"
	"github.com/gin-gonic/gin"
	"io"
)

// start a stream of cloudwatch log events, and stream the messages to
// the client until it finishes
func Stream(stream *aws.Stream, c *gin.Context) {
	defer stream.Stop()
	go func() {
		err := stream.Run()
		if err != nil {
			c.Error(err)
		}
	}()

	c.Stream(func(w io.Writer) bool {
		log, ok := <-stream.Events
		if ok {
			for _, e := range log.Events {
				_, err := bytes.NewBufferString((*e.Message) + "\n").WriteTo(w)
				if err != nil {
					c.Error(err)
					return false
				}
			}
		}
		return ok
	})
}
