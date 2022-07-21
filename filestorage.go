//Simple file based Fixed size Record Storage implementation, syncronized copy on write+ filesync
package fixregsto

import (
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

//FileStorageConf tells what kind of FileStorage instance is going to be created
//Name, is prefix for filename. Remember not to use same name (and path) in other databases or files
//RecordSize, is what your application needs (size of struct in bytes?)
//MaxFileCount, how many storage files exists on disk (plus "work file" without number)
//FileMaxSize, how many bytes one file takes up. Prefer some multiple of minimum file size on filesystem
//Path, path to directory where data is stored
type FileStorageConf struct {
	Name         string //Numbering _0, _1,_2 etc..
	RecordSize   int64  //One entry is this long
	MaxFileCount int64  //TODO if 0? no at least 1

	FileMaxSize int64 //How many bytes. Prefer multiple of 512 (erase blocks size optimal)
	Path        string
}

//FileStorage, includes conf and cached data
type FileStorage struct {
	conf FileStorageConf

	workBuffer     []byte //Latest
	readPosition   int64  //record counter
	recordsPerFile int64  //pre calculated values
}

//CheckErrors tell is there problems with configuration
func (p *FileStorageConf) CheckErrors() error {
	if !filenameOk(p.Name) {
		return fmt.Errorf("Invalid name %s", p.Name)
	}
	if p.RecordSize < 1 {
		return fmt.Errorf("Invalid RecordSize %v", p.RecordSize)
	}
	if p.FileMaxSize < 1 { //There is no point really to use file size under 512bytes
		return fmt.Errorf("Invalid FileMaxSize %v", p.FileMaxSize)
	}
	if p.FileMaxSize < p.RecordSize {
		return fmt.Errorf("MaxFileSize(%v) < RecordSize(%v)", p.FileMaxSize, p.RecordSize)
	}
	return nil
}

func (p *FileStorageConf) BaseFileName() string {
	return path.Join(p.Path, p.Name)
}

//InitFileStorage, Call this method after creating FileStorageConf.
//This creates dir if required
func (p *FileStorageConf) InitFileStorage() (FileStorage, error) {
	result := FileStorage{conf: *p}

	errMkdir := os.MkdirAll(p.Path, os.ModePerm)
	if errMkdir != nil {
		return result, fmt.Errorf("Error creating dir %v  err=%v", errMkdir.Error(), errMkdir)
	}
	//Read to work buffer
	workfile := p.BaseFileName()
	if fileExists(workfile) {
		var errRead error
		result.workBuffer, errRead = ioutil.ReadFile(workfile)
		if errRead != nil {
			return result, fmt.Errorf("Error reading %v err=%v", workfile, errRead.Error())
		}
	}
	_, fixPointerErr := result.Seek(0, io.SeekStart)
	if fixPointerErr != nil {
		return result, fmt.Errorf("Reset read failed in init err=%v", fixPointerErr.Error())
	}

	result.recordsPerFile = p.FileMaxSize / p.RecordSize

	return result, nil
}

//getNumberRangeOnDisk, private function gets minimum and maximum number in storage files and count of files  (count is important if missing files in between? Also decides is delete needed)
func (p *FileStorage) getNumberRangeOnDisk() (int64, int64, int64, error) {
	count := int64(0)
	fileinfos, errDir := ioutil.ReadDir(filepath.Dir(p.conf.BaseFileName()))
	if errDir != nil {
		return 0, 0, 0, errDir
	}
	minresult := int64(math.MaxInt64)
	maxresult := int64(-1)
	//-1 not found
	prefix := p.conf.Name + "_"
	for _, finfo := range fileinfos {
		name := finfo.Name()
		if !strings.HasPrefix(name, prefix) || finfo.IsDir() {
			continue //It is not with correct name
		}
		count++
		sNumber := strings.Replace(name, prefix, "", 1)
		n, parseErr := strconv.ParseInt(sNumber, 10, 64)
		if parseErr == nil {
			if maxresult < n && 0 < finfo.Size() {
				maxresult = n
			}
			if n < minresult {
				minresult = n
			}
		}
	}
	if maxresult == -1 {
		return -1, -1, count, nil
	}

	return minresult, maxresult, count, nil
}

//gets filename for filestorage
func (p *FileStorage) filename(n int64) string {
	return fmt.Sprintf("%s_%v", p.conf.BaseFileName(), n)
}

//Write implements writer interface. Only complete records are accepted
func (p *FileStorage) Write(raw []byte) (n int, err error) { //size must be recordsize*N
	if len(raw)%int(p.conf.RecordSize) != 0 {
		return 0, fmt.Errorf("Appended data length %v is not multiple of %v", len(raw), p.conf.RecordSize)
	}

	newRecordCount := int64(len(raw)) / p.conf.RecordSize
	recordsInWork := int64(len(p.workBuffer)) / p.conf.RecordSize
	recordsFreeInWork := p.recordsPerFile - recordsInWork

	//Easy case, just update work buffer and sync that to disk
	if newRecordCount < recordsFreeInWork { //Not need yet to rename work
		p.workBuffer = append(p.workBuffer, raw...) //Ok, fill up work buffer
		_, wErr := writeWithFsyncCow(p.conf.BaseFileName(), p.workBuffer)
		if wErr != nil {
			return 0, wErr
		}
		return len(raw), nil
	}

	originalTotal := len(raw)

	//Put what is required, fill completely up and write to target file
	newPiece := raw[0 : recordsFreeInWork*p.conf.RecordSize]
	p.workBuffer = append(p.workBuffer, newPiece...)
	raw = raw[recordsFreeInWork*p.conf.RecordSize:]

	minFileNumber, maxFileNumber, filecount, errRange := p.getNumberRangeOnDisk()
	if errRange != nil {
		return 0, fmt.Errorf("FileStorage Write erro gettin number range err=%w", errRange)
	}

	_, wErr := writeWithFsyncCow(p.filename(maxFileNumber+1), p.workBuffer)
	if wErr != nil {
		return 0, wErr
	}

	if p.conf.MaxFileCount <= filecount {
		oldFileName := p.filename(minFileNumber)
		removeErr := os.Remove(oldFileName)
		if removeErr != nil {
			return 0, fmt.Errorf("Error removing file on FileStorage Write err=%v  conf.maxFileCount=%v, maxFileNumber=%v minFileNumber=%v", removeErr.Error(), p.conf.MaxFileCount, minFileNumber, maxFileNumber)
		}
	}

	maxFileNumber++
	p.workBuffer = []byte{}
	os.Remove(p.conf.BaseFileName()) //Remove that there would not be wrong material if fail

	//Check is there need to write multiple files completely
	bytesPerFile := p.recordsPerFile * p.conf.RecordSize
	for int(bytesPerFile) <= len(raw) { //While there is data for enough for complete files
		minFileNumber, maxFileNumber, filecount, errRange := p.getNumberRangeOnDisk()
		if errRange != nil {
			return 0, fmt.Errorf("FileStorage Write erro gettin number range err=%w", errRange)
		}

		_, wErr := writeWithFsyncCow(p.filename(maxFileNumber+1), raw[0:bytesPerFile])
		if wErr != nil {
			return 0, wErr
		}
		raw = raw[bytesPerFile:]
		if p.conf.MaxFileCount <= filecount {
			oldFileName := p.filename(minFileNumber)
			removeErr := os.Remove(oldFileName)
			if removeErr != nil {
				return 0, fmt.Errorf("Error removing file on FileStorage Write err=%v  conf.maxFileCount=%v, maxFileNumber=%v minFileNumber=%v", removeErr.Error(), p.conf.MaxFileCount, minFileNumber, maxFileNumber)
			}
		}
	}

	p.workBuffer = raw //let this be work buffer

	//Write work file
	basefilename := p.conf.BaseFileName() //Known as work file
	_, wErr = writeWithFsyncCow(basefilename, p.workBuffer)
	if wErr != nil {
		return 0, wErr
	}
	return originalTotal, nil
}

//Len returns how many records are stored
func (p *FileStorage) Len() (int64, error) {
	bytecount := int64(len(p.workBuffer))
	minFileNumber, maxFileNumber, _, errRange := p.getNumberRangeOnDisk()
	if errRange != nil {
		return 0, errRange
	}

	sizemap, errListFileSizes := listFileSizes(p.conf.Path, p.conf.Name)
	if errListFileSizes != nil {
		return 0, errListFileSizes
	}

	for i := minFileNumber; i <= maxFileNumber; i++ {
		filesize, haz := sizemap[fmt.Sprintf("%s_%v", p.conf.Name, i)]
		if haz {
			bytecount += int64(filesize)
		}
	}
	return bytecount / p.conf.RecordSize, nil
}

func (p *FileStorage) GetLatest(nRecords int64) ([]byte, error) {
	if nRecords < 1 {
		return nil, fmt.Errorf("wrong parameter nRecords=%v", nRecords)
	}
	minFileNumber, maxFileNumber, filecount, errRange := p.getNumberRangeOnDisk()
	if errRange != nil {
		return nil, errRange
	}
	targetSize := nRecords * p.conf.RecordSize
	result := append([]byte{}, p.workBuffer...)
	if 0 < filecount {
		for fileNumber := maxFileNumber; minFileNumber <= fileNumber; fileNumber-- {
			byt, errRead := ioutil.ReadFile(p.filename(fileNumber))
			if errRead != nil {
				return result, errRead
			}
			result = append(byt, result...)
			if int64(len(result)) <= targetSize {
				break
			}
		}
	}

	if int64(len(result)) <= targetSize {
		return result, nil
	}
	return result[int64(len(result))-targetSize:], nil
}

//GetFirst nRecords without moving seek cursor
func (p *FileStorage) GetFirst(nRecords int64) ([]byte, error) {
	if nRecords < 1 {
		return nil, fmt.Errorf("wrong parameter nRecords=%v", nRecords)
	}
	minFileNumber, maxFileNumber, filecount, errRange := p.getNumberRangeOnDisk()
	if errRange != nil {
		return nil, errRange
	}
	targetSize := nRecords * p.conf.RecordSize
	result := []byte{}
	if 0 < filecount {
		for fileNumber := minFileNumber; fileNumber <= maxFileNumber; fileNumber++ {
			byt, errRead := ioutil.ReadFile(p.filename(fileNumber))
			if errRead != nil {
				return result, errRead
			}
			result = append(result, byt...)
			if targetSize <= int64(len(result)) {
				break
			}
		}
	}
	result = append(result, p.workBuffer...)

	if int64(len(result)) <= targetSize {
		return result, nil
	}
	return result[0:targetSize], nil
}

//Read implements Reader interface.  Except only array length must be multiple of recordsize for normal operation
func (p *FileStorage) Read(arr []byte) (n int, err error) {
	minFileNumber, maxFileNumber, filecount, errRange := p.getNumberRangeOnDisk()
	if errRange != nil {
		return 0, errRange
	}
	fileNumber := p.readPosition / p.recordsPerFile

	if fileNumber < int64(minFileNumber) { //If already dropped
		p.readPosition = int64(minFileNumber) * int64(p.recordsPerFile)
		fileNumber = minFileNumber
	}

	recordsNeeded := len(arr) / int(p.conf.RecordSize)             //rounded down
	bytesNeeded := int64(p.conf.RecordSize) * int64(recordsNeeded) // return n is this or lower

	crudeBuf := []byte{}                                                                //Crude way but minimize bugs first, then make unit tests and then optimize memory usage and speed
	startIndex := (p.readPosition % int64(p.recordsPerFile)) * int64(p.conf.RecordSize) //Inside file or this crude buf
	//Pick file numbers that are available, skip others. Break when enough
	if 0 < filecount {
		for ; fileNumber <= maxFileNumber; fileNumber++ {
			fname := p.filename(fileNumber)
			if fileExists(fname) {
				byt, errRead := ioutil.ReadFile(fname)
				if errRead != nil {
					return 0, errRead
				}
				crudeBuf = append(crudeBuf, byt...)
			}
			if int(bytesNeeded) <= len(crudeBuf)-int(startIndex) { //ok, got enough
				resultPiece := crudeBuf[startIndex : startIndex+bytesNeeded]
				if len(resultPiece) != int(bytesNeeded) {
					return 0, fmt.Errorf("INTERNAL ERR!!! vääräpalakoko %v vs %v", len(resultPiece), int(bytesNeeded))
				}
				nCopied := copy(arr[0:], resultPiece)
				if nCopied != int(bytesNeeded) {
					return 0, fmt.Errorf("Copied wrong number (%v) asked %v", nCopied, bytesNeeded)
				}
				if len(resultPiece) == 0 {
					return 0, io.EOF
				}
				p.readPosition += int64(len(resultPiece) / int(p.conf.RecordSize))
				return len(resultPiece), nil
			}
		}
	}
	//Append what is is work buffer
	crudeBuf = append(crudeBuf, p.workBuffer...)
	endIndex := startIndex + bytesNeeded
	resultPiece := []byte{}
	if int(endIndex) < len(crudeBuf) {
		resultPiece = crudeBuf[startIndex:endIndex]
	} else {
		resultPiece = crudeBuf[startIndex:]
	}
	nCopied := copy(arr[0:len(resultPiece)], resultPiece)
	if nCopied != len(resultPiece) {
		return 0, fmt.Errorf("Copied wrong number of bytes %v asked %v", nCopied, len(resultPiece))
	}
	if len(resultPiece) == 0 {
		return 0, io.EOF
	}
	p.readPosition += int64(len(resultPiece) / int(p.conf.RecordSize))
	return len(resultPiece), nil
}

//Seeks, For implementing seeker interface
//Seeks file with byte by byte but rounds up new position where record starts (or ends)
func (p *FileStorage) Seek(offset int64, whence int) (int64, error) {
	minFileNumber, maxFileNumber, filecount, errRange := p.getNumberRangeOnDisk()
	if errRange != nil {
		return 0, errRange
	}

	maxPosition := int64(len(p.workBuffer))/p.conf.RecordSize + int64(maxFileNumber+1)*p.recordsPerFile
	minPosition := int64(minFileNumber) * p.recordsPerFile

	if filecount == 0 {
		minPosition = 0
		maxPosition = int64(len(p.workBuffer)) / int64(p.conf.RecordSize)
	}

	switch whence {
	case io.SeekStart: // seek relative to the origin of the file
		p.readPosition = minPosition + offset/int64(p.conf.RecordSize)
	case io.SeekCurrent: // seek relative to the current offset
		p.readPosition += offset / int64(p.conf.RecordSize)
	case io.SeekEnd: //seek relative to the end
		p.readPosition = maxPosition + offset/int64(p.conf.RecordSize)
	default:
		return p.readPosition * int64(p.conf.RecordSize), fmt.Errorf("Whence %v unknow", whence)
	}
	//Set limits and report
	if maxPosition < p.readPosition {
		p.readPosition = maxPosition
	}
	if p.readPosition < minPosition {
		p.readPosition = minPosition
	}
	return p.readPosition * int64(p.conf.RecordSize), nil
}
