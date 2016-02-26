package main

import (
	"github.com/shmookey/eris-db/Godeps/_workspace/src/github.com/gin-gonic/gin"
	ess "github.com/shmookey/eris-db/erisdb/erisdbss"
	"github.com/shmookey/eris-db/server"
	"os"
	"path"
)

func main() {
	gin.SetMode(gin.ReleaseMode)

	baseDir := path.Join(os.TempDir(), "/.edbservers")
	ss := ess.NewServerServer(baseDir)
	proc := server.NewServeProcess(nil, ss)
	err := proc.Start()
	if err != nil {
		panic(err.Error())
	}
	<-proc.StopEventChannel()
	os.RemoveAll(baseDir)
}
