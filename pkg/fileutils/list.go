package fileutils

import "os"

// ListFiles returns a list of the files in the given path, much like
// the bash command `ls`
func ListFiles(path string) (fileNames []string) {
	fileInfo, _ := os.ReadDir(path)
	for _, file := range fileInfo {
		fileNames = append(fileNames, file.Name())
	}
	return fileNames
}
