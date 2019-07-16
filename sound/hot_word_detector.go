package sound

/*
#cgo pkg-config: alsa

// Porcupine
#cgo CFLAGS: -I${SRCDIR}/../../Porcupine/include
#cgo linux,amd64 LDFLAGS: -L${SRCDIR}/../../Porcupine/lib/linux/x86_64
#cgo linux,arm   LDFLAGS: -L${SRCDIR}/../../Porcupine/lib/beaglebone
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

#define ESTRN 256
typedef char EStr[ESTRN];

#define eprintf(fromat...) snprintf(*estr, ESTRN, fromat)

static snd_pcm_t* openCaptureDev(const char* deviceName, unsigned int rate, EStr* estr) {
   	int err;
   	snd_pcm_t* handle = NULL;

   	if ((err = snd_pcm_open(&handle, deviceName, SND_PCM_STREAM_CAPTURE, 0)) < 0) {
   		eprintf("Cannot open capture audio device %s (%s, %d)", deviceName, snd_strerror(err), err);
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

   	return handle;
}

static pv_porcupine_object_t* createPorcupine(const char *modelPath, const char *keywordPath, float sensitivity, EStr* estr) {
   	pv_porcupine_object_t* porcupine = NULL;
   	pv_status_t status = pv_porcupine_init(modelPath, keywordPath, sensitivity, &porcupine);
   	if (status != PV_STATUS_SUCCESS) {
   		eprintf("Failed to initialize Porcupine");
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
	float sensitivity,
	EStr* estr)
{
   	Detector* d = calloc(1, sizeof(Detector));
   	if (d == NULL) {
		eprintf("Unable to alloc memmory for Detector");
		return NULL;
	}

   	d->sampleRate = pv_sample_rate();
   	d->capDev = openCaptureDev(deviceName, d->sampleRate, estr);
   	if (d->capDev == NULL)
   		goto error;

   	if (modelPath != NULL && keywordPath != NULL)
   	{
   		d->porcupine = createPorcupine(modelPath, keywordPath, sensitivity, estr);
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

static bool startSession(Detector* d, int32_t* stopFlagPtr, EStr* estr) {
   	bool retval = false;
  	snd_pcm_hw_params_t* params = NULL;
	int rate = d->sampleRate;
   	int err;

   	if ((err = snd_pcm_hw_params_malloc(&params)) < 0) {
   		eprintf("Cannot allocate hardware parameter structure (%s, %d)", snd_strerror(err), err);
   		goto out;
   	}

   	if ((err = snd_pcm_hw_params_any(d->capDev, params)) < 0) {
   		eprintf("Cannot initialize hardware parameter structure (%s, %d)", snd_strerror(err), err);
   		goto out;
   	}

   	if ((err = snd_pcm_hw_params_set_access(d->capDev, params, SND_PCM_ACCESS_RW_INTERLEAVED)) < 0) {
   		eprintf("Cannot set access type (%s, %d)", snd_strerror(err), err);
   		goto out;
   	}

   	if ((err = snd_pcm_hw_params_set_format(d->capDev, params,SND_PCM_FORMAT_S16_LE)) < 0) {
   		eprintf("Cannot set sample format (%s, %d)", snd_strerror(err), err);
   		goto out;
   	}

   	if ((err = snd_pcm_hw_params_set_rate_near(d->capDev, params, &rate, 0)) < 0) {
   		eprintf("Cannot set sample rate (%s, %d)", snd_strerror(err), err);
   		goto out;
   	}

   	if ((err = snd_pcm_hw_params_set_channels(d->capDev, params, 1)) < 0) {
   		eprintf("Cannot set channel count (%s, %d)", snd_strerror(err), err);
   		goto out;
   	}

   	if ((err = snd_pcm_hw_params(d->capDev, params)) < 0) {
   		eprintf("Cannot set parameters (%s, %d)", snd_strerror(err), err);
   		goto out;
   	}

	d->stopFlagPtr = stopFlagPtr;
	resetPorcupine(d);
   	retval = true;

out:
   	if (params)
   		snd_pcm_hw_params_free(params);

   	return retval;
}

static void sessionClosed(Detector* d) {
   	snd_pcm_drop(d->capDev);
}

static inline bool notStopped(Detector* d) {
   	return *d->stopFlagPtr == 0;
}

static int readSamples(Detector* d, int16_t* buf, int maxSampleCount, EStr* estr) {
	int err;
	while (notStopped(d)) {
   		int n = snd_pcm_readi(d->capDev, buf, maxSampleCount);
   		if (n == 0)
   			continue;

   		if (n > 0) {
   			return n;
   		}

   		err = n;
   		eprintf("Read from audio interface failed (%s, %d)", snd_strerror(err), err);
   		if (err != -32)
   			return err;

   		// Broken pipe
   		if ((err = snd_pcm_prepare(d->capDev)) < 0) {
   			eprintf("Cannot prepare audio interface for use (%s, %d)", snd_strerror(err), err);
   			return err;
   		}
   	}
   	return -EINTR;
}

#define DEBUG_VOICE
#define NOISE_THRESHOLD 5000
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

static int waitHotWord(Detector* d, EStr* estr) {
   	const int bufSize = pv_porcupine_frame_length();
   	int16_t buf[bufSize];

   	bool detected = false;
   	while (notStopped(d)) {
   		int n = readSamples(d, buf, bufSize, estr);
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

static int soundCapture(Detector* d, int16_t* outputBuffer, int maxSampleCount, EStr* estr) {
   	const int bufSize = pv_porcupine_frame_length();
   	int16_t buf[bufSize];

   	int16_t silenseSens = -1;
   	int currentBufferFill = 0;
   	int startSilenceFrames = 0;
   	while (notStopped(d)) {
   		int n = readSamples(d, buf, bufSize, estr);
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

static int detect(Detector* d, int16_t* buffer, int maxSampleCount, EStr* estr) {
   	int err = waitHotWord(d, estr);
   	if (err)
   		return err;
   	return soundCapture(d, buffer, maxSampleCount, estr);
}
*/
import "C"

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	log "github.com/sirupsen/logrus"

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
	originState    string
}

func (estr *C.EStr) String() string {
	return C.GoString((*C.char)(unsafe.Pointer(estr)))
}

// NewHotWordDetector creates HotWordDetector
func NewHotWordDetector(params HotWordDetectorParams) (*HotWordDetector, error) {
	d := &HotWordDetector{
		mutex:       &sync.Mutex{},
		sessionChan: make(chan *hotWordDetectorSession),
	}

	var estr C.EStr
	d.detector = C.newDetector(
		C.CString(params.CaptureDeviceName),
		C.CString(params.ModelPath), C.CString(params.KeywordPath),
		C.float(sensitivity),
		&estr,
	)
	if d.detector == nil {
		err := fmt.Errorf("Couldn't create detector: %s", &estr)
		return nil, err
	}

	go d.run()
	return d, nil
}

// Destroy destroys HotWordDetector
func (d *HotWordDetector) Destroy() {
	close(d.sessionChan)
}

// SampleRate gets sample rate
func (d *HotWordDetector) SampleRate() int {
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

	log.Info("HotWordDetector: start session")
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
		log.Panic("HotWordDetector.sessionClosed: session != d.currentSession")
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

	var estr C.EStr
	if !C.startSession(d.detector, (*C.int32_t)(&session.stopFlag), &estr) {
		// TODO:  Reaction to an error
		err := fmt.Errorf("Failed to start a new session of the hotword detector: %v", &estr)
		log.Errorf("HotWordDetector: %v", err)
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

	log.Info("HotWordDetector: session closed")
}

func (d *HotWordDetector) makeSampleBuf() (buf []byte, cptr *C.int16_t, maxSampleCount int) {
	maxSampleCount = int(recTime * int64(d.SampleRate()) / int64(time.Second))
	buf = make([]byte, maxSampleCount*S16LE.Size())
	cptr = (*C.int16_t)(unsafe.Pointer(&buf[0]))
	return
}

func (d *HotWordDetector) makeAudioData(buf []byte, sampleCount C.int) *AudioData {
	sizeInBytes := int(sampleCount) * S16LE.Size()
	return NewMonoS16LE(d.SampleRate(), buf[0:sizeInBytes])
}

func (d *HotWordDetector) handleError(session *hotWordDetectorSession, op string, estr *C.EStr) {
	if session.notStopped() {
		// TODO:  Reaction to an error
		err := fmt.Errorf("%s failed: %v", op, estr)
		log.Errorf("HotWordDetector: %v", err)
	}
}

func (d *HotWordDetector) doDetectHotWord(session *hotWordDetectorSession) {
	buf, cptr, maxSampleCount := d.makeSampleBuf()
	var estr C.EStr
	sampleCount := C.detect(d.detector, cptr, C.int(maxSampleCount), &estr)
	if sampleCount < 0 {
		d.handleError(session, "HotWordDetect", &estr)
		return
	}

	session.eventChan <- NewHotWordDetectedEvent(d.makeAudioData(buf, sampleCount))
}

func (d *HotWordDetector) doSoundCapture(session *hotWordDetectorSession) {
	buf, cptr, maxSampleCount := d.makeSampleBuf()
	var estr C.EStr
	sampleCount := C.soundCapture(d.detector, cptr, C.int(maxSampleCount), &estr)
	if sampleCount < 0 {
		d.handleError(session, "SoundCapture", &estr)
		return
	}

	samples := buf[0:sampleCount]
	if len(samples) > 0 {
		session.eventChan <- NewSoundCapturedEvent(d.makeAudioData(buf, sampleCount))
	} else {
		session.eventChan <- NewSoundEmptyEvent()
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
