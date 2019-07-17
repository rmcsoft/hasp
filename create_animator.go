package hasp

import (
	"fmt"
	"strings"

	"github.com/rmcsoft/chanim"
	"github.com/sirupsen/logrus"
)

func isTransitFrameSeries(frameSeries chanim.FrameSeries) bool {
	// <AnimationName>_entry - transition frames to entry the animation
	if strings.HasSuffix(frameSeries.Name, "_entry") {
		return true
	}

	// <AnimationName>_exit  - transition frames to exit the animation
	if strings.HasSuffix(frameSeries.Name, "_exit") {
		return true
	}

	return false
}

func isAnimationFrameSeries(frameSeries chanim.FrameSeries) bool {
	return !isTransitFrameSeries(frameSeries)
}

func getExitFrameSeries(animation chanim.Animation, allFrameSeries []chanim.FrameSeries) *chanim.FrameSeries {
	exitFrameSeriesName := animation.Name + "_exit"
	for _, frameSeries := range allFrameSeries {
		if frameSeries.Name == exitFrameSeriesName {
			return &frameSeries
		}
	}
	return nil
}

func getEntryFrameSeries(animation chanim.Animation, allFrameSeries []chanim.FrameSeries) *chanim.FrameSeries {
	entryFrameSeriesName := animation.Name + "_entry"
	for _, frameSeries := range allFrameSeries {
		if frameSeries.Name == entryFrameSeriesName {
			return &frameSeries
		}
	}
	return nil
}

func getAnimationFrames(animation chanim.Animation, allFrameSeries []chanim.FrameSeries) []chanim.Frame {
	for _, frameSeries := range allFrameSeries {
		if animation.FrameSeriesName == frameSeries.Name {
			return frameSeries.Frames
		}
	}
	return nil
}

func createAnimations(allFrameSeries []chanim.FrameSeries) (chanim.Animations, error) {
	animations := make(chanim.Animations, 0)
	for _, frameSeries := range allFrameSeries {
		if isAnimationFrameSeries(frameSeries) {
			if len(frameSeries.Frames) == 0 {
				return nil, fmt.Errorf("Animation '%s' has no frame", frameSeries.Name)
			}

			// The name of the animation matches the name of its frame series.
			animation := chanim.Animation{
				Name:            frameSeries.Name,
				FrameSeriesName: frameSeries.Name,
			}
			animations = append(animations, animation)
		}
	}
	return animations, nil
}

func createTransitionBetween(from chanim.Animation, to chanim.Animation, allFrameSeries *[]chanim.FrameSeries) chanim.Transition {
	exitFrameSeries := getExitFrameSeries(from, *allFrameSeries)
	entryFrameSeries := getEntryFrameSeries(to, *allFrameSeries)

	if exitFrameSeries == nil && entryFrameSeries == nil {
		return chanim.Transition{
			DestAnimationName: to.Name,
		}
	}

	if exitFrameSeries != nil && entryFrameSeries == nil {
		return chanim.Transition{
			DestAnimationName: to.Name,
			FrameSeriesName:   exitFrameSeries.Name,
		}
	}

	if exitFrameSeries == nil && entryFrameSeries != nil {
		return chanim.Transition{
			DestAnimationName: to.Name,
			FrameSeriesName:   entryFrameSeries.Name,
		}
	}

	// exitFrameSeries != nil && entryFrameSeries != nil
	// transition requires a new series
	newFrameSeries := chanim.FrameSeries{
		Name:   fmt.Sprintf("%s -> %s", from.Name, to.Name),
		Frames: append(exitFrameSeries.Frames, entryFrameSeries.Frames...),
	}
	*allFrameSeries = append(*allFrameSeries, newFrameSeries)

	return chanim.Transition{
		DestAnimationName: to.Name,
		FrameSeriesName:   newFrameSeries.Name,
	}
}

func makeTransitionsFrom(from chanim.Animation, animations chanim.Animations, allFrameSeries *[]chanim.FrameSeries) []chanim.Transition {
	transitions := make([]chanim.Transition, 0, len(animations)-1)
	for _, to := range animations {
		if from.Name == to.Name {
			continue
		}

		transition := createTransitionBetween(from, to, allFrameSeries)
		transitions = append(transitions, transition)
	}
	return transitions
}

func initTransitionFrames(animations chanim.Animations, allFrameSeries []chanim.FrameSeries) []chanim.FrameSeries {
	for _, animation := range animations {
		animationFrames := getAnimationFrames(animation, allFrameSeries)
		transitions := makeTransitionsFrom(animation, animations, &allFrameSeries)

		// All animations have two transitional frames - the first and the last
		firstTransitFrame := &animationFrames[0]
		firstTransitFrame.Transitions = transitions

		secondTransitFrame := &animationFrames[len(animationFrames)-1]
		secondTransitFrame.Transitions = transitions
	}

	return allFrameSeries
}

// CreateAnimator creates an animator
func CreateAnimator(paintEngine chanim.PaintEngine, frameSeriesPath string) (*chanim.Animator, error) {
	logrus.Debug("Loading frames")
	allFrameSeries, err := LoadFrameSeries(frameSeriesPath)
	if err != nil {
		return nil, err
	}

	logrus.Debug("Creating animations")
	animations, err := createAnimations(allFrameSeries)
	if err != nil {
		return nil, err
	}

	logrus.Debug("Initializing transition frames")
	allFrameSeries = initTransitionFrames(animations, allFrameSeries)

	logrus.Debug("Making animator")
	return chanim.NewAnimator(paintEngine, animations, allFrameSeries)
}
