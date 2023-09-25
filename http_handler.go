package benchmark

import (
	"net/http"

	"github.com/duh-rpc/duh-go"
	"github.com/duh-rpc/duh-go-benchmarks/server"
	v1 "github.com/duh-rpc/duh-go-benchmarks/v1"
)

func NewHTTPHandler(service *server.RouteGuideService) *Handler {
	return &Handler{service: service}
}

type Handler struct {
	service *server.RouteGuideService
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	// Use a simple switch for static routes just like GRPC does
	case "/v1/route.getFeature":
		h.handleGetFeature(w, r)
		return
	case "/v1/say.hello":
		w.Header().Set("Content-Type", duh.ContentOctetStream)
		_, _ = w.Write([]byte("Hello!"))
		return
	}
	duh.ReplyWithCode(w, r, duh.CodeNotImplemented, nil, "no such method; "+r.URL.Path)
}

func (h *Handler) handleGetFeature(w http.ResponseWriter, r *http.Request) {
	var req v1.Point
	if err := duh.ReadRequest(r, &req); err != nil {
		duh.ReplyError(w, r, err)
		return
	}
	resp, err := h.service.GetFeature(r.Context(), &req)
	if err != nil {
		duh.ReplyError(w, r, err)
		return
	}
	duh.Reply(w, r, duh.CodeOK, resp)
}
