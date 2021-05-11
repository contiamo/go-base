package fileutils

import "io/ioutil"

// ListFiles returns a list of the files in the given path, much like
// the bash command `ls`
func ListFiles(path string) (fileNames []string) {
	fileInfo, _ := ioutil.ReadDir(path)
	for _, file := range fileInfo {
		fileNames = append(fileNames, file.Name())
	}
	return fileNames
}
