/*
Misc util functions
*/
package fixregsto

import (
	"os"
	"strings"
)

//filenameOk tells is name acceptable for operational system
func filenameOk(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil || os.IsNotExist(err)
}

//fileExists tells is file existing
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

//Creates filename-filesize map from all files in dir having specific prefix
func listFileSizes(path string, prefix string) (map[string]int, error) {
	fEntries, errDir := os.ReadDir(path)
	if errDir != nil {
		return nil, errDir
	}
	result := make(map[string]int)
	for _, entry := range fEntries {
		if strings.HasPrefix(entry.Name(), prefix) {
			fInfo, errFinfo := entry.Info()
			if errFinfo != nil {
				return result, errFinfo
			}
			result[fInfo.Name()] = int(fInfo.Size())
		}
	}
	return result, nil
}

//Really paranoidic way of writing file. TODO restore function if rename is failed?
func writeWithFsyncCow(filename string, content []byte) (int, error) {
	f, errOpen := os.OpenFile(filename+"_TMP", os.O_RDWR|os.O_CREATE, 0755)
	if errOpen != nil {
		return 0, errOpen
	}
	n, wErr := f.Write(content) //Write returns non-nil error when n!=len(content)
	if wErr != nil {
		return 0, wErr
	}
	syncErr := f.Sync()
	if syncErr != nil {
		return 0, syncErr
	}
	closeErr := f.Close()
	if closeErr != nil {
		return 0, closeErr
	}
	//rename tmp
	renErr := os.Rename(filename+"_TMP", filename)
	if renErr != nil {
		return 0, renErr
	}

	return n, nil
}
