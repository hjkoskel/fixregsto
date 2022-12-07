/*
Misc util functions
*/
package fixregsto

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"strings"
)

const COMPRESSIONMETHOD_GZ = "gz" //TODO other compression methods?

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

func readCompressedFile(filename string, method string, bitslices []int) ([]byte, error) {
	if len(method) == 0 {
		content, readErr := os.ReadFile(filename)
		if readErr != nil {
			return nil, readErr
		}
		return unsliceBitArr(content, bitslices)
	}
	if method != COMPRESSIONMETHOD_GZ {
		return nil, fmt.Errorf("only gz is supported not, %s", method)
	}

	f, fErr := os.Open(filename)
	if fErr != nil {
		return nil, fErr
	}

	zr, zrErr := gzip.NewReader(f)
	if zrErr != nil {
		return nil, zrErr
	}

	result, readErr := io.ReadAll(zr)
	if readErr != nil {
		return nil, readErr
	}

	zcloseErr := zr.Close()
	if zcloseErr != nil {
		return nil, zcloseErr
	}

	return unsliceBitArr(result, bitslices)
}

//Paranoidic way to write file with compression. And slice and re-arrange bits for better compression
func writeWithFsyncCowCompressed(filename string, contentOriginal []byte, method string, bitslices []int) (int, error) {
	content, slicingError := sliceBitArr(contentOriginal, bitslices)
	if slicingError != nil {
		return 0, slicingError
	}

	if len(method) == 0 {
		return writeWithFsyncCow(filename, content)
	}
	if method != COMPRESSIONMETHOD_GZ {
		return 0, fmt.Errorf("only gz is supported not, %s", method)
	}

	f, errOpen := os.OpenFile(filename+"_TMP", os.O_RDWR|os.O_CREATE, 0755)
	if errOpen != nil {
		return 0, errOpen
	}

	zw := gzip.NewWriter(f)
	n, wErr := zw.Write(content)
	if wErr != nil {
		return 0, wErr
	}
	gzCloseErr := zw.Close()
	if gzCloseErr != nil {
		return 0, fmt.Errorf("gz compression err %v", gzCloseErr.Error())
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

	//Internal runtime testing, remove later for better performance. Used early to detect issues IF system produces invalid files and important data is lost
	refContent, refReadErr := readCompressedFile(filename, method, bitslices)
	if refReadErr != nil {
		return n, fmt.Errorf("error reading back file %v, err=%v", filename, refReadErr)
	}
	if !bytes.Equal(contentOriginal, refContent) {
		return n, fmt.Errorf("read back does not match on file %v", filename)
	}
	return n, nil
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
