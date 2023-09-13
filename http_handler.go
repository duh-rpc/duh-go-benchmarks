package benchmark

import (
	"bytes"
	"io"
	"log"
	"net/http"

	"github.com/duh-rpc/duh-grpc-benchmarks/server"
	v1 "github.com/duh-rpc/duh-grpc-benchmarks/v1"
	"github.com/golang/protobuf/proto"
)

func NewHTTPHandler(service *server.RouteGuideService) *HTTPHandler {
	return &HTTPHandler{service: service}
}

type HTTPHandler struct {
	service *server.RouteGuideService
}

// TODO: Improve this with DUH
func (h *HTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	// Use a simple switch for static routes just like GRPC does
	case "/v1/route.getFeature":
		b := bufferPool.Get().(*bytes.Buffer)
		defer bufferPool.Put(b)
		b.Reset()

		_, err := io.Copy(b, r.Body)
		if err != nil {
			log.Fatal(err)
		}
		var req v1.Point

		if err := proto.Unmarshal(b.Bytes(), &req); err != nil {
			log.Fatal(err)
		}

		resp, err := h.service.GetFeature(r.Context(), &req)
		if err != nil {
			log.Fatal(err)
		}

		payload, err := protoMarshal(b, resp)
		w.Header().Set("Content-Type", "application/protobuf")
		_, _ = w.Write(payload)
		return
	case "/v1/say.hello":
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("Hello!"))
		return
	}
	w.WriteHeader(http.StatusMethodNotAllowed)
	// TODO: Add the other methods using `duh`
}
