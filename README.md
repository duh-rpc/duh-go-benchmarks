### Benchmarks for duh-go
The benchmarks here compare HTTP requests using duh-go with GRPC

The goal of this benchmark suite is to show that regular HTTP requests/responses are as fast, or faster than GPRC

```bash
$ go test -bench=. -benchmem=1  -benchtime=30s
goos: darwin
goarch: arm64
pkg: github.com/duh-rpc/duh-go-benchmarks
BenchmarkGRPCServer/grpc.GetFeature()-10                  642619             55743 ns/op           12950 B/op        326 allocs/op
BenchmarkHTTPServer/http.GetFeature()-10                  735146             49172 ns/op           11412 B/op        272 allocs/op
PASS
ok      github.com/duh-rpc/duh-go-benchmarks    73.998s
```