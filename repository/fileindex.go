// Copyright 2013 Andreas Koch. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package repository

import (
	"fmt"
	p "github.com/andreaskoch/allmark/path"
	"github.com/howeyc/fsnotify"
	"io/ioutil"
	"path/filepath"
	"strings"
)

func NewFileIndex(directory string) *FileIndex {

	getFilesFunc := func() []*File {
		files := getFiles(directory)
		return files
	}

	fileIndex := &FileIndex{
		Items: getFilesFunc,

		path: directory,
	}

	fileIndex.RegisterOnChangeCallback("ReindexOnFilesystemEvent", func(fileIndex *FileIndex) {
		fileIndex.Items = getFilesFunc
	})

	return fileIndex
}

type FileIndex struct {
	Items func() []*File

	path               string
	onChangeCallbacks  map[string]func(fileIndex *FileIndex)
	itemIsBeingWatched bool
}

func (fileIndex *FileIndex) String() string {
	return fmt.Sprintf("%s", fileIndex.path)
}

func (fileIndex *FileIndex) Path() string {
	return fileIndex.path
}

func (fileIndex *FileIndex) Directory() string {
	return fileIndex.Path()
}

func (fileIndex *FileIndex) PathType() string {
	return p.PatherTypeIndex
}

func (fileIndex *FileIndex) GetFilesByPath(path string, condition func(pather p.Pather) bool) []*File {

	// normalize path
	path = strings.Replace(path, p.UrlDirectorySeperator, p.FilesystemDirectorySeperator, -1)
	path = strings.Trim(path, p.FilesystemDirectorySeperator)

	// make path relative
	if strings.Index(path, FilesDirectoryName) == 0 {
		path = path[len(FilesDirectoryName):]
	}

	matchingFiles := make([]*File, 0)

	for _, file := range fileIndex.Items() {

		filePath := file.Path()
		indexPath := fileIndex.Path()

		if strings.Index(filePath, indexPath) != 0 {
			continue
		}

		relativeFilePath := filePath[len(indexPath):]
		fileMatchesPath := strings.HasPrefix(relativeFilePath, path)
		if fileMatchesPath && condition(file) {
			matchingFiles = append(matchingFiles, file)
		}
	}

	return matchingFiles
}

func (fileIndex *FileIndex) RegisterOnChangeCallback(name string, callbackFunction func(fileIndex *FileIndex)) {

	if fileIndex.onChangeCallbacks == nil {
		// initialize on first use
		fileIndex.onChangeCallbacks = make(map[string]func(fileIndex *FileIndex))

		// start watching for changes
		fileIndex.startWatch()
	}

	if _, ok := fileIndex.onChangeCallbacks[name]; ok {
		fmt.Printf("Change callback %q already present.", name)
	}

	fileIndex.onChangeCallbacks[name] = callbackFunction
}

func (fileIndex *FileIndex) pauseWatch() {
	fmt.Printf("Pausing watch on fileIndex %s\n", fileIndex)
	fileIndex.itemIsBeingWatched = false
}

func (fileIndex *FileIndex) watchIsPaused() bool {
	return fileIndex.itemIsBeingWatched == false
}

func (fileIndex *FileIndex) resumeWatch() {
	fmt.Printf("Resuming watch on file index %s\n", fileIndex)
	fileIndex.itemIsBeingWatched = true
}

func (fileIndex *FileIndex) startWatch() *FileIndex {

	fmt.Printf("Starting watch on fileIndex %s\n", fileIndex)
	fileIndex.itemIsBeingWatched = true

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Printf("Error while creating watch for file index %q. Error: %v", fileIndex, err)
		return fileIndex
	}

	go func() {
		for {
			select {
			case event := <-watcher.Event:

				if !fileIndex.watchIsPaused() {
					fmt.Println("File index changed ->", event)

					for name, callback := range fileIndex.onChangeCallbacks {
						fmt.Printf("File index changed. Executing callback %q on for file index %q\n", name, fileIndex)
						callback(fileIndex)
					}
				}

			case err := <-watcher.Error:
				fmt.Printf("Watch error on file index %q. Error: %v\n", fileIndex, err)
			}
		}
	}()

	err = watcher.Watch(fileIndex.Directory())
	if err != nil {
		fmt.Printf("Error while creating watch for folder %q. Error: %v\n", fileIndex.Directory(), err)
	}

	return fileIndex
}

func getFiles(directory string) []*File {

	filesDirectoryEntries, err := ioutil.ReadDir(directory)
	if err != nil {
		fmt.Printf("Cannot read files from directory %q. Error: %s", directory, err)
		return make([]*File, 0)
	}

	files := make([]*File, 0, 5)
	for _, directoryEntry := range filesDirectoryEntries {

		// recurse
		if directoryEntry.IsDir() {
			subDirectory := filepath.Join(directory, directoryEntry.Name())
			files = append(files, getFiles(subDirectory)...)
			continue
		}

		// append new file
		filePath := filepath.Join(directory, directoryEntry.Name())
		files = append(files, NewFile(filePath))
	}

	return files
}
