package sound

// SampleType is numerical representation of sample
type SampleType int

const (
	// S16LE Signed 16 bit Little Endian
	S16LE SampleType = iota
)

// AudioFormat audio Data Description
type AudioFormat struct {
	ChannelCount int
	SampleType   SampleType
	SampleRate   int
}

// AudioData audio data
type AudioData struct {
	format  AudioFormat
	samples []byte
}

// NewMonoS16LE creates new
func NewMonoS16LE(samples []byte, sampleRate int) *AudioData {
	return &AudioData{
		format: AudioFormat{
			ChannelCount: 1,
			SampleType:   S16LE,
			SampleRate:   sampleRate,
		},
		samples: samples,
	}
}

// Samples gets samples
func (a *AudioData) Samples() []byte {
	return a.samples
}

// Format gets AudioFormat
func (a *AudioData) Format() AudioFormat {
	return a.format
}

// SampleType gets sample type
func (a *AudioData) SampleType() SampleType {
	return a.format.SampleType
}

// SampleRate gets sample rate
func (a *AudioData) SampleRate() int {
	return a.format.SampleRate
}

// SampleSize returns sample size
func (a *AudioData) SampleSize() int {
	return a.format.SampleType.Size()
}

// SampleCount gets sample count
func (a *AudioData) SampleCount() int {
	return len(a.samples) / a.SampleSize()
}

// Size returns sample size
func (st SampleType) Size() int {
	switch st {
	case S16LE:
		return 2
	default:
		panic("Invalid SampleType")
	}
}
