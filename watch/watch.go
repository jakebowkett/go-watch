package watch

import (
	"errors"
	"io/ioutil"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

const retryDelta = time.Millisecond * 50
const loadRetries = 5

type Options struct {

	// Minimum amount of time between file updates in seconds.
	Delta int

	// Number of times to attempt loading a
	// watched file if it initially fails.
	FileReloads int
}

type File struct {
	Data     []byte
	MinDelta int
}

func load(path string) (data []byte, err error) {

	for i := 1; i <= loadRetries; i++ {

		data, err = ioutil.ReadFile(path)

		if err == nil && len(data) > 0 {
			break
		}

		time.Sleep(retryDelta)
	}

	if err != nil || len(data) == 0 {
		return nil, errors.New("watch: unable to load file")
	}

	return data, nil
}

func This(path string, delta int, callback func(f *File, err error)) {

	path, err := filepath.Abs(path)
	if err != nil {
		callback(nil, err)
		return
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		callback(nil, err)
		return
	}
	defer watcher.Close()

	err = watcher.Add(path)
	if err != nil {
		callback(nil, err)
		return
	}

	f := &File{MinDelta: delta}
	lastUpdate := time.Now()

	for {

		select {

		case event, ok := <-watcher.Events:

			if !ok {
				callback(nil, errors.New("watch: watcher events channel closed"))
				return
			}

			if event.Op != fsnotify.Write {
				continue
			}

			// Events are duplicated on Windows so we disable
			// updates within a short period of time.
			if time.Since(lastUpdate) < time.Second*time.Duration(f.MinDelta) {
				continue
			}

			// Load data from file and update *File struct.
			data, err := load(path)
			if err != nil {
				callback(nil, err)
				return
			}
			f.Data = data
			callback(f, nil)

			lastUpdate = time.Now()

		case err, ok := <-watcher.Errors:

			if !ok {
				callback(nil, errors.New("watch: watcher errors channel closed"))
				return
			}

			if err != nil {
				callback(nil, err)
				return
			}
		}
	}
}
