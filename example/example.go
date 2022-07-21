/*
Simple example just how to use FixRegSto interface
*/

package main

import (
	"fixregsto"
	"fmt"
	"io/ioutil"
)

func testWith8byteRecords(sto fixregsto.FixRegSto) error {
	n, lenErr := sto.Len()
	if lenErr != nil {
		return lenErr
	}
	fmt.Printf("Have stored %v records\n", n)

	for i := 0; i < 255; i++ {
		_, errWrite := sto.Write([]byte{1, 1, 1, 1, 1, 1, 1, 1})
		//_, errWrite := sto.Write([]byte{2, 2, 2, 2, 2, 2, 2, 2})
		//_, errWrite := sto.Write([]byte{10, 10, 10, 10, 10, 10, 10, 10})
		if errWrite != nil {
			return errWrite
		}
	}
	//fmt.Printf("wrote %v\n", wrote)

	latest, latestErr := sto.GetLatest(32)
	if latestErr != nil {
		return latestErr
	}
	fmt.Printf("Latest (%v) %#v\n", len(latest), latest)
	first, firstErr := sto.GetFirst(32)
	if firstErr != nil {
		return firstErr
	}
	fmt.Printf("First (%v) %#v\n", len(first), first)

	return nil
}

func main() {

	alphaConf := fixregsto.FileStorageConf{
		Name:         "alpha",
		RecordSize:   8,
		MaxFileCount: 4,
		FileMaxSize:  512 * 8, //So it is 512 records per file...
		Path:         "./exampledata",
	}
	fmt.Printf("Conf is %#v\n", alphaConf)

	sto, errInit := alphaConf.InitFileStorage()
	if errInit != nil {
		fmt.Printf("Init err %v\n", errInit.Error())
	}
	fmt.Printf("sto %#v\n", sto)
	testErr := testWith8byteRecords(&sto)
	if testErr != nil {
		fmt.Printf("testing failed %v\n", testErr.Error())
	} else {
		fmt.Printf("Test ok")
	}

	allbuf, errbuf := ioutil.ReadAll(&sto)
	if errbuf != nil {
		fmt.Printf("errbuf=%v\n", errbuf.Error())
		return
	}
	fmt.Printf("Allbuf size=%v\n", len(allbuf))

	sto.Write([]byte{0x69, 0x69, 0x69, 0x69, 0x69, 0x69, 0x69, 0x69})

	nextbuf, errnextbuf := ioutil.ReadAll(&sto)
	if errnextbuf != nil {
		fmt.Printf("errnextbuf=%v\n", errnextbuf.Error())
		return
	}
	fmt.Printf("nextbuf %#v \nsize=%v\n", nextbuf, len(nextbuf))

	//use io.copy and make databackup

}
