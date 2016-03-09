package main

import (
	"fmt"
	edb "github.com/shmookey/eris-db/erisdb"
	"os"
	"C"
)

//export Run
func Run(dataDir string) {
	var baseDir string
	if len(os.Args) == 2 {
		baseDir = os.Args[1]
	} else {
		baseDir = os.Getenv("HOME") + "/.erisdb"
	}
  if dataDir != "" {
    baseDir = dataDir
  }

	proc, errSt := edb.ServeErisDB(baseDir)
	if errSt != nil {
		panic(errSt.Error())
	}
	errSe := proc.Start()
	if errSe != nil {
		panic(errSe.Error())
	}
	// TODO For now.
	fmt.Println("DONTMINDME55891")
	<-proc.StopEventChannel()
}

func main() {
  Run("")
}
