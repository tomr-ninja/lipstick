package lipstick

import (
	"bytes"
	"compress/flate"
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	magicHeader  = "BBRT"
	silentHeader = "BBRS"
)

// Encoder implements Encoder
type Encoder struct {
	order           int
	frameSize       int
	silenceDetector SilenceDetector
	history         []float64
	huffman         *flate.Writer
	huffmanBuf      *bytes.Buffer
	bufs            struct {
		qlpc       []int16
		res        []float64
		qres       []int8
		qresBinned []uint8
	}
}

func NewEncoder(order, frameSize int) *Encoder {
	enc := &Encoder{
		order:     order,
		frameSize: frameSize,
		silenceDetector: SilenceDetector{
			noiseFloor: 1e-6, // initial guess
			alpha:      0.95, // smoothing
			threshold:  3.0,  // energy must exceed the noise floor by this factor
		},
		history:    make([]float64, order),
		huffmanBuf: bytes.NewBuffer(make([]byte, 0, frameSize)),
		bufs: struct {
			qlpc       []int16
			res        []float64
			qres       []int8
			qresBinned []uint8
		}{
			qlpc:       make([]int16, order),
			res:        make([]float64, frameSize),
			qres:       make([]int8, frameSize),
			qresBinned: make([]uint8, frameSize),
		},
	}

	var err error
	enc.huffman, err = flate.NewWriter(enc.huffmanBuf, flate.HuffmanOnly)
	if err != nil {
		panic(err)
	}

	return enc
}

func (e *Encoder) Encode(from []float32, to []byte) (int, error) {
	if len(from) != e.frameSize {
		return 0, fmt.Errorf("from length must be %d", e.frameSize)
	}
	if len(to) != 0 {
		to = to[:0]
	}

	if e.silenceDetector.IsSilence(from) {
		to = append(to, []byte(silentHeader)...)
		return len(silentHeader), nil
	}

	order := e.order
	x := make([]float64, e.frameSize)
	for i := 0; i < e.frameSize; i++ {
		x[i] = float64(from[i])
	}

	lpcCoeffs, err := levinsonDurbin(autoCorrelation(x, order), order)
	if err != nil {
		return 0, err
	}

	residual := e.bufs.res
	for n := 0; n < e.frameSize; n++ {
		var pred float64
		for k := 1; k <= order; k++ {
			idx := n - k
			var sample float64
			if idx >= 0 {
				sample = x[idx]
			} else {
				// pull from previous frame history
				// history layout: history[0] is oldest, history[order-1] is most recent
				hIdx := order + idx // idx is negative
				if hIdx >= 0 {
					sample = e.history[hIdx]
				}
			}
			pred += lpcCoeffs[k-1] * sample
		}
		residual[n] = x[n] + pred
	}

	// Quantize residual to int8
	qres := e.bufs.qres
	scaleRes := quantize8(residual, qres)
	qresBinned := e.bufs.qresBinned
	bin(qres, qresBinned)

	// Compress residual using Huffman coding
	e.huffmanBuf.Reset()
	e.huffman.Reset(e.huffmanBuf)
	if err = binary.Write(e.huffman, binary.LittleEndian, qresBinned); err != nil {
		return 0, err
	}
	if err = e.huffman.Close(); err != nil {
		return 0, err
	}
	qresCompressed := e.huffmanBuf.Bytes()

	// Pack into bytes
	buf := bytes.NewBuffer(to)

	// header
	buf.Write([]byte(magicHeader)) // 4 bytes
	buf.WriteByte(byte(order))     // 1 byte
	// LPC
	if order > 3 {
		// main 3 LPC coefficients first
		if err = bufWriteQuantized(buf, lpcCoeffs[:3], e.bufs.qlpc[:3], quantize16); err != nil {
			return 0, err
		}
		// remaining LPC coefficients next
		if err = bufWriteQuantized(buf, lpcCoeffs[3:], e.bufs.qlpc[3:], quantize16); err != nil {
			return 0, err
		}
	} else {
		// write everything at once
		if err = bufWriteQuantized(buf, lpcCoeffs, e.bufs.qlpc, quantize16); err != nil {
			return 0, err
		}
	}
	// residual scale; 32 bit float is enough
	if err = binary.Write(buf, binary.LittleEndian, scaleRes); err != nil {
		return 0, err
	}
	// residual number of compressed bytes
	if err = binary.Write(buf, binary.LittleEndian, uint16(len(qresCompressed))); err != nil {
		return 0, err
	}
	// residual compressed bytes
	if _, err = buf.Write(qresCompressed); err != nil {
		return 0, err
	}

	// Update history with tail of current frame for next call
	copy(e.history, x[e.frameSize-order:])

	return buf.Len(), nil
}

// autoCorrelation computes autocorrelation r[0..order] for signal x
func autoCorrelation(x []float64, order int) []float64 {
	r := make([]float64, order+1)
	n := len(x)
	for k := 0; k <= order; k++ {
		var sum float64
		for i := 0; i < n-k; i++ {
			sum += x[i] * x[i+k]
		}
		r[k] = sum
	}

	return r
}

// Levinson-Durbin: solve Toeplitz system for LPC coefficients
// returns a[0..order-1] corresponding to a1..aP (predictor coefficients)
// reference: standard Levinson-Durbin recursion
func levinsonDurbin(r []float64, order int) ([]float64, error) {
	if len(r) < order+1 {
		return nil, errors.New("autocorrelation too short")
	}
	a := make([]float64, order)
	var e = r[0]
	if e == 0 {
		// silent frame -> zero coefficients
		for i := range a {
			a[i] = 0
		}
		return a, nil
	}
	// temporary arrays
	ref := make([]float64, order)
	for i := 0; i < order; i++ {
		// compute lambda (kth reflection)
		var acc float64
		for j := 0; j < i; j++ {
			acc += a[j] * r[i-j]
		}
		num := -(r[i+1] + acc)
		k := num / e
		ref[i] = k
		// update a
		newA := make([]float64, i+1)
		for j := 0; j < i; j++ {
			newA[j] = a[j] + k*a[i-1-j]
		}
		newA[i] = k
		for j := 0; j <= i; j++ {
			a[j] = newA[j]
		}
		// update prediction error
		e = e * (1.0 - k*k)
		if e <= 0 {
			e = 1e-9
		}
	}

	return a, nil
}

// bin maps int8 residuals to a compact uint8 symbol set:
//
//	0 -> 0
//	1 <= |v| <= 4: exact (index = |v|, sign in bit7)
//	|v| >= 5: logarithmic bins with mid-point reconstruction (indices 5..9)
//
// Indices (lower 7 bits):
//
//	5: [5,7]
//	6: [8,15]
//	7: [16,31]
//	8: [32,63]
//	9: [64,127]
//
// Sign: bit7 = 1 if negative.
// This keeps small residuals lossless (removing most audible distortion) while
// retaining strong entropy concentration for Huffman-only compression.
func bin(in []int8, out []uint8) []uint8 {
	for i, v := range in {
		if v == 0 {
			out[i] = 0
			continue
		}
		neg := v < 0
		m := int(v)
		if m < 0 {
			m = -m
		}
		if m > 127 {
			m = 127
		}

		var idx uint8
		if m <= 4 {
			idx = uint8(m) // exact
		} else {
			switch {
			case m <= 7:
				idx = 5
			case m <= 15:
				idx = 6
			case m <= 31:
				idx = 7
			case m <= 63:
				idx = 8
			default:
				idx = 9 // 64..127
			}
		}
		if neg {
			idx |= 0x80
		}
		out[i] = idx
	}
	return out
}

// unbin reverses bin (lossless for |v| ≤ 4, approximate otherwise).
// For log bins it returns the midpoint of the original magnitude range.
func unbin(in []uint8, to []int8) {
	for i, code := range in {
		if code == 0 {
			to[i] = 0
			continue
		}
		neg := (code & 0x80) != 0
		idx := int(code & 0x7F)

		var mag int
		switch {
		case idx >= 1 && idx <= 4:
			mag = idx
		case idx == 5:
			mag = (5 + 7) / 2
		case idx == 6:
			mag = (8 + 15) / 2
		case idx == 7:
			mag = (16 + 31) / 2
		case idx == 8:
			mag = (32 + 63) / 2
		case idx == 9:
			mag = (64 + 127) / 2
		default:
			// Fallback: treat unknown as zero (should not happen)
			to[i] = 0
			continue
		}
		if mag > 127 {
			mag = 127
		}
		if neg {
			mag = -mag
		}
		to[i] = int8(mag)
	}
}
