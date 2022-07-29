![FixRegSto](./doc/fixregsto.png)
# FixRegSto

**NOTE THIS IS NOT YET PRODUCTION READY**

This library allows to store fixed length binary blobs in "rotating log" style in different kind of storage mediums in reliable way. This is meant for logging small data droplets that have to be persisted even in case of embedded systems.


Following core specifications
- Reliability is the first priority (copy on write with fsync)
- Store fixed length records on file, write size quantized N*record size
- Does not care about content of file, just fixed record size
- Predictable storage consumption (all files are same size)
- Create interface and support other storage hardware (I2C eeprom?, block device?, raw NAND)
- No data deletion, except old data removal when limit is reached
- Fixed file size limit (tries match erase size and/or minimum file size)

Allowing only fixed size records to rotating log might sound restrictive but it provides
- Faster way to seek data
- Predictability
- Efficient storage usage


Typical use case for fixed size record is for storing vital information like events, counters etc.. Datapoints that are critical for operation but very old entries are not anymore relevant.

## How to use

There is interface *FixRegSto* for accessing stored data. FixRegSto implements ReadWriteSeeker interface. (exception is that it access complete records so N*recordSize quantities). It is possible to read latest data with ioutil.ReadAll IF recordSize is power of two. (ReadAll queries with power of two size chunks)

There are now two implementations
- FileStorage for for file based persistent disk storage. 
    - Does copy on write and fsync. Tries to be atomic
    - Creates number of equal size storage files named with increasing _0,_1,_2 numbering.
    - Set fileMaxSize to N times minimum file size (typical 4k) or page size and get efficient storage
- MemLoop for memory based volatile storage

Check ./example on this repository
