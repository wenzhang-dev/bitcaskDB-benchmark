# bitcaskDB-benchmark

 Here we compare the read/write 4KB performance of several popular KV store engines, including badger, levelDB, and BoltDB.

```shell
go test -bench=Read -benchtime=60s -timeout=30m -count=3
goos: linux
goarch: amd64
pkg: github.com/wenzhang-dev/bitcaskDB-benchmark
cpu: Intel(R) Xeon(R) Gold 5318N CPU @ 2.10GHz
BenchmarkReadWithBitcaskDB/read4K-8  11459024   6313 ns/op  1.217 AvgRSS(GB)  1.275 PeakRSS(GB)  10120 B/op  10 allocs/op
BenchmarkReadWithBitcaskDB/read4K-8  12512324   6522 ns/op  1.220 AvgRSS(GB)  1.234 PeakRSS(GB)  10120 B/op  10 allocs/op
BenchmarkReadWithBitcaskDB/read4K-8  12414660   6468 ns/op  1.206 AvgRSS(GB)  1.231 PeakRSS(GB)  10120 B/op  10 allocs/op
BenchmarkReadWithBadger/read4K-8      4575487  13526 ns/op  2.716 AvgRSS(GB)  4.350 PeakRSS(GB)  19416 B/op  43 allocs/op
BenchmarkReadWithBadger/read4K-8      4960239  13741 ns/op  1.629 AvgRSS(GB)  1.681 PeakRSS(GB)  19406 B/op  43 allocs/op
BenchmarkReadWithBadger/read4K-8      4851144  14429 ns/op  1.591 AvgRSS(GB)  1.650 PeakRSS(GB)  19422 B/op  44 allocs/op
BenchmarkReadWithLevelDB/read4K-8     1569663  50710 ns/op  0.111 AvgRSS(GB)  0.134 PeakRSS(GB)  55021 B/op  35 allocs/op
BenchmarkReadWithLevelDB/read4K-8     1000000  63066 ns/op  0.113 AvgRSS(GB)  0.129 PeakRSS(GB)  54264 B/op  35 allocs/op
BenchmarkReadWithLevelDB/read4K-8     1236408  57268 ns/op  0.114 AvgRSS(GB)  0.138 PeakRSS(GB)  54624 B/op  35 allocs/op
BenchmarkReadWithBoltDB/read4K-8     12587562   5269 ns/op  5.832 AvgRSS(GB)  5.838 PeakRSS(GB)    832 B/op  13 allocs/op
BenchmarkReadWithBoltDB/read4K-8     16920481   4482 ns/op  5.832 AvgRSS(GB)  5.833 PeakRSS(GB)    832 B/op  13 allocs/op
BenchmarkReadWithBoltDB/read4K-8     19141418   5276 ns/op  5.832 AvgRSS(GB)  5.835 PeakRSS(GB)    832 B/op  13 allocs/op
PASS
ok   github.com/wenzhang-dev/bitcaskDB-benchmark 1475.172s
```

```shell
go test -bench=Write -benchtime=60s -timeout=30m -count=3
goos: linux
goarch: amd64
pkg: github.com/wenzhang-dev/bitcaskDB-benchmark
cpu: Intel(R) Xeon(R) Gold 5318N CPU @ 2.10GHz
BenchmarkWriteWithBitcaskDB/write4K-8  8334304  13217 ns/op  0.7905 AvgRSS(GB)   0.934 PeakRSS(GB)    1666 B/op   11 allocs/op
BenchmarkWriteWithBitcaskDB/write4K-8  5323338  14976 ns/op  0.9732 AvgRSS(GB)   1.058 PeakRSS(GB)    1727 B/op   12 allocs/op
BenchmarkWriteWithBitcaskDB/write4K-8  5435398  13929 ns/op  0.9639 AvgRSS(GB)   1.122 PeakRSS(GB)    1756 B/op   12 allocs/op
BenchmarkWriteWithLevelDB/write4K-8    1047753  68691 ns/op  0.0615 AvgRSS(GB)  0.0636 PeakRSS(GB)    2946 B/op   16 allocs/op
BenchmarkWriteWithLevelDB/write4K-8    1179555  71497 ns/op  0.0617 AvgRSS(GB)  0.0634 PeakRSS(GB)    3250 B/op   18 allocs/op
BenchmarkWriteWithLevelDB/write4K-8     992488  74130 ns/op  0.0613 AvgRSS(GB)  0.0625 PeakRSS(GB)    3444 B/op   19 allocs/op
BenchmarkWriteWithBadger/write4K-8     3776720  20036 ns/op   6.409 AvgRSS(GB)   7.534 PeakRSS(GB)   30062 B/op   68 allocs/op
BenchmarkWriteWithBadger/write4K-8     4106070  50959 ns/op   10.77 AvgRSS(GB)   13.63 PeakRSS(GB)  115442 B/op  152 allocs/op
BenchmarkWriteWithBadger/write4K-8     1491906  49955 ns/op   11.45 AvgRSS(GB)   13.72 PeakRSS(GB)   88941 B/op  130 allocs/op
BenchmarkWriteWithBoltDB/write4K-8     2808206  23131 ns/op   0.626 AvgRSS(GB)   0.999 PeakRSS(GB)    7579 B/op   11 allocs/op
BenchmarkWriteWithBoltDB/write4K-8     4303538  22836 ns/op   1.713 AvgRSS(GB)   2.971 PeakRSS(GB)    7765 B/op   11 allocs/op
BenchmarkWriteWithBoltDB/write4K-8     3755002  19385 ns/op   2.481 AvgRSS(GB)   2.872 PeakRSS(GB)    7896 B/op   12 allocs/op
PASS
ok   github.com/wenzhang-dev/bitcaskDB-benchmark 1541.068s
```
