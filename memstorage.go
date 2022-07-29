//Memory based loop
//For testing, volatile storage etc...
package fixregsto

import (
	"fmt"
	"io"
)

type MemloopConf struct {
	RecordSize int64 //One entry is this long
	MaxRecords int64
}

type Memloop struct {
	mem       []byte
	conf      MemloopConf
	readIndex int64 //index in mem where read was. Rotating memory also moves this index
}

func (p *MemloopConf) InitMemLoop() (Memloop, error) {
	return Memloop{
		mem:       make([]byte, 0),
		conf:      *p,
		readIndex: 0,
	}, nil
}

//size must be recordsize*N
func (p *Memloop) Write(raw []byte) (n int, err error) {
	if len(raw)%int(p.conf.RecordSize) != 0 {
		return 0, fmt.Errorf("Appended data length %v is not multiple of %v", len(raw), p.conf.RecordSize)
	}
	maxSize := int(p.conf.RecordSize * p.conf.MaxRecords)
	if maxSize < len(raw) {
		return 0, fmt.Errorf("Appended data length %v is over memory size %v", len(raw), maxSize)
	}
	p.mem = append(p.mem, raw...)

	if len(p.mem) <= maxSize {
		return len(raw), nil
	}
	//Just cut?
	p.mem = p.mem[len(p.mem)-maxSize : len(p.mem)]
	return len(raw), nil
}

func (p *Memloop) Len() (int64, error) { //Number of records
	if p.mem == nil {
		return 0, fmt.Errorf("mem is nil")
	}
	return int64(len(p.mem)) / p.conf.RecordSize, nil
}

func (p *Memloop) GetLatest(nRecords int64) ([]byte, error) {
	firstIndex := int64(len(p.mem)) - nRecords*p.conf.RecordSize
	if firstIndex < 0 {
		firstIndex = 0
	}
	return p.mem[firstIndex:len(p.mem)], nil
}

func (p *Memloop) GetFirst(nRecords int64) ([]byte, error) {
	n := int(nRecords * p.conf.RecordSize)
	if len(p.mem) < n {
		n = len(p.mem)
	}
	return p.mem[0:n], nil
}

//ReadAll gets all content. Use with caution, small storages
func (p *Memloop) ReadAll() ([]byte, error) {
	return p.mem, nil
}

func (p *Memloop) Read(arr []byte) (n int, err error) {
	if len(arr) < int(p.conf.RecordSize) { //Breaks read interface but it have to. Avoid io.ReadAll
		//usually problem if non power of 2 record size and io.ReadAll kind of method
		return 0, fmt.Errorf("Asked %v bytes, minimum record size is %v", len(arr), p.conf.RecordSize)
	}

	if p.readIndex < 0 {
		p.readIndex = 0
	}

	maxRecords := int64(len(p.mem)) / p.conf.RecordSize

	recordsNeeded := int64(len(arr)) / p.conf.RecordSize //rounded down
	if maxRecords < recordsNeeded {
		recordsNeeded = maxRecords
	}
	bytesNeeded := p.conf.RecordSize * recordsNeeded // return n is this or lower

	endIndex := bytesNeeded + p.readIndex + 1
	if int64(len(p.mem)) < endIndex {
		endIndex = int64(len(p.mem))
	}
	if endIndex <= p.readIndex {
		return 0, io.EOF
	}

	piece := p.mem[p.readIndex:endIndex]
	/*	if len(piece) != int(bytesNeeded) {
			return 0, fmt.Errorf("Internal error %v vs %v  (lenmem=%v readindex=%v endindex=%v)", len(piece), int(bytesNeeded), len(p.mem), p.readIndex, endIndex)
		}
	*/

	written := copy(arr, piece)
	p.readIndex += int64(written)
	return written, nil
}

func (p *Memloop) Seek(offset int64, whence int) (int64, error) {
	maxPosition := int64(len(p.mem)) / p.conf.RecordSize

	off := p.conf.RecordSize * (offset / p.conf.RecordSize)

	switch whence {
	case io.SeekStart: // seek relative to the origin of the file
		p.readIndex = off
	case io.SeekCurrent: // seek relative to the current offset
		p.readIndex += off
	case io.SeekEnd: //seek relative to the end
		p.readIndex = maxPosition + off
	default:
		return p.readIndex, fmt.Errorf("Whence %v unknow", whence)
	}

	if p.readIndex < 0 {
		p.readIndex = 0
	}

	if maxPosition < p.readIndex {
		p.readIndex = maxPosition
	}
	return p.readIndex, nil
}

/*
func (p *Memloop) ResetRead() error {
	p.readIndex = 0
	return nil
}
*/
