package fixregsto

import (
	"io"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

//Same goes to all implementations?  No, number of available records depend on filesize
func emptytest(t *testing.T, dut FixRegSto) {
	n0, n0err := dut.Len()
	assert.Equal(t, int64(0), n0)
	assert.Equal(t, nil, n0err)

	bytEmptyFirst, errEmptyFirst := dut.GetFirst(1)
	assert.Equal(t, nil, errEmptyFirst)
	assert.Equal(t, []byte{}, bytEmptyFirst)

	bytEmptyLatest, errEmptyLatest := dut.GetLatest(1)
	assert.Equal(t, nil, errEmptyLatest)
	assert.Equal(t, []byte{}, bytEmptyLatest)

	emptystart, errResetReadZero := dut.Seek(0, io.SeekStart)
	assert.Equal(t, nil, errResetReadZero)
	assert.Equal(t, int64(0), emptystart)

	readEmptyBytes, errReadEmptyBytes := ioutil.ReadAll(dut)
	assert.Equal(t, []byte{}, readEmptyBytes)
	assert.Equal(t, nil, errReadEmptyBytes)

	/*
		Reads on empty
	*/

	wrote0, errWriteWrong := dut.Write([]byte{1, 2, 3})
	assert.Equal(t, 0, wrote0)
	assert.NotEqual(t, nil, errWriteWrong)

	/*wrote0, errWriteWrong = dut.Write(make([]byte, 8*1024))
	assert.Equal(t, 0, wrote0)
	assert.NotEqual(t, nil, errWriteWrong)*/

	/*
		Write something
	*/
	wrote1, errWrite1 := dut.Write([]byte{69, 42, 69, 42, 69, 42, 69, 42}) //Special start
	assert.Equal(t, 8, wrote1)
	assert.Equal(t, nil, errWrite1)

	read1Entry, err1Entry := ioutil.ReadAll(dut)
	assert.Equal(t, []byte{69, 42, 69, 42, 69, 42, 69, 42}, read1Entry)
	assert.Equal(t, nil, err1Entry)

	readEmptyBytes, errReadEmptyBytes = ioutil.ReadAll(dut) //Yes it is empty
	assert.Equal(t, []byte{}, readEmptyBytes)
	assert.Equal(t, nil, errReadEmptyBytes)

	bytFirsts, errFirsts := dut.GetFirst(1)
	assert.Equal(t, nil, errFirsts)
	assert.Equal(t, []byte{69, 42, 69, 42, 69, 42, 69, 42}, bytFirsts)

	//Lets write more
	for i := 1; i < 17; i++ {
		b := byte(i)
		wroteMore, errWriteMore := dut.Write([]byte{b, b, b, b, b, b, b, b})
		assert.Equal(t, 8, wroteMore)
		assert.Equal(t, nil, errWriteMore)
	}

	bytLasts, errLasts := dut.GetLatest(1)
	assert.Equal(t, nil, errLasts)
	assert.Equal(t, []byte{0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10}, bytLasts)

	//And then with double size message
	wroteDouble, errWriteDouble := dut.Write([]byte{0xA, 0xB, 0xA, 0xB, 0xA, 0xB, 0xA, 0xB, 0xC, 0xD, 0xC, 0xD, 0xC, 0xD, 0xC, 0xD})
	assert.Equal(t, 16, wroteDouble)
	assert.Equal(t, nil, errWriteDouble)

	readAllEntry, errAllEntry := ioutil.ReadAll(dut)
	assert.Equal(t, []byte{
		1, 1, 1, 1, 1, 1, 1, 1,
		2, 2, 2, 2, 2, 2, 2, 2,
		3, 3, 3, 3, 3, 3, 3, 3,
		4, 4, 4, 4, 4, 4, 4, 4,
		5, 5, 5, 5, 5, 5, 5, 5,
		6, 6, 6, 6, 6, 6, 6, 6,
		7, 7, 7, 7, 7, 7, 7, 7,
		8, 8, 8, 8, 8, 8, 8, 8,
		9, 9, 9, 9, 9, 9, 9, 9,
		0xA, 0xA, 0xA, 0xA, 0xA, 0xA, 0xA, 0xA,
		0xB, 0xB, 0xB, 0xB, 0xB, 0xB, 0xB, 0xB,
		0xC, 0xC, 0xC, 0xC, 0xC, 0xC, 0xC, 0xC,
		0xD, 0xD, 0xD, 0xD, 0xD, 0xD, 0xD, 0xD,
		0xE, 0xE, 0xE, 0xE, 0xE, 0xE, 0xE, 0xE,
		0xF, 0xF, 0xF, 0xF, 0xF, 0xF, 0xF, 0xF,
		0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10,
		0xA, 0xB, 0xA, 0xB, 0xA, 0xB, 0xA, 0xB,
		0xC, 0xD, 0xC, 0xD, 0xC, 0xD, 0xC, 0xD,
	}, readAllEntry)
	assert.Equal(t, nil, errAllEntry)

	//Check latest
	lat, latErr := dut.GetLatest(2)
	assert.Equal(t, []byte{
		0xa, 0xb, 0xa, 0xb, 0xa, 0xb, 0xa, 0xb,
		0xc, 0xd, 0xc, 0xd, 0xc, 0xd, 0xc, 0xd}, lat)
	assert.Equal(t, nil, latErr)

	//Zero fill storage.. except counter?
	for i := 0; i < 10; i++ {
		arr := make([]byte, 128)
		for j := range arr {
			arr[j] = byte(i)
		}
		wroteZero, errZero := dut.Write(arr)
		assert.Equal(t, 128, wroteZero)
		assert.Equal(t, nil, errZero)
	}

	wroteTwo, errWroteTwo := dut.Write([]byte{
		1, 1, 1, 1, 1, 1, 1, 1,
		2, 2, 2, 2, 2, 2, 2, 2,
	})

	assert.Equal(t, nil, errWroteTwo)
	assert.Equal(t, 16, wroteTwo)

	latest3, latest3Err := dut.GetLatest(3)
	assert.Equal(t, nil, latest3Err)
	assert.Equal(t, []byte{
		0x9, 0x9, 0x9, 0x9, 0x9, 0x9, 0x9, 0x9,
		0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1,
		0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2}, latest3)

	//Same with seek
	seeklatest3, errseeklatest3 := dut.Seek(-8*3, io.SeekEnd)
	assert.Equal(t, nil, errseeklatest3)
	assert.Equal(t, int64(1424), seeklatest3)

	seek3data, errseek3data := ioutil.ReadAll(dut) //Must read same as get latest
	assert.Equal(t, nil, errseek3data)
	assert.Equal(t, []byte{
		0x9, 0x9, 0x9, 0x9, 0x9, 0x9, 0x9, 0x9,
		0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1,
		0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2}, seek3data)

	first3, first3Err := dut.GetFirst(6)
	assert.Equal(t, nil, first3Err)
	assert.Equal(t, []byte{
		0x5, 0x5, 0x5, 0x5, 0x5, 0x5, 0x5, 0x5,
		0x5, 0x5, 0x5, 0x5, 0x5, 0x5, 0x5, 0x5,
		0x5, 0x5, 0x5, 0x5, 0x5, 0x5, 0x5, 0x5,
		0x6, 0x6, 0x6, 0x6, 0x6, 0x6, 0x6, 0x6,
		0x6, 0x6, 0x6, 0x6, 0x6, 0x6, 0x6, 0x6,
		0x6, 0x6, 0x6, 0x6, 0x6, 0x6, 0x6, 0x6,
	}, first3)

	/*
		some seek tests
	*/
	seekEndPos, errSeekEndPos := dut.Seek(0, io.SeekEnd)
	assert.Equal(t, nil, errSeekEndPos)
	assert.Equal(t, int64(1448), seekEndPos)

	nodata, errNodata := ioutil.ReadAll(dut)
	assert.Equal(t, nil, errNodata)
	assert.Equal(t, []byte{}, nodata)

	//TODO seek vähän takaisinpäin,
	seekEndPosm1, errSeekEndPosm1 := dut.Seek(-10, io.SeekEnd)
	assert.Equal(t, nil, errSeekEndPosm1)
	assert.Equal(t, int64(1440), seekEndPosm1)

	m1data, errm1data := ioutil.ReadAll(dut)
	assert.Equal(t, nil, errm1data)
	assert.Equal(t, []byte{0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2}, m1data) //TODO HUOMENNA. Seek loppuun niin ei pitäisi tulla tavuja

}

const (
	TMPTESTDIR = "/tmp/filetest12356789"
)

//TODO  generalize
func TestFileOnTMP(t *testing.T) {
	os.RemoveAll(TMPTESTDIR)
	cfg := FileStorageConf{
		Name:         "unitTest",
		RecordSize:   8,
		MaxFileCount: 4,
		FileMaxSize:  128,
		Path:         TMPTESTDIR,
	}
	fl, flErr := cfg.InitFileStorage()
	assert.Equal(t, nil, flErr)

	emptytest(t, &fl)

	//Reload from disk
	flReloaded, flReloadedErr := cfg.InitFileStorage()
	if flReloadedErr != nil {
		t.Error(flReloadedErr)
	}
	finalCount, errFinalCount := flReloaded.Len()
	if errFinalCount != nil {
		t.Error(errFinalCount)
	}
	assert.Equal(t, int64(69), finalCount)
	/*
		Lets try with non matching file
	*/
	os.RemoveAll(TMPTESTDIR)
	cfg = FileStorageConf{
		Name:         "noneven",
		RecordSize:   7,
		MaxFileCount: 4,
		FileMaxSize:  128,
		Path:         TMPTESTDIR,
	}
	fl, flErr = cfg.InitFileStorage()
	if flErr != nil {
		t.Error(flErr)
	}

	wrote7, wrote7err := fl.Write([]byte{1, 2, 3, 4, 5, 6, 7})
	assert.Equal(t, 7, wrote7)
	assert.Equal(t, nil, wrote7err)

	wrote7all, wrote7allerr := fl.Write(make([]byte, 70))
	assert.Equal(t, 70, wrote7all)
	assert.Equal(t, nil, wrote7allerr)

	testReadArr := make([]byte, 8)
	testReadArr[7] = 100
	nread7, nread7err := fl.Read(testReadArr)
	assert.Equal(t, 7, nread7)
	assert.Equal(t, nil, nread7err)
	assert.Equal(t, []byte{0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x64}, testReadArr)

	wrotemucho, wrotemuchoerr := fl.Write(make([]byte, 70000))
	assert.Equal(t, 70000, wrotemucho)
	assert.Equal(t, nil, wrotemuchoerr)

	testReadMuchoArr := make([]byte, 500)
	nreadmucho, nreadmucherr := fl.Read(testReadMuchoArr)
	assert.Equal(t, 497, nreadmucho)
	assert.Equal(t, nil, nreadmucherr)
}
