package watch

import (
	"errors"
	"io/ioutil"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

type File struct {
	Data     []byte
	MinDelta int
}

func load(path string) ([]byte, error) {

	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	f, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return f, nil
}

func This(path string, delta int, callback func(f *File, err error)) {

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
				callback(nil, errors.New("watcher events channel closed"))
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
				callback(nil, errors.New("watcher errors channel closed"))
				return
			}

			if err != nil {
				callback(nil, err)
				return
			}
		}
	}
}
