### Benchmark GRPC vs HTTP
This is a simple benchmark that uses the [Route
Guide](https://github.com/grpc/grpc-go/tree/master/examples/route_guide) code
as provided by the GRPC project to compare gPRC performance with standard HTTP
in golang.

### Test Setup
The server hosts a lone instance of RouteGuideService that awaits requests
through gRPC, HTTP/1, HTTP/2(H2C), or HTTP/2(TLS). Before executing each test,
we check the port to confirm the service is operational and ready to accept
requests. Then, a client tailored for that particular test is set up, and an
initial request is dispatched to connect to the server which ensures a live
connection exists between the client and server before the benchmark begins.

### Apples to Apples Comparison
Each test involves a single-threaded request to the GetFeature() method of the
RouteGuideService. Both gRPC and HTTP tests utilize protobuf for serialization
to ensure an equitable comparison, without the inclusion of any extra
middleware or injectors. The HTTP handler employs a straightforward switch for
routing requests to handlers, mirroring the way gRPC manages route dispatching
internally. In fact, gRPC usually outpaces REST because while REST facilitates
intricate route handling, gRPC simply matches the request path through a basic
string comparison. Our objective with these measures is to provide the most
impartial comparison between gRPC and HTTP.

### Results
The results are quite surprising! HTTP/2 (H2C and TLS) is slower than gRPC, and
gRPC is slower than HTTP/1!

```bash
$ go test -bench=. -benchmem=1  -benchtime=30s
goos: darwin
goarch: arm64
pkg: github.com/duh-rpc/duh-go-benchmarks
BenchmarkGRPC
BenchmarkGRPC/grpc.GetFeature()
BenchmarkGRPC/grpc.GetFeature()-10         	   19750	     56548 ns/op
BenchmarkHTTP2
Proto: 2
BenchmarkHTTP2/http.GetFeature()
BenchmarkHTTP2/http.GetFeature()-10        	   16590	     72107 ns/op
BenchmarkHTTP1
Proto: 1
BenchmarkHTTP1/http.GetFeature()
BenchmarkHTTP1/http.GetFeature()-10        	   24290	     49005 ns/op
BenchmarkHTTPS
2023/10/02 12:31:54 Generating CA Certificates....
2023/10/02 12:31:54 Generating Server Private Key and Certificate....
2023/10/02 12:31:54 Cert DNS names: (localhost)
2023/10/02 12:31:54 Cert IPs: (127.0.0.1)
2023/10/02 12:31:54 http: TLS handshake error from 127.0.0.1:51711: EOF
Proto: 2
BenchmarkHTTPS/http.GetFeature()
BenchmarkHTTPS/http.GetFeature()-10        	   16632	     71318 ns/op
PASS
```

### HTTP/1 is faster than HTTP/2 on golang
This is a known issue and is well documented.
* https://github.com/golang/go/issues/47840
* https://github.com/nspeed-app/http2issue
* https://github.com/kgersen/h3ctx
* https://www.emcfarlane.com/blog/2023-05-15-grpc-servehttp

### HTTP/1 is faster than GRPC on golang
We initiated this project after observing that our HTTP/1 services were
consistently outpacing our gRPC-based services in production. The fact that a
benchmark corroborated our production findings was surprising. We had
previously believed—and to some extent, still believe—that the performance
difference was more linked to our implementation than gRPC's lack of high-speed
performance.

We are in the process of moving a single service back to HTTP/1 to gauge the
impact this will have on our systems.

Please feel free to open an issue if you have a comment on this benchmark.
