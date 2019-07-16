package sound

/*
#cgo pkg-config: alsa

#include <stdint.h>
#include <alsa/asoundlib.h>

static snd_pcm_t* openDevice(const char *deviceName, unsigned int rate) {
	int err = 0;

	snd_pcm_hw_params_t* params = NULL;
	snd_pcm_t* handle = NULL;

	if ((err = snd_pcm_open(&handle, deviceName, SND_PCM_STREAM_PLAYBACK, 0)) < 0)
	{
		fprintf(stderr, "Cannot open playback audio device %s (%s, %d)\n", deviceName, snd_strerror(err), err);
		goto out;
	}

	if ((err = snd_pcm_hw_params_malloc(&params)) < 0)
	{
		fprintf(stderr, "Cannot allocate hardware parameter structure (%s, %d)\n", snd_strerror(err), err);
		goto out;
	}

	if ((err = snd_pcm_hw_params_any(handle, params)) < 0)
	{
		fprintf(stderr, "Cannot initialize hardware parameter structure (%s, %d)\n", snd_strerror(err), err);
		goto out;
	}

	if ((err = snd_pcm_hw_params_set_access(handle, params, SND_PCM_ACCESS_RW_INTERLEAVED)) < 0)
	{
		fprintf(stderr, "Cannot set access type (%s, %d)\n", snd_strerror(err), err);
		goto out;
	}

	if ((err = snd_pcm_hw_params_set_format(handle, params, SND_PCM_FORMAT_S16_LE)) < 0)
	{
		fprintf(stderr, "Cannot set sample format (%s, %d)\n", snd_strerror(err), err);
		goto out;
	}

	if ((err = snd_pcm_hw_params_set_rate_near(handle, params, &rate, 0)) < 0)
	{
		fprintf(stderr, "Cannot set sample rate (%s, %d)\n", snd_strerror(err), err);
		goto out;
	}

	if ((err = snd_pcm_hw_params_set_channels(handle, params, 1))< 0)
	{
		fprintf(stderr, "Cannot set channel count (%s, %d)\n", snd_strerror(err), err);
		goto out;
	}

	if ((err = snd_pcm_hw_params(handle, params)) < 0)
	{
		fprintf(stderr, "Cannot set parameters (%s, %d)\n", snd_strerror(err), err);
		goto out;
	}

out:
	if (err < 0) {
		errno = -err;
		if (handle != NULL) {
			snd_pcm_close(handle);
			handle = NULL;
		}
	}

	if (params)
		snd_pcm_hw_params_free(params);

	return handle;
}

static int playback(snd_pcm_t* handle, const int16_t* buf, int bufSize) {
	int err = 0;

    if ((err = snd_pcm_writei(handle, buf, bufSize)) != bufSize)
    {
        fprintf(stderr, "write to audio interface failed (%s)\n", snd_strerror (err));
        return err;
	}

    snd_pcm_drain(handle);

	return 0;
}
*/
import "C"

import (
	"errors"
	"fmt"
	"sync"
	"unsafe"

	"github.com/rmcsoft/hasp/events"
)

// SoundPlayedEventName an event with this name is emitted when the sound is played
const SoundPlayedEventName = "SoundPlayedEvent"

// SoundPlayer sound player
type SoundPlayer struct {
	devName string

	devClosedCond *sync.Cond
	devMutex      *sync.Mutex
	dev           *C.snd_pcm_t
}

// NewSoundPlayer creates new SoundPlayer
func NewSoundPlayer(devName string) (*SoundPlayer, error) {
	sp := &SoundPlayer{
		devName:  devName,
		devMutex: &sync.Mutex{},
	}
	sp.devClosedCond = sync.NewCond(sp.devMutex)
	return sp, nil
}

// Play starts playing back buffer
func (p *SoundPlayer) Play(audioData *AudioData) (events.EventSource, error) {

	p.devMutex.Lock()
	defer p.devMutex.Unlock()
	p.stop(false)

	if audioData.SampleType() != S16LE || audioData.ChannelCount() != 1 {
		return nil, errors.New("Unsupported audio format")
	}

	p.dev = C.openDevice(C.CString(p.devName), C.uint(audioData.SampleRate()))
	if p.dev == nil {
		return nil, errors.New("Could't open audio device for playback")
	}

	asyncPlay := func() *events.Event {
		fmt.Println("StartPlay")
		sampleCount := audioData.SampleCount()
		if sampleCount == 0 {
			fmt.Println("NothingToPlay")
			p.closeDev()
			return &events.Event{Name: SoundPlayedEventName}
		}
		samples := audioData.Samples()
		cptr := (*C.int16_t)(unsafe.Pointer(&samples[0]))
		C.playback(p.dev, cptr, C.int(sampleCount))
		p.closeDev()
		fmt.Println("StopPlay")
		return &events.Event{Name: SoundPlayedEventName}
	}

	return events.NewSingleEventSource("SoundPlayerEventSource", asyncPlay), nil
}

// Stop playing
func (p *SoundPlayer) Stop() {
	p.stop(true)
}

func (p *SoundPlayer) stop(useLock bool) {
	if useLock {
		p.devMutex.Lock()
		defer p.devMutex.Unlock()
	}

	if p.dev != nil {
		C.snd_pcm_drop(p.dev)

		for p.dev != nil {
			p.devClosedCond.Wait()
		}
	}
}

func (p *SoundPlayer) closeDev() {
	p.devMutex.Lock()
	if p.dev != nil {
		C.snd_pcm_close(p.dev)
		p.dev = nil
	}
	p.devClosedCond.Signal()
	p.devMutex.Unlock()
}
