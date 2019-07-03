package hasp

/*
#cgo pkg-config: alsa

// Porcupine
#cgo CFLAGS: -I${SRCDIR}/../Porcupine/include
#cgo linux,amd64 LDFLAGS: -L${SRCDIR}/../Porcupine/lib/linux/x86_64
#cgo linux,arm   LDFLAGS: -L${SRCDIR}/../Porcupine/lib/beaglebone
#cgo LDFLAGS: -lpv_porcupine

#include <errno.h>
#include <stdbool.h>
#include <stdlib.h>
#include <stdint.h>
#include <string.h>

#include <alsa/asoundlib.h>
#include <pv_porcupine.h>

typedef struct {
	volatile int32_t* stopFlagPtr;

	snd_pcm_t* capDev;
	pv_porcupine_object_t* porcupine;
} Detector;

static snd_pcm_t* openCaptureDev(const char* deviceName, unsigned int rate) {
	int err;

	snd_pcm_hw_params_t* params = NULL;
	snd_pcm_t* handle = NULL;

	if ((err = snd_pcm_open(&handle, deviceName, SND_PCM_STREAM_CAPTURE, 0)) < 0) {
		fprintf(stderr, "Cannot open capture audio device %s (%s, %d)\n", deviceName, snd_strerror(err), err);
		goto out;
	}

	if ((err = snd_pcm_hw_params_malloc(&params)) < 0) {
		fprintf(stderr, "Cannot allocate hardware parameter structure (%s, %d)\n", snd_strerror(err), err);
		goto out;
	}

	if ((err = snd_pcm_hw_params_any(handle, params)) < 0) {
		fprintf(stderr, "Cannot initialize hardware parameter structure (%s, %d)\n", snd_strerror(err), err);
		goto out;
	}

	if ((err = snd_pcm_hw_params_set_access(handle, params, SND_PCM_ACCESS_RW_INTERLEAVED)) < 0) {
		fprintf(stderr, "Cannot set access type (%s, %d)\n", snd_strerror(err), err);
		goto out;
	}

	if ((err = snd_pcm_hw_params_set_format(handle, params,SND_PCM_FORMAT_S16_LE)) < 0) {
		fprintf(stderr, "Cannot set sample format (%s, %d)\n", snd_strerror(err), err);
		goto out;
	}

	if ((err = snd_pcm_hw_params_set_rate_near(handle, params, &rate, 0)) < 0) {
		fprintf(stderr, "Cannot set sample rate (%s, %d)\n", snd_strerror(err), err);
		goto out;
	}

	if ((err = snd_pcm_hw_params_set_channels(handle, params, 1)) < 0) {
		fprintf(stderr, "Cannot set channel count (%s, %d)\n", snd_strerror(err), err);
		goto out;
	}

	if ((err = snd_pcm_hw_params(handle, params)) < 0) {
		fprintf(stderr, "Cannot set parameters (%s, %d)\n", snd_strerror(err), err);
		goto out;
	}

out:
	if (err < 0) {
		errno = -err;
		if (handle != NULL)
			snd_pcm_close(handle);
	}

	if (params)
		snd_pcm_hw_params_free(params);

	return handle;
}

pv_porcupine_object_t* createPorcupine(const char *modelPath, const char *keywordPath, float sensitivity) {
	pv_porcupine_object_t* porcupine = NULL;
	pv_status_t status = pv_porcupine_init(modelPath, keywordPath, sensitivity, &porcupine);
	if (status != PV_STATUS_SUCCESS) {
		fprintf(stderr, "Failed to initialize Porcupine\n");
		return NULL;
	}

	return porcupine;
}

static void destroyDetector(Detector* d) {
	if (d != NULL) {
		if (d->capDev != NULL) {
			snd_pcm_close(d->capDev);
			d->capDev = NULL;
		}

		if (d->porcupine != NULL) {
			pv_porcupine_delete(d->porcupine);
			d->porcupine = NULL;
		}
	}
}

static Detector* newDetector(
	const char* deviceName, unsigned int rate,
	const char *modelPath, const char *keywordPath, float sensitivity,
	int32_t* stopFlagPtr)
{
	Detector* d = calloc(1, sizeof(Detector));
	if (d == NULL)
		return NULL;

	d->capDev = openCaptureDev(deviceName, rate);
	if (d->capDev == NULL)
		goto error;

	d->porcupine = createPorcupine(modelPath, keywordPath, sensitivity);
	if (d->porcupine == NULL)
		goto error;

	d->stopFlagPtr = stopFlagPtr;
	return d;

error:
	destroyDetector(d);
	return NULL;
}

static bool startDetect(Detector* d) {
	int err = snd_pcm_start(d->capDev);
	if (err < 0) {
		fprintf(stderr, "Cannot start soundcard (%s, %d)\n", snd_strerror(err), err);
		return false;
	}
	return true;
}

static int readSamples(Detector* d, int16_t* buf, int maxSampleCount) {
	int err;
	while (*d->stopFlagPtr == 0) {
		int n = snd_pcm_readi(d->capDev, buf, maxSampleCount);
		if (n == 0)
			continue;

		if (n > 0) {
			return n;
		}

		err = n;
		fprintf(stderr, "read from audio interface failed (%s, %d)\n", snd_strerror(err), err);
		if (err != -32)
			return err;

		// Broken pipe
		if ((err = snd_pcm_prepare(d->capDev)) < 0) {
			fprintf(stderr, "Cannot prepare audio interface for use (%s, %d)\n", snd_strerror(err), err);
			return err;
		}
	}
	return 0;
}

#define DEBUG_VOICE
#define NOISE_THRESHOLD 5000
#define NOISE_FRAMES 30

static short maxSampleValue(int16_t* samples, int sampleCount) {
	int16_t max = INT16_MAX;
	int16_t* end = samples + sampleCount;
	for (; samples != end; ++samples) {
		int16_t v = *samples;
		if (v < 0)
			v = -v;
		if (v > max)
			max = v;
	}
	return max;
}

static int waitHotWord(Detector* d) {
	const int bufSize = pv_porcupine_frame_length();
	int16_t buf[bufSize];

	bool detected = false;
	while (*d->stopFlagPtr == 0) {
		int n = readSamples(d, buf, bufSize);
		if (n < 0)
			return n;

		pv_porcupine_process(d->porcupine, buf, &detected);
		if (detected) {
#ifdef DEBUG_VOICE
			time_t rawtime;
			struct tm* timeinfo;

			time(&rawtime);
			timeinfo = localtime(&rawtime);

			// Detected keyword. Do something!
			printf("\n%s Detected keyword!\n", asctime(timeinfo));
#endif
			return 0;
		}
	}

	return -EINTR;
}

static int readVoice(Detector* d, int16_t* voiceBuffer, int maxSampleCount) {
	const int bufSize = pv_porcupine_frame_length();
	int16_t buf[bufSize];

	int16_t silenseSens = -1;
	int currentBufferFill = 0;
	int startSilenceFrames = 0;
	while (*d->stopFlagPtr == 0) {
		int n = readSamples(d, buf, bufSize);
		if (n < 0)
			return n;

		int16_t maxLoud = maxSampleValue(buf, n);

		if (n > maxSampleCount - currentBufferFill)
			n = maxSampleCount - currentBufferFill;
		if (silenseSens < 0) {
			if (maxLoud > NOISE_THRESHOLD) {
				#ifdef DEBUG_VOICE
				printf("%d[", maxLoud); fflush(stdout);
				#endif

				silenseSens = NOISE_FRAMES;
				memcpy(voiceBuffer + currentBufferFill, buf, n*sizeof(int16_t));
				currentBufferFill += n;
			} else {
				#ifdef DEBUG_VOICE
				printf("?"); fflush(stdout);
				#endif

				startSilenceFrames++;
				if (startSilenceFrames >= NOISE_FRAMES) {
					return 0;
				}
			}
		} else {
			memcpy(voiceBuffer + currentBufferFill, buf, n*sizeof(int16_t));
			currentBufferFill += n;

			if (maxLoud > NOISE_THRESHOLD) {
				if (silenseSens < NOISE_FRAMES) {
					#ifdef DEBUG_VOICE
					printf("+"); fflush(stdout);
					#endif

					silenseSens = NOISE_FRAMES;
				} else {
					#ifdef DEBUG_VOICE
					printf("."); fflush(stdout);
					#endif
				}
			} else {
				#ifdef DEBUG_VOICE
				printf("-"); fflush(stdout);
				#endif

				silenseSens--;
				if (silenseSens <= 0) {
					#ifdef DEBUG_VOICE
					printf("]"); fflush(stdout);
					#endif

					return currentBufferFill;
				}
			}
		}

		if (currentBufferFill == maxSampleCount) {
			return currentBufferFill;
		}
	}

	return -EINTR;
}

static int detect(Detector* d, int16_t* voiceBuffer, int maxSampleCount) {
	int err = waitHotWord(d);
	if (err)
		return err;
	return readVoice(d, voiceBuffer, maxSampleCount);
}
*/
import "C"

import (
	"errors"
	"fmt"
	"sync/atomic"
)

const (
	voiceBufSize int     = 64 * 1024
	sampleRate   int     = 16000
	sensitivity  float32 = 0.5
)

type hotWordDetector struct {
	eventChan chan *Event
	stopFlag  int32

	detector *C.Detector
}

const (
	// HotWordDetectedEventName the name of the HotWordDetected event.
	HotWordDetectedEventName = "HotWordDetected"
)

// NewHotWordDetectedEvent creates HotWordDetectedEvent
func NewHotWordDetectedEvent(samples []int16) *Event {
	return &Event{
		Name: HotWordDetectedEventName,
		Args: []interface{}{
			samples,
		},
	}
}

// GetVoice gets voice from event
func (event *Event) GetVoice() ([]int16, error) {
	if event.Name != HotWordDetectedEventName {
		return nil, fmt.Errorf("")
	}

	if len(event.Args) != 1 {
		return nil, errors.New("Event does not contain voice")
	}

	samples, ok := event.Args[0].([]int16)
	if !ok {
		return nil, errors.New("Event does not contain voice")
	}

	return samples, nil
}

// NewHotWordDetector creates HotWordDetector
func NewHotWordDetector(deviceName string, modelPath string, keywordPath string) (EventSource, error) {
	es := &hotWordDetector{
		eventChan: make(chan *Event),
	}
	es.detector = C.newDetector(
		C.CString(deviceName), C.uint(sampleRate),
		C.CString(modelPath), C.CString(keywordPath), C.float(sensitivity),
		(*C.int32_t)(&es.stopFlag),
	)
	if es.detector == nil {
		return nil, errors.New("Couldn't create detector")
	}

	go es.run()
	return es, nil
}

func (h *hotWordDetector) Name() string {
	return "HotWordDetector"
}

func (h *hotWordDetector) Events() chan *Event {
	return h.eventChan
}

func (h *hotWordDetector) Close() {
	atomic.StoreInt32(&h.stopFlag, 1)
}

func (h *hotWordDetector) run() {
	defer h.destroyDetector()
	defer close(h.eventChan)

	if !C.startDetect(h.detector) {
		return
	}

	for h.notStopped() {
		voice := [voiceBufSize]int16{}
		sampleCount := C.detect(h.detector, (*C.int16_t)(&voice[0]), C.int(voiceBufSize))
		if sampleCount > 0 {
			h.hotWordDetected(voice[0:sampleCount])
		} else {
			if h.notStopped() {
				h.hotWordDetected(nil)
			}
		}
	}
}

func (h *hotWordDetector) notStopped() bool {
	return atomic.LoadInt32(&h.stopFlag) == 0
}

func (h *hotWordDetector) hotWordDetected(samples []int16) {
	h.eventChan <- NewHotWordDetectedEvent(samples)
}

func (h *hotWordDetector) destroyDetector() {
	if h.detector != nil {
		C.destroyDetector(h.detector)
		h.detector = nil
	}
}
