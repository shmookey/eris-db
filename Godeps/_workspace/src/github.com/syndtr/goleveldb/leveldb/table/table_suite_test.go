package table

import (
	"testing"

	"github.com/shmookey/eris-db/Godeps/_workspace/src/github.com/syndtr/goleveldb/leveldb/testutil"
)

func TestTable(t *testing.T) {
	testutil.RunSuite(t, "Table Suite")
}
