package main

import (
	"io"
	"io/ioutil"
	"os"
	"strconv"
)

func makeFileName(fileInfo os.FileInfo) string {
	if fileInfo.Size() > 0 {
		return fileInfo.Name() + " (" + strconv.FormatInt(fileInfo.Size(), 10) + "b)"
	} else {
		return fileInfo.Name() + " (empty)"
	}
}

func findNextObject(files []os.FileInfo, i int, path string) (fileInfo os.FileInfo, nextI int, fullPath string, err error) {
	nextI = i

	if i >= len(files) {
		return
	}

	fullPath = path + string(os.PathSeparator) + files[i].Name()
	fileInfo, err = os.Stat(fullPath)
	if err != nil {
		return
	}

	nextI++
	return
}

type findNextObjectFunc = func([]os.FileInfo, int, string) (os.FileInfo, int, string, error)

func findNextDir(files []os.FileInfo, i int, path string) (fileInfo os.FileInfo, nextI int, fullPath string, err error) {
	for i < len(files) {
		fileInfo, nextI, fullPath, err = findNextObject(files, i, path)

		if err != nil {
			return
		}

		if fileInfo.IsDir() {
			for nextI < len(files) {
				var nextFileInfo os.FileInfo
				var nextDir int
				nextFileInfo, nextDir, _, err = findNextObject(files, nextI, path)
				if err != nil {
					return
				}

				if nextFileInfo.IsDir() {
					return
				}

				nextI = nextDir
			}

			return
		}

		i = nextI
	}

	return nil, len(files), "", nil
}

func dirTreeInternal(out io.Writer, path string, printFiles bool, prefix string) error {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}

	var nextFileSystemObject findNextObjectFunc
	if printFiles {
		nextFileSystemObject = findNextObject
	} else {
		nextFileSystemObject = findNextDir
	}

	var i int
	for {
		fileInfo, nextI, fullPath, err := nextFileSystemObject(files, i, path)

		if err != nil {
			return err
		}

		if fileInfo == nil {
			break
		}

		var currentPrefix string
		var nextPrefix string
		if nextI == len(files) {
			currentPrefix = "└───"
			nextPrefix = "\t"
		} else {
			currentPrefix = "├───"
			nextPrefix = "│\t"
		}

		if fileInfo.IsDir() {
			dirName := prefix + currentPrefix + fileInfo.Name() + "\n"
			io.WriteString(out, dirName)

			err = dirTreeInternal(out, fullPath, printFiles, prefix+nextPrefix)
			if err != nil {
				return err
			}
		} else if printFiles {
			fileName := prefix + currentPrefix + makeFileName(fileInfo) + "\n"
			io.WriteString(out, fileName)
		}

		i = nextI
	}

	return nil
}

func dirTree(out io.Writer, path string, printFiles bool) error {
	return dirTreeInternal(out, path, printFiles, "")
}

func main() {
	out := os.Stdout
	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		panic("usage go run main.go . [-f]")
	}
	path := os.Args[1]
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"
	err := dirTree(out, path, printFiles)
	if err != nil {
		panic(err.Error())
	}
}
