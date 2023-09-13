package benchmark

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/duh-rpc/duh-grpc-benchmarks/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/runtime/protoiface"
)

var bufferPool = sync.Pool{
	New: func() interface{} { return new(bytes.Buffer) },
}

type HTTPClient struct {
	baseURL string
	client  *http.Client
}

func NewClient(client *http.Client, address string) *HTTPClient {
	return &HTTPClient{client: client, baseURL: fmt.Sprintf("http://%s", address)}
}

func (c *HTTPClient) GetFeature(ctx context.Context, in *v1.Point) (*v1.Feature, error) {
	body := bufferPool.Get().(*bytes.Buffer)
	body.Reset()
	defer bufferPool.Put(body)

	b, err := protoMarshal(body, in)
	if err != nil {
		return nil, fmt.Errorf("while marshaling request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/%s", c.baseURL, "v1/route.getFeature"), bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	var resp v1.Feature
	if err := c.Do(req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *HTTPClient) Do(req *http.Request, out proto.Message) error {
	// To make it a fair comparison, we could use RoundTrip() which bypasses auth handling,
	// deprecated timeout handling, cookies, and redirect functionality which
	// occurs in http.Client.Do(). Even with that, HTTP is still faster than GRPC.

	//rs, err := c.client.Transport.RoundTrip(req)
	rs, err := c.client.Do(req)
	if err != nil {
		return wrapRequestError(err, req)
	}
	defer func() { _ = rs.Body.Close() }()

	if rs.StatusCode != http.StatusOK {
		return errFromResponse(rs)
	}

	body := bufferPool.Get().(*bytes.Buffer)
	body.Reset()
	defer bufferPool.Put(body)

	if _, err = io.Copy(body, rs.Body); err != nil {
		return fmt.Errorf("while reading response body: %v", err)
	}

	if err := proto.Unmarshal(body.Bytes(), out); err != nil {
		return fmt.Errorf("while parsing response body '%s': %w", body, err)
	}
	return nil
}

func errFromResponse(rs *http.Response) error {
	body, err := io.ReadAll(io.LimitReader(rs.Body, 1024*1024))
	if err != nil {
		return fmt.Errorf("HTTP request error, status=%s", rs.Status)
	}
	return fmt.Errorf("HTTP request error, status=%s, message=%s",
		rs.Status, strings.Trim(string(body), "\n"))
}

func wrapRequestError(err error, rq *http.Request) error {
	var headers []string
	for key, values := range rq.Header {
		headers = append(headers, fmt.Sprintf("(%s: %s)", key, values))
	}
	return fmt.Errorf("%s %s with [%s]: %w",
		rq.Method, rq.URL.String(), strings.Join(headers, ","), err)
}

func protoMarshal(body *bytes.Buffer, in proto.Message) ([]byte, error) {
	out, err := proto.MarshalOptions{}.MarshalState(protoiface.MarshalInput{
		Message: in.ProtoReflect(),
		Buf:     body.Bytes(),
	})
	return out.Buf, err
}
