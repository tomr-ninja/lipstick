package test

import (
	"fmt"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gopkg.in/hraban/opus.v2"

	"github.com/tomr-ninja/lipstick"
)

const (
	LipstickLPCOrder  = 10
	FrameMilliseconds = 20
)

func TestCodecs(t *testing.T) {
	require.NoError(t, os.MkdirAll("out", os.ModePerm))

	for _, wavPath := range listWavs("testdata") {
		t.Run(wavPath, func(t *testing.T) {
			testCodecsWithFile(t, wavPath)
		})
	}
}

func testCodecsWithFile(t *testing.T, wavPath string) {
	data, sampleRate, err := ReadWAV(wavPath)
	require.NoError(t, err)

	frameSize := sampleRate * FrameMilliseconds / 1000
	audioLength := calcAudioLength(len(data), sampleRate)
	originalBitrate := sampleRate * bitDepth
	numFrames := len(data) / frameSize

	codecs := []codec{
		prepareLipstick(t, frameSize),
		prepareOpus(t, sampleRate),
	}

	for _, c := range codecs {
		t.Run(c.codecName, func(t *testing.T) {
			bytesTotal := 0
			biggestFrame := 0

			transcoded := make([]float32, 0, len(data))
			for i := range len(data) / frameSize {
				encoded := make([]byte, 1024)
				n, err := c.enc(data[i*frameSize:(i+1)*frameSize], encoded)
				require.NoError(t, err)
				encoded = encoded[:n]

				bytesTotal += len(encoded)
				biggestFrame = max(biggestFrame, len(encoded))

				decoded := make([]float32, frameSize)
				require.NoError(t, c.dec(encoded, decoded))

				transcoded = append(transcoded, decoded...)
			}

			originalFileName := path.Base(wavPath)
			transcodedFileName := fmt.Sprintf("%s.%s.wav", originalFileName, c.codecName)
			require.NoError(t, WriteWav(fmt.Sprintf("out/%s", transcodedFileName), transcoded, sampleRate))

			encodedBitrate := calcBitRate(bytesTotal, audioLength)
			averageFrame := bytesTotal / numFrames
			compressionRatio := float64(encodedBitrate) / float64(originalBitrate)

			t.Logf("bytesTotal: %d; biggestFrame: %d, averageFrame: %d", bytesTotal, biggestFrame, averageFrame)
			t.Logf("bitrate: %d, compression ratio: %.2f", encodedBitrate, compressionRatio)
		})
	}
}

type codec struct {
	codecName string
	enc       func(from []float32, to []byte) (int, error)
	dec       func(from []byte, to []float32) error
}

func prepareLipstick(_ testing.TB, frameSize int) codec {
	enc := lipstick.NewEncoder(LipstickLPCOrder, frameSize)
	dec := lipstick.NewDecoder(LipstickLPCOrder, frameSize)

	return codec{
		codecName: "lipstick",
		enc:       enc.Encode,
		dec:       dec.Decode,
	}
}

func prepareOpus(t testing.TB, sampleRate int) codec {
	enc, err := opus.NewEncoder(sampleRate, 1, opus.AppVoIP)
	require.NoError(t, err)

	dec, err := opus.NewDecoder(sampleRate, 1)
	require.NoError(t, err)

	return codec{
		codecName: "opus",
		enc:       enc.EncodeFloat32,
		dec: func(from []byte, to []float32) error {
			_, err := dec.DecodeFloat32(from, to)
			return err
		},
	}
}

func calcAudioLength(bytes int, sampleRate int) time.Duration {
	seconds := float64(bytes) / float64(sampleRate)

	return time.Second * time.Duration(seconds)
}

func calcBitRate(bytes int, dur time.Duration) int {
	return 8 * int(float64(bytes)/dur.Seconds())
}

func listWavs(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		panic(err)
	}

	wavs := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".wav") {
			continue
		}

		wavs = append(wavs, path.Join(dir, entry.Name()))
	}

	return wavs
}
