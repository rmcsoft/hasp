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
	int sampleRate;
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
   		if (handle != NULL) {
   			snd_pcm_close(handle);
   			handle = NULL;
   		}
   	}

   	if (params)
   		snd_pcm_hw_params_free(params);

   	return handle;
}

static pv_porcupine_object_t* createPorcupine(const char *modelPath, const char *keywordPath, float sensitivity) {
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

   		free(d);
   	}
}

static Detector* newDetector(
   	const char* deviceName,
   	const char *modelPath, const char *keywordPath,
   	float sensitivity)
{
   	Detector* d = calloc(1, sizeof(Detector));
   	if (d == NULL)
   		return NULL;

   	d->sampleRate = pv_sample_rate();
   	d->capDev = openCaptureDev(deviceName, d->sampleRate);
   	if (d->capDev == NULL)
   		goto error;

   	if (modelPath != NULL && keywordPath != NULL)
   	{
   		d->porcupine = createPorcupine(modelPath, keywordPath, sensitivity);
   		if (d->porcupine == NULL)
   			goto error;
   	}

   	return d;

error:
   	destroyDetector(d);
   	return NULL;
}

static void resetPorcupine(Detector* d) {
	const int bufSize = pv_porcupine_frame_length();
	int16_t buf[bufSize];
	bool detected = false;
	memset(buf, 0, sizeof(int16_t)*bufSize);
	pv_porcupine_process(d->porcupine, buf, &detected);
}

static bool startSession(Detector* d, int32_t* stopFlagPtr) {
   	bool retval = false;
   	int err;

   	err = snd_pcm_prepare(d->capDev);
   	if (err < 0) {
   		fprintf(stderr, "Cannot start soundcard (%s, %d)\n", snd_strerror(err), err);
   		goto out;
   	}

	d->stopFlagPtr = stopFlagPtr;
	resetPorcupine(d);
   	retval = true;

out:
   	return retval;
}

static void sessionClosed(Detector* d) {
   	snd_pcm_drop(d->capDev);
}

static inline bool notStopped(Detector* d) {
   	return *d->stopFlagPtr == 0;
}

static int readSamples(Detector* d, int16_t* buf, int maxSampleCount) {
	int err;
	while (notStopped(d)) {
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
   	return -EINTR;
}

#define DEBUG_VOICE
#define NOISE_THRESHOLD 4000
#define NOISE_FRAMES 30

static short getMaxLoud(const int16_t* samples, int sampleCount) {
   	int16_t max = 0;
   	const int16_t* end = samples + sampleCount;
   	for (; samples != end; ++samples) {
   		int16_t v = *samples;
   		if (v < 0) {
   			v = -v;
   		}
   		if (v > max) {
   			max = v;
   		}
   	}
   	return max;
}

static int waitHotWord(Detector* d) {
   	const int bufSize = pv_porcupine_frame_length();
   	int16_t buf[bufSize];

   	bool detected = false;
   	while (notStopped(d)) {
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

static int soundCapture(Detector* d, int16_t* outputBuffer, int maxSampleCount) {
   	const int bufSize = pv_porcupine_frame_length();
   	int16_t buf[bufSize];

   	int16_t silenseSens = -1;
   	int currentBufferFill = 0;
   	int startSilenceFrames = 0;
   	while (notStopped(d)) {
   		int n = readSamples(d, buf, bufSize);
   		if (n < 0)
   			return n;

   		int16_t maxLoud = getMaxLoud(buf, n);

   		if (n > maxSampleCount - currentBufferFill)
   			n = maxSampleCount - currentBufferFill;
   		if (silenseSens < 0) {
   			if (maxLoud > NOISE_THRESHOLD) {
#ifdef DEBUG_VOICE
   				printf("%d[", maxLoud); fflush(stdout);
#endif

   				silenseSens = NOISE_FRAMES;
   				memcpy(outputBuffer + currentBufferFill, buf, n*sizeof(int16_t));
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
   			memcpy(outputBuffer + currentBufferFill, buf, n*sizeof(int16_t));
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

static int detect(Detector* d, int16_t* buffer, int maxSampleCount) {
   	int err = waitHotWord(d);
   	if (err)
   		return err;
   	return soundCapture(d, buffer, maxSampleCount);
}
*/
import "C"

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rmcsoft/hasp/events"
)

const (
	recTime     int64   = int64(time.Duration(10) * time.Second)
	sensitivity float32 = 0.5
)

// HotWordDetectorParams HotWordDetector params
type HotWordDetectorParams struct {
	CaptureDeviceName string
	ModelPath         string
	KeywordPath       string
}

type hotWordDetectorMode int

const (
	detectHotWordMode hotWordDetectorMode = iota
	soundCaptureMode
)

type hotWordDetectorSession struct {
	owner *HotWordDetector

	mode      hotWordDetectorMode
	stopFlag  int32
	eventChan chan *events.Event
}

// HotWordDetector Implements a hotword detector
type HotWordDetector struct {
	mutex          *sync.Mutex
	detector       *C.Detector
	sessionChan    chan *hotWordDetectorSession
	currentSession *hotWordDetectorSession
}

// NewHotWordDetector creates HotWordDetector
func NewHotWordDetector(params HotWordDetectorParams) (*HotWordDetector, error) {
	d := &HotWordDetector{
		mutex:       &sync.Mutex{},
		sessionChan: make(chan *hotWordDetectorSession),
	}

	d.detector = C.newDetector(
		C.CString(params.CaptureDeviceName),
		C.CString(params.ModelPath), C.CString(params.KeywordPath),
		C.float(sensitivity),
	)
	if d.detector == nil {
		return nil, errors.New("Couldn't create detector")
	}

	go d.run()
	return d, nil
}

// Destroy destroys HotWordDetector
func (d *HotWordDetector) Destroy() {
	close(d.sessionChan)
}

// GetSampleRate gets sample rate
func (d *HotWordDetector) GetSampleRate() int {
	return int(d.detector.sampleRate)
}

// StartDetect starts hotword detection
func (d *HotWordDetector) StartDetect() (events.EventSource, error) {
	return d.startSession(detectHotWordMode)
}

// StartSoundCapture starts capturing sound
func (d *HotWordDetector) StartSoundCapture() (events.EventSource, error) {
	return d.startSession(soundCaptureMode)
}

func (d *HotWordDetector) startSession(mode hotWordDetectorMode) (events.EventSource, error) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.currentSession != nil {
		return nil, errors.New("HotWordDetector busy")
	}

	session := &hotWordDetectorSession{
		owner:     d,
		mode:      mode,
		eventChan: make(chan *events.Event),
	}

	d.currentSession = session
	d.sessionChan <- session
	return session, nil
}

func (d *HotWordDetector) sessionClosed(session *hotWordDetectorSession) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if session != d.currentSession {
		panic("HotWordDetector.sessionClosed: session != d.currentSession")
	}

	d.currentSession = nil
}

func (d *HotWordDetector) run() {
	for session := range d.sessionChan {
		d.runSession(session)
	}
	C.destroyDetector(d.detector)
	d.detector = nil
}

func (d *HotWordDetector) runSession(session *hotWordDetectorSession) {
	defer close(session.eventChan)
	defer C.sessionClosed(d.detector)

	if !C.startSession(d.detector, (*C.int32_t)(&session.stopFlag)) {
		fmt.Fprintf(os.Stderr, "Failed to start a new session of the hotword detector\n")
		return
	}

	for session.notStopped() {
		switch session.mode {
		case detectHotWordMode:
			d.doDetectHotWord(session)
		case soundCaptureMode:
			d.doSoundCapture(session)
		}
	}
}

func (d *HotWordDetector) makeSampleBuf() (buf []int16, maxSampleCount int) {
	maxSampleCount = int(recTime * int64(d.GetSampleRate()) / int64(time.Second))
	buf = make([]int16, maxSampleCount)
	return
}

func (d *HotWordDetector) handleError(session *hotWordDetectorSession, op string, errcode int) {
	if session.notStopped() {
		fmt.Fprintf(os.Stderr, "%s failed: err=%v\n", op, errcode)
	}
}

func (d *HotWordDetector) doDetectHotWord(session *hotWordDetectorSession) {
	// fmt.Println("HotWordDetector.doDetectHotWord")

	buf, maxSampleCount := d.makeSampleBuf()
	sampleCount := C.detect(d.detector, (*C.int16_t)(&buf[0]), C.int(maxSampleCount))
	if sampleCount < 0 {
		d.handleError(session, "HotWordDetect", int(sampleCount))
		return
	}

	samples := buf[0:sampleCount]
	session.eventChan <- events.NewHotWordDetectedEvent(samples, d.GetSampleRate())
}

func (d *HotWordDetector) doSoundCapture(session *hotWordDetectorSession) {
	// fmt.Println("HotWordDetector.doSoundCapture")

	buf, maxSampleCount := d.makeSampleBuf()
	sampleCount := C.soundCapture(d.detector, (*C.int16_t)(&buf[0]), C.int(maxSampleCount))
	if sampleCount < 0 {
		d.handleError(session, "SoundCapture", int(sampleCount))
		return
	}

	samples := buf[0:sampleCount]
	if len(samples) > 0 {
		session.eventChan <- events.NewSoundCapturedEvent(samples, d.GetSampleRate())
	} else {
		session.eventChan <- events.NewSoundEmptyEvent()
	}
}

func (s *hotWordDetectorSession) Name() string {
	return "HotWordDetector"
}

func (s *hotWordDetectorSession) Events() chan *events.Event {
	return s.eventChan
}

func (s *hotWordDetectorSession) Close() {
	atomic.StoreInt32(&s.stopFlag, 1)
	s.owner.sessionClosed(s)
}

func (s *hotWordDetectorSession) notStopped() bool {
	return atomic.LoadInt32(&s.stopFlag) == 0
}
