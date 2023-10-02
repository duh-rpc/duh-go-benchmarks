package benchmark_test

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	benchmark "github.com/duh-rpc/duh-go-benchmarks"
	"github.com/duh-rpc/duh-go-benchmarks/server"
	pb "github.com/duh-rpc/duh-go-benchmarks/v1"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func BenchmarkGRPC(b *testing.B) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()

	const GRPCAddress = "localhost:9081"
	listener, err := net.Listen("tcp", GRPCAddress)
	if err != nil {
		b.Fatalf("failed to listen: %v", err)
	}
	defer func() { _ = listener.Close() }()

	grpcServer := grpc.NewServer()
	go func() {
		pb.RegisterRouteGuideServer(grpcServer, server.NewRouteGuideServer())
		if err := grpcServer.Serve(listener); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				log.Fatal(err)
			}
		}
	}()

	// Wait for the server in the go routine to start
	if err := WaitForConnect(ctx, GRPCAddress); err != nil {
		b.Fatal(err)
	}

	// To make it a fair comparison, and avoid any slow down when establishing a
	// connection, We use `WithBlock()` to wait for a connection here before moving on with the test.
	conn, err := grpc.Dial(GRPCAddress, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		b.Fatalf("fail to dial: %v", err)
	}
	defer func() { _ = conn.Close() }()
	client := pb.NewRouteGuideClient(conn)

	b.Run("grpc.GetFeature()", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			_, err := client.GetFeature(ctx, &pb.Point{Latitude: 409146138, Longitude: -746188906})
			if err != nil {
				b.Fatalf("client.GetFeature failed: %v", err)
			}
		}
	})
	b.ReportAllocs()
	grpcServer.GracefulStop()
}

func BenchmarkHTTP2(b *testing.B) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()

	const HTTPAddress = "localhost:9080"
	listener, err := net.Listen("tcp", HTTPAddress)
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = listener.Close() }()

	handler := benchmark.NewHTTPHandler(server.NewRouteGuideServer())

	// Support H2C (HTTP/2 ClearText)
	// See https://github.com/thrawn01/h2c-golang-example
	h2s := &http2.Server{}

	srv := &http.Server{
		Addr:    HTTPAddress,
		Handler: h2c.NewHandler(handler, h2s),
	}

	go func() {
		if err := srv.Serve(listener); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				panic(err)
			}
		}
	}()

	// Wait for the server in the go routine to start
	if err := WaitForConnect(ctx, HTTPAddress); err != nil {
		b.Fatal(err)
	}

	// For a fair comparison, we establish a connection the HTTP server before the benchmark test begins by making a
	// GET call to the server. See WithBlock() on grpc.Dial() in the GRPC benchmark test above
	hc := &http.Client{
		Transport: &http2.Transport{
			// So http2.Transport doesn't complain the URL scheme isn't 'https'
			AllowHTTP: true,
			// Pretend we are dialing a TLS endpoint. (Note, we ignore the passed tls.Config)
			DialTLSContext: func(ctx context.Context, network, addr string, cfg *tls.Config) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, network, addr)
			},
		},
	}
	r, err := hc.Get("http://" + HTTPAddress + "/v1/say.hello")
	if err != nil {
		b.Fatal(err)
		return
	}
	fmt.Printf("Proto: %d\n", r.ProtoMajor)
	_ = r.Body.Close()

	client := benchmark.NewClient(hc, fmt.Sprintf("http://%s", HTTPAddress))

	b.Run("http.GetFeature()", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			var resp pb.Feature
			err := client.GetFeature(ctx, &pb.Point{Latitude: 409146138, Longitude: -746188906}, &resp)
			if err != nil {
				b.Fatalf("client.GetFeature failed: %v", err)
			}
		}
	})
	b.ReportAllocs()
	_ = srv.Shutdown(context.Background())
}

func BenchmarkHTTP1(b *testing.B) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()

	const HTTPAddress = "localhost:9081"
	listener, err := net.Listen("tcp", HTTPAddress)
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = listener.Close() }()

	handler := benchmark.NewHTTPHandler(server.NewRouteGuideServer())

	srv := &http.Server{
		Addr:    HTTPAddress,
		Handler: handler,
	}

	go func() {
		if err := srv.Serve(listener); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				panic(err)
			}
		}
	}()

	// Wait for the server in the go routine to start
	if err := WaitForConnect(ctx, HTTPAddress); err != nil {
		b.Fatal(err)
	}

	// For a fair comparison, we establish a connection the HTTP server before the benchmark test begins by making a
	// GET call to the server. See WithBlock() on grpc.Dial() in the GRPC benchmark test above
	hc := &http.Client{Transport: http.DefaultTransport}
	r, err := hc.Get("http://" + HTTPAddress + "/v1/say.hello")
	if err != nil {
		b.Fatal(err)
		return
	}
	fmt.Printf("Proto: %d\n", r.ProtoMajor)
	_ = r.Body.Close()

	client := benchmark.NewClient(hc, fmt.Sprintf("http://%s", HTTPAddress))

	b.Run("http.GetFeature()", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			var resp pb.Feature
			err := client.GetFeature(ctx, &pb.Point{Latitude: 409146138, Longitude: -746188906}, &resp)
			if err != nil {
				b.Fatalf("client.GetFeature failed: %v", err)
			}
		}
	})
	b.ReportAllocs()
	_ = srv.Shutdown(context.Background())
}

func BenchmarkHTTPS(b *testing.B) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()

	var conf benchmark.TLSConfig
	if err := benchmark.SetupTLS(&conf); err != nil {
		b.Fatal(err)
	}

	const HTTPAddress = "localhost:9082"
	listener, err := net.Listen("tcp", HTTPAddress)
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = listener.Close() }()

	handler := benchmark.NewHTTPHandler(server.NewRouteGuideServer())

	srv := &http.Server{
		TLSConfig: conf.ServerTLS,
		Addr:      HTTPAddress,
		Handler:   handler,
	}

	go func() {
		if err := srv.ServeTLS(listener, "", ""); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				panic(err)
			}
		}
	}()

	// Wait for the server in the go routine to start
	if err := WaitForConnect(ctx, HTTPAddress); err != nil {
		b.Fatal(err)
	}

	// For a fair comparison, we establish a connection the HTTP server before the benchmark test begins by making a
	// GET call to the server. See WithBlock() on grpc.Dial() in the GRPC benchmark test above
	hc := &http.Client{
		Transport: &http2.Transport{
			TLSClientConfig: conf.ClientTLS,
		},
	}
	r, err := hc.Get("https://" + HTTPAddress + "/v1/say.hello")
	if err != nil {
		b.Fatal(err)
		return
	}
	fmt.Printf("Proto: %d\n", r.ProtoMajor)
	_ = r.Body.Close()

	client := benchmark.NewClient(hc, fmt.Sprintf("https://%s", HTTPAddress))

	b.Run("http.GetFeature()", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			var resp pb.Feature
			err := client.GetFeature(ctx, &pb.Point{Latitude: 409146138, Longitude: -746188906}, &resp)
			if err != nil {
				b.Fatalf("client.GetFeature failed: %v", err)
			}
		}
	})
	b.ReportAllocs()
	_ = srv.Shutdown(context.Background())
}

// WaitForConnect waits until the passed address is accepting connections.
// It will continue to attempt a connection until context is canceled.
func WaitForConnect(ctx context.Context, address string) error {
	if address == "" {
		return fmt.Errorf("WaitForConnect() requires a valid address")
	}

	var errs []error
	for {
		errs = nil
		d := net.Dialer{}
		conn, err := d.DialContext(ctx, "tcp", address)
		if err != nil {
			if ctx.Err() != nil {
				errs = append(errs, ctx.Err())
				break
			}
			errs = append(errs, err)
			time.Sleep(time.Millisecond * 100)
			continue
		}
		_ = conn.Close()
		return nil
	}

	if len(errs) == 0 {
		return nil
	}

	if len(errs) != 0 {
		var errStrings []string
		for _, err := range errs {
			errStrings = append(errStrings, err.Error())
		}
		return errors.New(strings.Join(errStrings, "\n"))
	}
	return nil
}
