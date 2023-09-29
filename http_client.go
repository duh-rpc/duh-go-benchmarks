package benchmark

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/duh-rpc/duh-go"
	"github.com/duh-rpc/duh-go-benchmarks/v1"
	"google.golang.org/protobuf/proto"
)

var bufferPool = sync.Pool{
	New: func() interface{} { return new(bytes.Buffer) },
}

type HTTPClient struct {
	*duh.Client
	endpoint string
}

func NewClient(client *http.Client, address string) *HTTPClient {
	return &HTTPClient{
		endpoint: fmt.Sprintf("http://%s", address),
		Client: &duh.Client{
			Client: client,
		},
	}
}

func (c *HTTPClient) GetFeature(ctx context.Context, req *v1.Point, resp *v1.Feature) error {
	payload, err := proto.Marshal(req)
	if err != nil {
		return duh.NewClientError(fmt.Errorf("while marshaling request payload: %w", err), nil)
	}

	r, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/%s", c.endpoint, "v1/route.getFeature"), bytes.NewReader(payload))
	if err != nil {
		return duh.NewClientError(err, nil)
	}

	r.Header.Set("Content-Type", duh.ContentTypeProtoBuf)
	return c.Do(r, resp)
}
