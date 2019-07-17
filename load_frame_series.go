package hasp

import (
	"image"
	"io/ioutil"
	"path/filepath"
	"sort"

	"github.com/rmcsoft/chanim"
)

func loadFrames(path string) ([]chanim.Frame, error) {
	ppixmapFiles, err := filepath.Glob(filepath.Join(path, "*.ppixmap"))
	if err != nil {
		return nil, err
	}
	sort.Strings(ppixmapFiles)

	frames := make([]chanim.Frame, 0, len(ppixmapFiles))
	for _, ppixmapFile := range ppixmapFiles {
		ppixmap, err := chanim.MMapPackedPixmap(ppixmapFile)
		if err != nil {
			return nil, err
		}

		drawOperation := chanim.NewDrawPackedPixmapOperation(image.Point{0, 0}, ppixmap)
		frames = append(frames, chanim.Frame{
			DrawOperations: []chanim.DrawOperation{drawOperation},
		})
	}

	return frames, nil
}

// LoadFrameSeries loads frame series
// It is assumed that each series is located in its own subdirectory.
// The name of the series is the same as the subdirectory name.
func LoadFrameSeries(path string) ([]chanim.FrameSeries, error) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}

	allFrameSeries := make([]chanim.FrameSeries, 0, len(files))
	for _, fileInfo := range files {
		if fileInfo.IsDir() {
			frameSeriesName := fileInfo.Name()
			frames, err := loadFrames(filepath.Join(path, fileInfo.Name()))
			if err != nil {
				return nil, err
			}

			allFrameSeries = append(allFrameSeries, chanim.FrameSeries{
				Name:   frameSeriesName,
				Frames: frames,
			})
		}
	}

	return allFrameSeries, nil
}
