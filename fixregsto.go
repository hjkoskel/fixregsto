//FixRegSto storage allows to store fixed length binary blobs in "rotating log" style in different kind
//of storage mediums in reliable way. This is meant for logging small data droplets that have to be
//persisted even in case of embedded systems.

package fixregsto

//FixRegSto implements ReadWriteSeeker interface
type FixRegSto interface {
	Write(raw []byte) (n int, err error)      //size must be recordsize*N
	Len() (int64, error)                      //Number of records
	GetLatest(nRecords int64) ([]byte, error) //Without chancing read pointer
	GetFirst(nRecords int64) ([]byte, error)  //Without chancing read pointer
	Seek(offset int64, whence int) (int64, error)
	Read(arr []byte) (n int, err error)
}
