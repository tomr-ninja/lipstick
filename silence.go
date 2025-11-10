package lipstick

// SilenceDetector holds thresholds and runs noise floor estimate.
type SilenceDetector struct {
	noiseFloor float64
	alpha      float64 // smoothing factor
	threshold  float64 // multiplier over noise floor
}

// IsSilence checks if a frame is silence.
func (sd *SilenceDetector) IsSilence(frame []float32) bool {
	N := len(frame)
	if N == 0 {
		return true
	}

	// compute energy
	var energy float64
	for _, v := range frame {
		v64 := float64(v)
		energy += v64 * v64
	}
	energy /= float64(N)

	// update noise floor estimate (when low energy)
	if energy < sd.noiseFloor*sd.threshold {
		sd.noiseFloor = sd.alpha*sd.noiseFloor + (1-sd.alpha)*energy
	}

	// check silence
	if energy < sd.noiseFloor*sd.threshold {
		return true
	}

	return false
}
