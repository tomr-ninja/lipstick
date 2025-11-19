package test

import (
	"fmt"
	"log"
	"math"
	"os"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
)

const bitDepth = 16

func ReadWAV(path string) (data []float32, sampleRate int, err error) {
	f, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	decoder := wav.NewDecoder(f)
	if !decoder.IsValidFile() {
		return nil, 0, fmt.Errorf("invalid WAV file")
	}
	if decoder.NumChans != 1 {
		return nil, 0, fmt.Errorf("only mono WAV files are supported")
	}
	if decoder.BitDepth != bitDepth {
		return nil, 0, fmt.Errorf("only 16-bit WAV files are supported")
	}

	buf, err := decoder.FullPCMBuffer()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to decode WAV file: %w", err)
	}

	// convert int16 values to float32
	floatBuf := make([]float32, len(buf.Data))
	for i, v := range buf.Data {
		floatBuf[i] = float32(v) / math.MaxInt16
	}

	return floatBuf, int(decoder.SampleRate), nil
}

// WriteWav writes float32 samples (-1..+1) into a mono WAV file
func WriteWav(filename string, data []float32, sampleRate int) error {
	// open file
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	// configure encoder (16-bit PCM, mono)
	enc := wav.NewEncoder(f, sampleRate, bitDepth, 1, 1)

	// convert float32 [-1,1] to int values
	ints := make([]int, len(data))
	for i, v := range data {
		if v > 1.0 {
			v = 1.0
		}
		if v < -1.0 {
			v = -1.0
		}
		ints[i] = int(v * math.MaxInt16) // 16-bit signed PCM
	}

	buf := &audio.IntBuffer{
		Format: &audio.Format{
			NumChannels: 1,
			SampleRate:  sampleRate,
		},
		Data:           ints,
		SourceBitDepth: bitDepth,
	}

	if err = enc.Write(buf); err != nil {
		return err
	}

	if err = enc.Close(); err != nil {
		return err
	}

	return nil
}
