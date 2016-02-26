package state

import (
	acm "github.com/shmookey/eris-db/Godeps/_workspace/src/github.com/tendermint/tendermint/account"
	. "github.com/shmookey/eris-db/Godeps/_workspace/src/github.com/tendermint/tendermint/common"
	"github.com/shmookey/eris-db/Godeps/_workspace/src/github.com/tendermint/tendermint/vm"
)

type AccountGetter interface {
	GetAccount(addr []byte) *acm.Account
}

type VMAccountState interface {
	GetAccount(addr Word256) *vm.Account
	UpdateAccount(acc *vm.Account)
	RemoveAccount(acc *vm.Account)
	CreateAccount(creator *vm.Account) *vm.Account
}
