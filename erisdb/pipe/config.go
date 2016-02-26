package pipe

import (
	"github.com/shmookey/eris-db/Godeps/_workspace/src/github.com/tendermint/log15"
	cfg "github.com/shmookey/eris-db/Godeps/_workspace/src/github.com/tendermint/tendermint/config"
)

var log = log15.New("module", "eris/erisdb_pipe")
var config cfg.Config

func init() {
	cfg.OnConfig(func(newConfig cfg.Config) {
		config = newConfig
	})
}
