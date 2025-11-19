package test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func BenchmarkCodecs(b *testing.B) {
	data, sampleRate, err := ReadWAV("testdata/example_16k_s16_mono.wav") // not in the repo; use your own file
	require.NoError(b, err)

	frameSize := sampleRate * FrameMilliseconds / 1000
	skipFrames := 50 // skip the first 50 frames
	dataFrameStart := frameSize * skipFrames
	dataFrame := data[dataFrameStart : dataFrameStart+frameSize] // use only 1 frame

	codecs := []codec{
		prepareLipstick(b, frameSize),
		prepareOpus(b, sampleRate),
	}

	for _, c := range codecs {
		b.Run(c.name, func(b *testing.B) {
			b.Run("encode", func(b *testing.B) {
				b.ReportAllocs()

				buf := make([]byte, 1024)
				for b.Loop() {
					_, _ = c.enc(dataFrame, buf)
				}
			})

			b.Run("decode", func(b *testing.B) {
				b.ReportAllocs()

				res := make([]float32, len(dataFrame))
				buf := make([]byte, 1024)
				n, _ := c.enc(dataFrame, buf)
				buf = buf[:n]

				for b.Loop() {
					_ = c.dec(buf, res)
				}
			})
		})
	}
}
