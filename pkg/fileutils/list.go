package fileutils

import "io/ioutil"

// LS returns a list of the files in the given path, much like
// the bash bommand `ls`
func LS(path string) (fileNames []string) {
	fileInfo, _ := ioutil.ReadDir(path)
	for _, file := range fileInfo {
		fileNames = append(fileNames, file.Name())
	}
	return
}
