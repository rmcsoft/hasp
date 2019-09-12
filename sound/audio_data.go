package sound

import (
	"fmt"
	"io/ioutil"
)

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

// Mime gets MIME for AudioFormat
func (af *AudioFormat) Mime() string {
	return fmt.Sprintf("audio/%v; rate=%v; channels=%v",
		af.SampleType.Mime(),
		af.SampleRate,
		af.ChannelCount,
	)
}

// NewAudioData creates new AudioData
func NewAudioData(format AudioFormat, samples []byte) *AudioData {
	return &AudioData{
		format:  format,
		samples: samples,
	}
}

// NewMonoS16LE creates new AudioData
func NewMonoS16LE(sampleRate int, samples []byte) *AudioData {
	return &AudioData{
		format: AudioFormat{
			ChannelCount: 1,
			SampleType:   S16LE,
			SampleRate:   sampleRate,
		},
		samples: samples,
	}
}

func LoadMonoS16LEFromPCM(fileName string, sampleRate int) (*AudioData, error) {
	samples, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	return NewMonoS16LE(sampleRate, samples), nil
}

/*
// NewMonoS16LEFromInt16 creates new AudioData from []int16
func NewMonoS16LEFromInt16(sampleRate int, samples []int16) *AudioData {
	buffer := bytes.NewBuffer(make([]byte, 0, 2*len(samples)))
	binary.Write(buffer, binary.LittleEndian, samples)
	return NewMonoS16LE(sampleRate, buffer.Bytes())
}
*/

// Samples gets samples
func (a *AudioData) Samples() []byte {
	return a.samples
}

// Format gets AudioFormat
func (a *AudioData) Format() AudioFormat {
	return a.format
}

// ChannelCount gets channel count
func (a *AudioData) ChannelCount() int {
	return a.format.ChannelCount
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

// Mime gets MIME for AudioData
func (a *AudioData) Mime() string {
	return a.format.Mime()
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

// Mime gets MIME for SampleType
func (st SampleType) Mime() string {
	switch st {
	case S16LE:
		return "l16"
	default:
		panic("Invalid SampleType")
	}
}
