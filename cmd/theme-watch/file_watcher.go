package main

import (
	"fmt"
	"github.com/csaunders/phoenix"
	"gopkg.in/fsnotify.v1"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

var assetLocations = []string{"templates/customers", "assets", "config", "layout", "snippets", "templates"}

type FsAssetEvent struct {
	asset     phoenix.Asset
	eventType phoenix.EventType
}

type FileReader func(filename string) ([]byte, error)

var WatcherFileReader FileReader = ioutil.ReadFile

func RestoreReader() {
	WatcherFileReader = ioutil.ReadFile
}

func (f FsAssetEvent) Asset() phoenix.Asset {
	return f.asset
}

func (f FsAssetEvent) Type() phoenix.EventType {
	return f.eventType
}

func NewFileWatcher(dir string, recur bool) (processor chan phoenix.AssetEvent) {
	if recur {
		processor, _ = watchDirRecur(dir)
	} else {
		processor, _ = watchDir(dir)
	}
	return
}

func LoadAsset(event fsnotify.Event) phoenix.Asset {
	contents, _ := WatcherFileReader(event.Name)
	return phoenix.Asset{Key: extractAssetKey(event.Name), Value: string(contents)}
}

func HandleEvent(event fsnotify.Event) FsAssetEvent {
	var eventType phoenix.EventType
	asset := LoadAsset(event)
	switch event.Op {
	case fsnotify.Create, fsnotify.Chmod:
		eventType = phoenix.Update
	case fsnotify.Remove:
		eventType = phoenix.Remove
	}
	return FsAssetEvent{asset: asset, eventType: eventType}
}

func extractAssetKey(filename string) string {
	for _, dir := range assetLocations {
		split := strings.SplitAfterN(filename, dir, 2)
		if len(split) > 1 {
			return fmt.Sprintf("%s%s", dir, split[len(split)-1])
		}
	}
	return ""
}

func convertFsEvents(events chan fsnotify.Event) chan phoenix.AssetEvent {
	results := make(chan phoenix.AssetEvent)
	go func() {
		for {
			results <- HandleEvent(<-events)
		}
	}()
	return results
}

func watchDirRecur(dir string) (results chan phoenix.AssetEvent, err error) {
	results = make(chan phoenix.AssetEvent)
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			channel, _ := watchDir(path)
			go func() {
				for {
					results <- <-channel
				}
			}()
		}
		return err
	})
	return
}

func watchDir(dir string) (results chan phoenix.AssetEvent, err error) {
	watcher, err := fsnotify.NewWatcher()
	err = watcher.Add(dir)
	if err != nil {
		results = make(chan phoenix.AssetEvent)
		close(results)
	} else {
		results = convertFsEvents(watcher.Events)
	}
	return
}