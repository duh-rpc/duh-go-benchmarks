### Benchmarks for duh-go
The benchmarks here compare HTTP requests using duh-go with GRPC

The goal of this benchmark suite is to show that regular HTTP requests/responses are as fast, or faster than GPRC

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
