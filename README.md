# 💄 lipstick

This is a simple audio codec for VoIP applications. It mostly uses outdated techniques, but on the flip side, it has
zero dependencies.

The point of this is not to compete with the best, but rather:

1) Have a relatively small implementation that is easy to understand and recreate in other programming languages.
2) Be a library without external dependencies like libopus or libvorbis and therefore make it simple to use native
Golang's cross-compilation and avoid complicating the build process in general.

**Both encoder and decoder are implemented.**

If you need a high-quality production-ready codec, consider using [Opus](https://github.com/hraban/opus) instead.

> [!WARNING]
> lipstick is unstable and experimental. I won't release a version until it's tested, but I am willing to make
> breaking changes before 1.0.

## Compression

lipstick achieves ~3X compression ratio for 16 kHz mono voice data. A 20ms frame is ~200 bytes.
That is ~5 times worse compared to Opus on 'Application: VoIP' quality setting, but still well below 
[maximum safe UDP packet size](https://stackoverflow.com/questions/1098897/what-is-the-largest-safe-udp-packet-size-on-the-internet).
So it's not an amazing compression, but it would technically work in production in real-time.

## Testing

For the sake of keeping the `go.mod` file clean, tests are moved to a separate module (`./test`)

```shell
cd test
go test -v # -v for verbose output with logs/stats
```

Tests are run using WAV files provided by the user. Place your files in `test/testdata` directory; they must be
**16 bit mono** uncompressed WAV files.
The test will produce transcoded (raw -> lipstick OR Opus -> raw) WAV files in `test/out` directory. You can just
listen to them to compare the quality of raw, lipstick-encoded and Opus-encoded audio.

You can use services like [this](https://cloudconvert.com/wav-converter) to convert your files to the required format.

I really don't know yet how to measure audio quality quantitatively; I guess there are some metrics out there,
but as of now I don't have any intention to implement them. I've tried things like MSE, but wasn't happy with the
results.

## Benchmark

There are some avoidable allocations; nonetheless, performance is decent.

```
goos: darwin
goarch: arm64
pkg: github.com/tomr-ninja/lipstick/test
cpu: Apple M3 Pro
BenchmarkCodecs/lipstick/encode-11        115600             10607 ns/op            3416 B/op         17 allocs/op
BenchmarkCodecs/lipstick/decode-11        125970              9651 ns/op            3120 B/op         10 allocs/op
BenchmarkCodecs/opus/encode-11              9814            128441 ns/op               0 B/op          0 allocs/op
BenchmarkCodecs/opus/decode-11            242533              4893 ns/op               0 B/op          0 allocs/op
```
