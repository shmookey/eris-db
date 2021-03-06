package consensus

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	_ "github.com/shmookey/eris-db/Godeps/_workspace/src/github.com/tendermint/tendermint/config/tendermint_test"
	"github.com/shmookey/eris-db/Godeps/_workspace/src/github.com/tendermint/tendermint/events"
	"github.com/shmookey/eris-db/Godeps/_workspace/src/github.com/tendermint/tendermint/types"
)

/*

ProposeSuite
x * TestEnterProposeNoValidator - timeout into prevote round
x * TestEnterPropose - finish propose without timing out (we have the proposal)
x * TestBadProposal - 2 vals, bad proposal (bad block state hash), should prevote and precommit nil
FullRoundSuite
x * TestFullRound1 - 1 val, full successful round
x * TestFullRoundNil - 1 val, full round of nil
x * TestFullRound2 - 2 vals, both required for fuill round
LockSuite
x * TestLockNoPOL - 2 vals, 4 rounds. one val locked, precommits nil every round except first.
x * TestLockPOLRelock - 4 vals, one precommits, other 3 polka at next round, so we unlock and precomit the polka
x * TestLockPOLUnlock - 4 vals, one precommits, other 3 polka nil at next round, so we unlock and precomit nil
x * TestLockPOLSafety1 - 4 vals. We shouldn't change lock based on polka at earlier round
x * TestLockPOLSafety2 - 4 vals. After unlocking, we shouldn't relock based on polka at earlier round
  * TestNetworkLock - once +1/3 precommits, network should be locked
  * TestNetworkLockPOL - once +1/3 precommits, the block with more recent polka is committed
SlashingSuite
x * TestSlashingPrevotes - a validator prevoting twice in a round gets slashed
x * TestSlashingPrecommits - a validator precomitting twice in a round gets slashed
CatchupSuite
  * TestCatchup - if we might be behind and we've seen any 2/3 prevotes, round skip to new round, precommit, or prevote
HaltSuite
x * TestHalt1 - if we see +2/3 precommits after timing out into new round, we should still commit

*/

//----------------------------------------------------------------------------------------------------
// ProposeSuite

func init() {
	fmt.Println("")
	timeoutPropose = 1000 * time.Millisecond
}

// a non-validator should timeout into the prevote round
func TestEnterProposeNoPrivValidator(t *testing.T) {
	css, _ := simpleConsensusState(1)
	cs := css[0]
	cs.SetPrivValidator(nil)

	timeoutChan := make(chan struct{})
	evsw := events.NewEventSwitch()
	evsw.OnStart()
	evsw.AddListenerForEvent("tester", types.EventStringTimeoutPropose(), func(data types.EventData) {
		timeoutChan <- struct{}{}
	})
	cs.SetFireable(evsw)

	// starts a go routine for EnterPropose
	cs.EnterNewRound(cs.Height, 0, false)

	// go to prevote
	<-cs.NewStepCh()

	// if we're not a validator, EnterPropose should timeout
	select {
	case rs := <-cs.NewStepCh():
		log.Info(rs.String())
		t.Fatal("Expected EnterPropose to timeout")
	case <-timeoutChan:
		rs := cs.GetRoundState()
		if rs.Proposal != nil {
			t.Error("Expected to make no proposal, since no privValidator")
		}
		break
	}
}

// a validator should not timeout of the prevote round (TODO: unless the block is really big!)
func TestEnterPropose(t *testing.T) {
	css, _ := simpleConsensusState(1)
	cs := css[0]

	timeoutChan := make(chan struct{})
	evsw := events.NewEventSwitch()
	evsw.OnStart()
	evsw.AddListenerForEvent("tester", types.EventStringTimeoutPropose(), func(data types.EventData) {
		timeoutChan <- struct{}{}
	})
	cs.SetFireable(evsw)

	// starts a go routine for EnterPropose
	cs.EnterNewRound(cs.Height, 0, false)

	// go to prevote
	<-cs.NewStepCh()

	// if we are a validator, we expect it not to timeout
	select {
	case <-cs.NewStepCh():
		rs := cs.GetRoundState()

		// Check that Proposal, ProposalBlock, ProposalBlockParts are set.
		if rs.Proposal == nil {
			t.Error("rs.Proposal should be set")
		}
		if rs.ProposalBlock == nil {
			t.Error("rs.ProposalBlock should be set")
		}
		if rs.ProposalBlockParts.Total() == 0 {
			t.Error("rs.ProposalBlockParts should be set")
		}
		break
	case <-timeoutChan:
		t.Fatal("Expected EnterPropose not to timeout")
	}
}

func TestBadProposal(t *testing.T) {
	css, privVals := simpleConsensusState(2)
	cs1, cs2 := css[0], css[1]
	cs1.newStepCh = make(chan *RoundState) // so it blocks

	timeoutChan := make(chan struct{})
	evsw := events.NewEventSwitch()
	evsw.OnStart()
	evsw.AddListenerForEvent("tester", types.EventStringTimeoutPropose(), func(data types.EventData) {
		timeoutChan <- struct{}{}
	})
	evsw.AddListenerForEvent("tester", types.EventStringTimeoutWait(), func(data types.EventData) {
		timeoutChan <- struct{}{}
	})
	cs1.SetFireable(evsw)

	// make the second validator the proposer
	propBlock := changeProposer(t, cs1, cs2)

	// make the block bad by tampering with statehash
	stateHash := propBlock.StateHash
	stateHash[0] = byte((stateHash[0] + 1) % 255)
	propBlock.StateHash = stateHash
	propBlockParts := propBlock.MakePartSet()
	proposal := types.NewProposal(cs2.Height, cs2.Round, propBlockParts.Header(), cs2.Votes.POLRound())
	if err := cs2.privValidator.SignProposal(cs2.state.ChainID, proposal); err != nil {
		t.Fatal("failed to sign bad proposal", err)
	}

	// start round
	cs1.EnterNewRound(cs1.Height, 0, false)

	// now we're on a new round and not the proposer
	<-cs1.NewStepCh()
	// so set the proposal block (and fix voting power)
	cs1.mtx.Lock()
	cs1.Proposal, cs1.ProposalBlock, cs1.ProposalBlockParts = proposal, propBlock, propBlockParts
	fixVotingPower(t, cs1, privVals[1].Address)
	cs1.mtx.Unlock()
	// and wait for timeout
	<-timeoutChan

	// go to prevote, prevote for nil (proposal is bad)
	<-cs1.NewStepCh()
	validatePrevote(t, cs1, 0, privVals[0], nil)

	// add bad prevote from cs2. we should precommit nil
	signAddVoteToFrom(types.VoteTypePrevote, cs1, cs2, propBlock.Hash(), propBlock.MakePartSet().Header())
	_, _, _ = <-cs1.NewStepCh(), <-timeoutChan, <-cs1.NewStepCh()
	validatePrecommit(t, cs1, 0, 0, privVals[0], nil, nil)
	signAddVoteToFrom(types.VoteTypePrecommit, cs1, cs2, propBlock.Hash(), propBlock.MakePartSet().Header())
}

//----------------------------------------------------------------------------------------------------
// FulLRoundSuite

// propose, prevote, and precommit a block
func TestFullRound1(t *testing.T) {
	css, privVals := simpleConsensusState(1)
	cs := css[0]

	// starts a go routine for EnterPropose
	cs.EnterNewRound(cs.Height, 0, false)
	// wait to finish propose and prevote
	_, _ = <-cs.NewStepCh(), <-cs.NewStepCh()

	// we should now be in precommit
	// verify our prevote is there
	cs.mtx.Lock()
	propBlockHash := cs.ProposalBlock.Hash()
	cs.mtx.Unlock()

	// the proposed block should be prevoted, precommitted, and locked
	validatePrevoteAndPrecommit(t, cs, 0, 0, privVals[0], propBlockHash, propBlockHash, nil)
}

// nil is proposed, so prevote and precommit nil
func TestFullRoundNil(t *testing.T) {
	css, privVals := simpleConsensusState(1)
	cs := css[0]
	cs.newStepCh = make(chan *RoundState) // so it blocks
	cs.SetPrivValidator(nil)

	timeoutChan := make(chan struct{})
	evsw := events.NewEventSwitch()
	evsw.OnStart()
	evsw.AddListenerForEvent("tester", types.EventStringTimeoutPropose(), func(data types.EventData) {
		timeoutChan <- struct{}{}
	})
	cs.SetFireable(evsw)

	// starts a go routine for EnterPropose
	cs.EnterNewRound(cs.Height, 0, false)

	// wait to finish propose (we should time out)
	<-cs.NewStepCh()
	cs.SetPrivValidator(privVals[0]) // this might be a race condition (uses the mutex that EnterPropose has just released and EnterPrevote is about to grab)
	<-timeoutChan

	// wait to finish prevote
	<-cs.NewStepCh()

	// should prevote and precommit nil
	validatePrevoteAndPrecommit(t, cs, 0, 0, privVals[0], nil, nil, nil)
}

// run through propose, prevote, precommit commit with two validators
// where the first validator has to wait for votes from the second
func TestFullRound2(t *testing.T) {
	css, privVals := simpleConsensusState(2)
	cs1, cs2 := css[0], css[1]
	cs1.newStepCh = make(chan *RoundState) // so it blocks
	cs2.newStepCh = make(chan *RoundState) // so it blocks

	// start round and wait for propose and prevote
	cs1.EnterNewRound(cs1.Height, 0, false)
	_, _ = <-cs1.NewStepCh(), <-cs1.NewStepCh()

	// we should now be stuck in limbo forever, waiting for more prevotes
	ensureNoNewStep(t, cs1)

	propBlockHash, propPartsHeader := cs1.ProposalBlock.Hash(), cs1.ProposalBlockParts.Header()

	// prevote arrives from cs2:
	signAddVoteToFrom(types.VoteTypePrevote, cs1, cs2, propBlockHash, propPartsHeader)

	// wait to finish precommit
	<-cs1.NewStepCh()

	// the proposed block should now be locked and our precommit added
	validatePrecommit(t, cs1, 0, 0, privVals[0], propBlockHash, propBlockHash)

	// we should now be stuck in limbo forever, waiting for more precommits
	ensureNoNewStep(t, cs1)

	// precommit arrives from cs2:
	signAddVoteToFrom(types.VoteTypePrecommit, cs1, cs2, propBlockHash, propPartsHeader)

	// wait to finish commit, propose in next height
	_, rs := <-cs1.NewStepCh(), <-cs1.NewStepCh()
	if rs.Height != 2 {
		t.Fatal("Expected height to increment")
	}
}

//------------------------------------------------------------------------------------------
// LockSuite

// two validators, 4 rounds.
// val1 proposes the first 2 rounds, and is locked in the first.
// val2 proposes the next two. val1 should precommit nil on all (except first where he locks)
func TestLockNoPOL(t *testing.T) {
	css, privVals := simpleConsensusState(2)
	cs1, cs2 := css[0], css[1]
	cs1.newStepCh = make(chan *RoundState) // so it blocks

	timeoutChan := make(chan struct{})
	evsw := events.NewEventSwitch()
	evsw.OnStart()
	evsw.AddListenerForEvent("tester", types.EventStringTimeoutPropose(), func(data types.EventData) {
		timeoutChan <- struct{}{}
	})
	evsw.AddListenerForEvent("tester", types.EventStringTimeoutWait(), func(data types.EventData) {
		timeoutChan <- struct{}{}
	})
	cs1.SetFireable(evsw)

	/*
		Round1 (cs1, B) // B B // B B2
	*/

	// start round and wait for propose and prevote
	cs1.EnterNewRound(cs1.Height, 0, false)
	_, _ = <-cs1.NewStepCh(), <-cs1.NewStepCh()

	// we should now be stuck in limbo forever, waiting for more prevotes
	// prevote arrives from cs2:
	signAddVoteToFrom(types.VoteTypePrevote, cs1, cs2, cs1.ProposalBlock.Hash(), cs1.ProposalBlockParts.Header())

	cs1.mtx.Lock() // XXX: sigh
	theBlockHash := cs1.ProposalBlock.Hash()
	cs1.mtx.Unlock()

	// wait to finish precommit
	<-cs1.NewStepCh()

	// the proposed block should now be locked and our precommit added
	validatePrecommit(t, cs1, 0, 0, privVals[0], theBlockHash, theBlockHash)

	// we should now be stuck in limbo forever, waiting for more precommits
	// lets add one for a different block
	// NOTE: in practice we should never get to a point where there are precommits for different blocks at the same round
	hash := cs1.ProposalBlock.Hash()
	hash[0] = byte((hash[0] + 1) % 255)
	signAddVoteToFrom(types.VoteTypePrecommit, cs1, cs2, hash, cs1.ProposalBlockParts.Header())

	// (note we're entering precommit for a second time this round)
	// but with invalid args. then we EnterPrecommitWait, and the timeout to new round
	_, _ = <-cs1.NewStepCh(), <-timeoutChan

	log.Info("#### ONTO ROUND 2")
	/*
		Round2 (cs1, B) // B B2
	*/

	incrementRound(cs2)

	// go to prevote
	<-cs1.NewStepCh()

	// now we're on a new round and the proposer
	if cs1.ProposalBlock != cs1.LockedBlock {
		t.Fatalf("Expected proposal block to be locked block. Got %v, Expected %v", cs1.ProposalBlock, cs1.LockedBlock)
	}

	// wait to finish prevote
	<-cs1.NewStepCh()

	// we should have prevoted our locked block
	validatePrevote(t, cs1, 1, privVals[0], cs1.LockedBlock.Hash())

	// add a conflicting prevote from the other validator
	signAddVoteToFrom(types.VoteTypePrevote, cs1, cs2, hash, cs1.ProposalBlockParts.Header())

	// now we're going to enter prevote again, but with invalid args
	// and then prevote wait, which should timeout. then wait for precommit
	_, _, _ = <-cs1.NewStepCh(), <-timeoutChan, <-cs1.NewStepCh()

	// the proposed block should still be locked and our precommit added
	// we should precommit nil and be locked on the proposal
	validatePrecommit(t, cs1, 1, 0, privVals[0], nil, theBlockHash)

	// add conflicting precommit from cs2
	// NOTE: in practice we should never get to a point where there are precommits for different blocks at the same round
	signAddVoteToFrom(types.VoteTypePrecommit, cs1, cs2, hash, cs1.ProposalBlockParts.Header())

	// (note we're entering precommit for a second time this round, but with invalid args
	// then we EnterPrecommitWait and timeout into NewRound
	_, _ = <-cs1.NewStepCh(), <-timeoutChan

	log.Info("#### ONTO ROUND 3")
	/*
		Round3 (cs2, _) // B, B2
	*/

	incrementRound(cs2)

	// now we're on a new round and not the proposer, so wait for timeout
	_, _ = <-cs1.NewStepCh(), <-timeoutChan
	if cs1.ProposalBlock != nil {
		t.Fatal("Expected proposal block to be nil")
	}

	// go to prevote, prevote for locked block
	<-cs1.NewStepCh()
	validatePrevote(t, cs1, 0, privVals[0], cs1.LockedBlock.Hash())

	// TODO: quick fastforward to new round, set proposer
	signAddVoteToFrom(types.VoteTypePrevote, cs1, cs2, hash, cs1.ProposalBlockParts.Header())
	_, _, _ = <-cs1.NewStepCh(), <-timeoutChan, <-cs1.NewStepCh()
	validatePrecommit(t, cs1, 2, 0, privVals[0], nil, theBlockHash)                             // precommit nil but be locked on proposal
	signAddVoteToFrom(types.VoteTypePrecommit, cs1, cs2, hash, cs1.ProposalBlockParts.Header()) // NOTE: conflicting precommits at same height

	<-cs1.NewStepCh()

	// before we time out into new round, set next proposer
	// and next proposal block
	_, v1 := cs1.Validators.GetByAddress(privVals[0].Address)
	v1.VotingPower = 1
	if updated := cs1.Validators.Update(v1); !updated {
		t.Fatal("failed to update validator")
	}

	cs2.decideProposal(cs2.Height, cs2.Round+1)
	prop, propBlock := cs2.Proposal, cs2.ProposalBlock
	if prop == nil || propBlock == nil {
		t.Fatal("Failed to create proposal block with cs2")
	}

	incrementRound(cs2)

	<-timeoutChan

	log.Info("#### ONTO ROUND 4")
	/*
		Round4 (cs2, C) // B C // B C
	*/

	// now we're on a new round and not the proposer
	<-cs1.NewStepCh()
	// so set the proposal block
	cs1.mtx.Lock()
	cs1.Proposal, cs1.ProposalBlock = prop, propBlock
	cs1.mtx.Unlock()
	// and wait for timeout
	<-timeoutChan
	// go to prevote, prevote for locked block (not proposal)
	<-cs1.NewStepCh()
	validatePrevote(t, cs1, 0, privVals[0], cs1.LockedBlock.Hash())

	signAddVoteToFrom(types.VoteTypePrevote, cs1, cs2, propBlock.Hash(), propBlock.MakePartSet().Header())
	_, _, _ = <-cs1.NewStepCh(), <-timeoutChan, <-cs1.NewStepCh()
	validatePrecommit(t, cs1, 2, 0, privVals[0], nil, theBlockHash)                                          // precommit nil but locked on proposal
	signAddVoteToFrom(types.VoteTypePrecommit, cs1, cs2, propBlock.Hash(), propBlock.MakePartSet().Header()) // NOTE: conflicting precommits at same height
}

// 4 vals, one precommits, other 3 polka at next round, so we unlock and precomit the polka
func TestLockPOLRelock(t *testing.T) {
	css, privVals := simpleConsensusState(4)
	cs1, cs2, cs3, cs4 := css[0], css[1], css[2], css[3]
	cs1.newStepCh = make(chan *RoundState) // so it blocks

	timeoutChan := make(chan *types.EventDataRoundState)
	voteChan := make(chan *types.EventDataVote)
	evsw := events.NewEventSwitch()
	evsw.OnStart()
	evsw.AddListenerForEvent("tester", types.EventStringTimeoutPropose(), func(data types.EventData) {
		timeoutChan <- data.(*types.EventDataRoundState)
	})
	evsw.AddListenerForEvent("tester", types.EventStringTimeoutWait(), func(data types.EventData) {
		timeoutChan <- data.(*types.EventDataRoundState)
	})
	evsw.AddListenerForEvent("tester", types.EventStringVote(), func(data types.EventData) {
		vote := data.(*types.EventDataVote)
		// we only fire for our own votes
		if bytes.Equal(cs1.privValidator.Address, vote.Address) {
			voteChan <- vote
		}
	})
	cs1.SetFireable(evsw)

	// everything done from perspective of cs1

	/*
		Round1 (cs1, B) // B B B B// B nil B nil

		eg. cs2 and cs4 didn't see the 2/3 prevotes
	*/

	// start round and wait for propose and prevote
	cs1.EnterNewRound(cs1.Height, 0, false)
	_, _, _ = <-cs1.NewStepCh(), <-voteChan, <-cs1.NewStepCh()

	theBlockHash := cs1.ProposalBlock.Hash()

	// wait to finish precommit after prevotes done
	// we do this in a go routine with another channel since otherwise
	// we may get deadlock with EnterPrecommit waiting to send on newStepCh and the final
	// signAddVoteToFrom waiting for the cs.mtx.Lock
	donePrecommit := make(chan struct{})
	go func() {
		<-voteChan
		<-cs1.NewStepCh()
		donePrecommit <- struct{}{}
	}()
	signAddVoteToFromMany(types.VoteTypePrevote, cs1, cs1.ProposalBlock.Hash(), cs1.ProposalBlockParts.Header(), cs2, cs3, cs4)
	<-donePrecommit

	// the proposed block should now be locked and our precommit added
	validatePrecommit(t, cs1, 0, 0, privVals[0], theBlockHash, theBlockHash)

	donePrecommitWait := make(chan struct{})
	go func() {
		// (note we're entering precommit for a second time this round)
		// but with invalid args. then we EnterPrecommitWait, twice (?)
		<-cs1.NewStepCh()
		donePrecommitWait <- struct{}{}
	}()
	// add precommits from the rest
	signAddVoteToFromMany(types.VoteTypePrecommit, cs1, nil, types.PartSetHeader{}, cs2, cs4)
	signAddVoteToFrom(types.VoteTypePrecommit, cs1, cs3, cs1.ProposalBlock.Hash(), cs1.ProposalBlockParts.Header())
	<-donePrecommitWait

	// before we time out into new round, set next proposer
	// and next proposal block
	_, v1 := cs1.Validators.GetByAddress(privVals[0].Address)
	v1.VotingPower = 1
	if updated := cs1.Validators.Update(v1); !updated {
		t.Fatal("failed to update validator")
	}

	cs2.decideProposal(cs2.Height, cs2.Round+1)
	prop, propBlock := cs2.Proposal, cs2.ProposalBlock
	if prop == nil || propBlock == nil {
		t.Fatal("Failed to create proposal block with cs2")
	}

	incrementRound(cs2, cs3, cs4)

	// timeout to new round
	te := <-timeoutChan
	if te.Step != RoundStepPrecommitWait.String() {
		t.Fatalf("expected to timeout of precommit into new round. got %v", te.Step)
	}

	log.Info("### ONTO ROUND 2")

	/*
		Round2 (cs2, C) // B C C C // C C C _)

		cs1 changes lock!
	*/

	// now we're on a new round and not the proposer
	<-cs1.NewStepCh()
	cs1.mtx.Lock()
	// so set the proposal block
	propBlockHash, propBlockParts := propBlock.Hash(), propBlock.MakePartSet()
	cs1.Proposal, cs1.ProposalBlock, cs1.ProposalBlockParts = prop, propBlock, propBlockParts
	cs1.mtx.Unlock()
	// and wait for timeout
	te = <-timeoutChan
	if te.Step != RoundStepPropose.String() {
		t.Fatalf("expected to timeout of propose. got %v", te.Step)
	}
	// go to prevote, prevote for locked block (not proposal), move on
	_, _ = <-voteChan, <-cs1.NewStepCh()
	validatePrevote(t, cs1, 0, privVals[0], theBlockHash)

	donePrecommit = make(chan struct{})
	go func() {
		//  we need this go routine because if we go into PrevoteWait it has to pull on newStepCh
		// before the final vote will get added (because it holds the mutex).
		select {
		case <-cs1.NewStepCh(): // we're in PrevoteWait, go to Precommit
			<-voteChan
		case <-voteChan: // we went straight to Precommit
		}
		donePrecommit <- struct{}{}
	}()
	// now lets add prevotes from everyone else for the new block
	signAddVoteToFromMany(types.VoteTypePrevote, cs1, propBlockHash, propBlockParts.Header(), cs2, cs3, cs4)
	<-donePrecommit

	// we should have unlocked and locked on the new block
	validatePrecommit(t, cs1, 1, 1, privVals[0], propBlockHash, propBlockHash)

	donePrecommitWait = make(chan struct{})
	go func() {
		// (note we're entering precommit for a second time this round)
		// but with invalid args. then we EnterPrecommitWait,
		<-cs1.NewStepCh()
		donePrecommitWait <- struct{}{}
	}()
	signAddVoteToFromMany(types.VoteTypePrecommit, cs1, propBlockHash, propBlockParts.Header(), cs2, cs3)
	<-donePrecommitWait

	<-cs1.NewStepCh()
	rs := <-cs1.NewStepCh()
	if rs.Height != 2 {
		t.Fatal("Expected height to increment")
	}

	if hash, _, ok := rs.LastCommit.TwoThirdsMajority(); !ok || !bytes.Equal(hash, propBlockHash) {
		t.Fatal("Expected block to get committed")
	}
}

// 4 vals, one precommits, other 3 polka at next round, so we unlock and precomit the polka
func TestLockPOLUnlock(t *testing.T) {
	css, privVals := simpleConsensusState(4)
	cs1, cs2, cs3, cs4 := css[0], css[1], css[2], css[3]
	cs1.newStepCh = make(chan *RoundState) // so it blocks

	timeoutChan := make(chan *types.EventDataRoundState)
	voteChan := make(chan *types.EventDataVote)
	evsw := events.NewEventSwitch()
	evsw.OnStart()
	evsw.AddListenerForEvent("tester", types.EventStringTimeoutPropose(), func(data types.EventData) {
		timeoutChan <- data.(*types.EventDataRoundState)
	})
	evsw.AddListenerForEvent("tester", types.EventStringTimeoutWait(), func(data types.EventData) {
		timeoutChan <- data.(*types.EventDataRoundState)
	})
	evsw.AddListenerForEvent("tester", types.EventStringVote(), func(data types.EventData) {
		vote := data.(*types.EventDataVote)
		// we only fire for our own votes
		if bytes.Equal(cs1.privValidator.Address, vote.Address) {
			voteChan <- vote
		}
	})
	cs1.SetFireable(evsw)

	// everything done from perspective of cs1

	/*
		Round1 (cs1, B) // B B B B // B nil B nil

		eg. didn't see the 2/3 prevotes
	*/

	// start round and wait for propose and prevote
	cs1.EnterNewRound(cs1.Height, 0, false)
	_, _, _ = <-cs1.NewStepCh(), <-voteChan, <-cs1.NewStepCh()

	theBlockHash := cs1.ProposalBlock.Hash()

	donePrecommit := make(chan struct{})
	go func() {
		<-voteChan
		<-cs1.NewStepCh()
		donePrecommit <- struct{}{}
	}()
	signAddVoteToFromMany(types.VoteTypePrevote, cs1, cs1.ProposalBlock.Hash(), cs1.ProposalBlockParts.Header(), cs2, cs3, cs4)
	<-donePrecommit

	// the proposed block should now be locked and our precommit added
	validatePrecommit(t, cs1, 0, 0, privVals[0], theBlockHash, theBlockHash)

	donePrecommitWait := make(chan struct{})
	go func() {
		<-cs1.NewStepCh()
		donePrecommitWait <- struct{}{}
	}()
	// add precommits from the rest
	signAddVoteToFromMany(types.VoteTypePrecommit, cs1, nil, types.PartSetHeader{}, cs2, cs4)
	signAddVoteToFrom(types.VoteTypePrecommit, cs1, cs3, cs1.ProposalBlock.Hash(), cs1.ProposalBlockParts.Header())
	<-donePrecommitWait

	// before we time out into new round, set next proposer
	// and next proposal block
	_, v1 := cs1.Validators.GetByAddress(privVals[0].Address)
	v1.VotingPower = 1
	if updated := cs1.Validators.Update(v1); !updated {
		t.Fatal("failed to update validator")
	}

	cs2.decideProposal(cs2.Height, cs2.Round+1)
	prop, propBlock := cs2.Proposal, cs2.ProposalBlock
	if prop == nil || propBlock == nil {
		t.Fatal("Failed to create proposal block with cs2")
	}

	incrementRound(cs2, cs3, cs4)

	// timeout to new round
	<-timeoutChan

	log.Info("#### ONTO ROUND 2")
	/*
		Round2 (cs2, C) // B nil nil nil // nil nil nil _

		cs1 unlocks!
	*/

	// now we're on a new round and not the proposer,
	<-cs1.NewStepCh()
	cs1.mtx.Lock()
	// so set the proposal block
	cs1.Proposal, cs1.ProposalBlock, cs1.ProposalBlockParts = prop, propBlock, propBlock.MakePartSet()
	lockedBlockHash := cs1.LockedBlock.Hash()
	cs1.mtx.Unlock()
	// and wait for timeout
	<-timeoutChan

	// go to prevote, prevote for locked block (not proposal)
	_, _ = <-voteChan, <-cs1.NewStepCh()
	validatePrevote(t, cs1, 0, privVals[0], lockedBlockHash)

	donePrecommit = make(chan struct{})
	go func() {
		select {
		case <-cs1.NewStepCh(): // we're in PrevoteWait, go to Precommit
			<-voteChan
		case <-voteChan: // we went straight to Precommit
		}
		donePrecommit <- struct{}{}
	}()
	// now lets add prevotes from everyone else for the new block
	signAddVoteToFromMany(types.VoteTypePrevote, cs1, nil, types.PartSetHeader{}, cs2, cs3, cs4)
	<-donePrecommit

	// we should have unlocked
	// NOTE: we don't lock on nil, so LockedRound is still 0
	validatePrecommit(t, cs1, 1, 0, privVals[0], nil, nil)

	donePrecommitWait = make(chan struct{})
	go func() {
		// the votes will bring us to new round right away
		// we should timeout of it
		_, _, _ = <-cs1.NewStepCh(), <-cs1.NewStepCh(), <-timeoutChan
		donePrecommitWait <- struct{}{}
	}()
	signAddVoteToFromMany(types.VoteTypePrecommit, cs1, nil, types.PartSetHeader{}, cs2, cs3)
	<-donePrecommitWait
}

// 4 vals
// a polka at round 1 but we miss it
// then a polka at round 2 that we lock on
// then we see the polka from round 1 but shouldn't unlock
func TestLockPOLSafety1(t *testing.T) {
	css, privVals := simpleConsensusState(4)
	cs1, cs2, cs3, cs4 := css[0], css[1], css[2], css[3]
	cs1.newStepCh = make(chan *RoundState) // so it blocks

	timeoutChan := make(chan *types.EventDataRoundState)
	voteChan := make(chan *types.EventDataVote)
	evsw := events.NewEventSwitch()
	evsw.OnStart()
	evsw.AddListenerForEvent("tester", types.EventStringTimeoutPropose(), func(data types.EventData) {
		timeoutChan <- data.(*types.EventDataRoundState)
	})
	evsw.AddListenerForEvent("tester", types.EventStringTimeoutWait(), func(data types.EventData) {
		timeoutChan <- data.(*types.EventDataRoundState)
	})
	evsw.AddListenerForEvent("tester", types.EventStringVote(), func(data types.EventData) {
		vote := data.(*types.EventDataVote)
		// we only fire for our own votes
		if bytes.Equal(cs1.privValidator.Address, vote.Address) {
			voteChan <- vote
		}
	})
	cs1.SetFireable(evsw)

	// start round and wait for propose and prevote
	cs1.EnterNewRound(cs1.Height, 0, false)
	_, _, _ = <-cs1.NewStepCh(), <-voteChan, <-cs1.NewStepCh()

	propBlock := cs1.ProposalBlock

	validatePrevote(t, cs1, 0, privVals[0], cs1.ProposalBlock.Hash())

	// the others sign a polka but we don't see it
	prevotes := signVoteMany(types.VoteTypePrevote, propBlock.Hash(), propBlock.MakePartSet().Header(), cs2, cs3, cs4)

	// before we time out into new round, set next proposer
	// and next proposal block
	_, v1 := cs1.Validators.GetByAddress(privVals[0].Address)
	v1.VotingPower = 1
	if updated := cs1.Validators.Update(v1); !updated {
		t.Fatal("failed to update validator")
	}

	log.Warn("old prop", "hash", fmt.Sprintf("%X", propBlock.Hash()))

	// we do see them precommit nil
	signAddVoteToFromMany(types.VoteTypePrecommit, cs1, nil, types.PartSetHeader{}, cs2, cs3, cs4)

	cs2.decideProposal(cs2.Height, cs2.Round+1)
	prop, propBlock := cs2.Proposal, cs2.ProposalBlock
	if prop == nil || propBlock == nil {
		t.Fatal("Failed to create proposal block with cs2")
	}

	incrementRound(cs2, cs3, cs4)

	log.Info("### ONTO ROUND 2")
	/*Round2
	// we timeout and prevote our lock
	// a polka happened but we didn't see it!
	*/

	// now we're on a new round and not the proposer,
	<-cs1.NewStepCh()
	// so set proposal
	cs1.mtx.Lock()
	propBlockHash, propBlockParts := propBlock.Hash(), propBlock.MakePartSet()
	cs1.Proposal, cs1.ProposalBlock, cs1.ProposalBlockParts = prop, propBlock, propBlockParts
	cs1.mtx.Unlock()
	// and wait for timeout
	<-timeoutChan
	if cs1.LockedBlock != nil {
		t.Fatal("we should not be locked!")
	}
	log.Warn("new prop", "hash", fmt.Sprintf("%X", propBlockHash))
	// go to prevote, prevote for proposal block
	_, _ = <-voteChan, <-cs1.NewStepCh()
	validatePrevote(t, cs1, 1, privVals[0], propBlockHash)

	// now we see the others prevote for it, so we should lock on it
	donePrecommit := make(chan struct{})
	go func() {
		select {
		case <-cs1.NewStepCh(): // we're in PrevoteWait, go to Precommit
			<-voteChan
		case <-voteChan: // we went straight to Precommit
		}
		<-cs1.NewStepCh()
		donePrecommit <- struct{}{}
	}()
	// now lets add prevotes from everyone else for nil
	signAddVoteToFromMany(types.VoteTypePrevote, cs1, propBlockHash, propBlockParts.Header(), cs2, cs3, cs4)
	<-donePrecommit

	// we should have precommitted
	validatePrecommit(t, cs1, 1, 1, privVals[0], propBlockHash, propBlockHash)

	// now we see precommits for nil
	donePrecommitWait := make(chan struct{})
	go func() {
		// the votes will bring us to new round
		// we should timeut of it and go to prevote
		<-cs1.NewStepCh()
		<-timeoutChan
		donePrecommitWait <- struct{}{}
	}()
	signAddVoteToFromMany(types.VoteTypePrecommit, cs1, nil, types.PartSetHeader{}, cs2, cs3)
	<-donePrecommitWait

	incrementRound(cs2, cs3, cs4)

	log.Info("### ONTO ROUND 3")
	/*Round3
	we see the polka from round 1 but we shouldn't unlock!
	*/

	// timeout of propose
	_, _ = <-cs1.NewStepCh(), <-timeoutChan

	// finish prevote
	_, _ = <-voteChan, <-cs1.NewStepCh()

	// we should prevote what we're locked on
	validatePrevote(t, cs1, 2, privVals[0], propBlockHash)

	// add prevotes from the earlier round
	addVoteToFromMany(cs1, prevotes, cs2, cs3, cs4)

	log.Warn("Done adding prevotes!")

	ensureNoNewStep(t, cs1)
}

// 4 vals.
// polka P1 at R1, P2 at R2, and P3 at R3,
// we lock on P1 at R1, don't see P2, and unlock using P3 at R3
// then we should make sure we don't lock using P2
func TestLockPOLSafety2(t *testing.T) {
	css, privVals := simpleConsensusState(4)
	cs1, cs2, cs3, cs4 := css[0], css[1], css[2], css[3]
	cs1.newStepCh = make(chan *RoundState) // so it blocks

	timeoutChan := make(chan *types.EventDataRoundState)
	voteChan := make(chan *types.EventDataVote)
	evsw := events.NewEventSwitch()
	evsw.OnStart()
	evsw.AddListenerForEvent("tester", types.EventStringTimeoutPropose(), func(data types.EventData) {
		timeoutChan <- data.(*types.EventDataRoundState)
	})
	evsw.AddListenerForEvent("tester", types.EventStringTimeoutWait(), func(data types.EventData) {
		timeoutChan <- data.(*types.EventDataRoundState)
	})
	evsw.AddListenerForEvent("tester", types.EventStringVote(), func(data types.EventData) {
		vote := data.(*types.EventDataVote)
		// we only fire for our own votes
		if bytes.Equal(cs1.privValidator.Address, vote.Address) {
			voteChan <- vote
		}
	})
	cs1.SetFireable(evsw)

	// start round and wait for propose and prevote
	cs1.EnterNewRound(cs1.Height, 0, false)
	_, _, _ = <-cs1.NewStepCh(), <-voteChan, <-cs1.NewStepCh()

	theBlockHash := cs1.ProposalBlock.Hash()

	donePrecommit := make(chan struct{})
	go func() {
		<-voteChan
		<-cs1.NewStepCh()
		donePrecommit <- struct{}{}
	}()
	signAddVoteToFromMany(types.VoteTypePrevote, cs1, cs1.ProposalBlock.Hash(), cs1.ProposalBlockParts.Header(), cs2, cs3, cs4)
	<-donePrecommit

	// the proposed block should now be locked and our precommit added
	validatePrecommit(t, cs1, 0, 0, privVals[0], theBlockHash, theBlockHash)

	donePrecommitWait := make(chan struct{})
	go func() {
		<-cs1.NewStepCh()
		donePrecommitWait <- struct{}{}
	}()
	// add precommits from the rest
	signAddVoteToFromMany(types.VoteTypePrecommit, cs1, nil, types.PartSetHeader{}, cs2, cs4)
	signAddVoteToFrom(types.VoteTypePrecommit, cs1, cs3, cs1.ProposalBlock.Hash(), cs1.ProposalBlockParts.Header())
	<-donePrecommitWait

	// before we time out into new round, set next proposer
	// and next proposal block
	_, v1 := cs1.Validators.GetByAddress(privVals[0].Address)
	v1.VotingPower = 1
	if updated := cs1.Validators.Update(v1); !updated {
		t.Fatal("failed to update validator")
	}

	cs2.decideProposal(cs2.Height, cs2.Round+1)
	prop, propBlock := cs2.Proposal, cs2.ProposalBlock
	if prop == nil || propBlock == nil {
		t.Fatal("Failed to create proposal block with cs2")
	}

	incrementRound(cs2, cs3, cs4)

	// timeout to new round
	<-timeoutChan

	log.Info("### ONTO Round 2")
	/*Round2
	// we timeout and prevote our lock
	// a polka happened but we didn't see it!
	*/

	// now we're on a new round and not the proposer, so wait for timeout
	_, _ = <-cs1.NewStepCh(), <-timeoutChan
	// go to prevote, prevote for locked block
	_, _ = <-voteChan, <-cs1.NewStepCh()
	validatePrevote(t, cs1, 0, privVals[0], cs1.LockedBlock.Hash())

	// the others sign a polka but we don't see it
	prevotes := signVoteMany(types.VoteTypePrevote, propBlock.Hash(), propBlock.MakePartSet().Header(), cs2, cs3, cs4)

	// once we see prevotes for the next round we'll skip ahead

	incrementRound(cs2, cs3, cs4)

	log.Info("### ONTO Round 3")
	/*Round3
	a polka for nil causes us to unlock
	*/

	// these prevotes will send us straight to precommit at the higher round
	donePrecommit = make(chan struct{})
	go func() {
		select {
		case <-cs1.NewStepCh(): // we're in PrevoteWait, go to Precommit
			<-voteChan
		case <-voteChan: // we went straight to Precommit
		}
		<-cs1.NewStepCh()
		donePrecommit <- struct{}{}
	}()
	// now lets add prevotes from everyone else for nil
	signAddVoteToFromMany(types.VoteTypePrevote, cs1, nil, types.PartSetHeader{}, cs2, cs3, cs4)
	<-donePrecommit

	// we should have unlocked
	// NOTE: we don't lock on nil, so LockedRound is still 0
	validatePrecommit(t, cs1, 2, 0, privVals[0], nil, nil)

	donePrecommitWait = make(chan struct{})
	go func() {
		// the votes will bring us to new round right away
		// we should timeut of it and go to prevote
		<-cs1.NewStepCh()
		// set the proposal block to be that which got a polka in R2
		cs1.mtx.Lock()
		cs1.Proposal, cs1.ProposalBlock, cs1.ProposalBlockParts = prop, propBlock, propBlock.MakePartSet()
		cs1.mtx.Unlock()
		// timeout into prevote, finish prevote
		_, _, _ = <-timeoutChan, <-voteChan, <-cs1.NewStepCh()
		donePrecommitWait <- struct{}{}
	}()
	signAddVoteToFromMany(types.VoteTypePrecommit, cs1, nil, types.PartSetHeader{}, cs2, cs3)
	<-donePrecommitWait

	log.Info("### ONTO ROUND 4")
	/*Round4
	we see the polka from R2
	make sure we don't lock because of it!
	*/
	// new round and not proposer
	// (we already timed out and stepped into prevote)

	log.Warn("adding prevotes from round 2")

	addVoteToFromMany(cs1, prevotes, cs2, cs3, cs4)

	log.Warn("Done adding prevotes!")

	// we should prevote it now
	validatePrevote(t, cs1, 3, privVals[0], cs1.ProposalBlock.Hash())

	// but we shouldn't precommit it
	precommits := cs1.Votes.Precommits(3)
	vote := precommits.GetByIndex(0)
	if vote != nil {
		t.Fatal("validator precommitted at round 4 based on an old polka")
	}
}

//------------------------------------------------------------------------------------------
// SlashingSuite

func TestSlashingPrevotes(t *testing.T) {
	css, _ := simpleConsensusState(2)
	cs1, cs2 := css[0], css[1]
	cs1.newStepCh = make(chan *RoundState) // so it blocks

	// start round and wait for propose and prevote
	cs1.EnterNewRound(cs1.Height, 0, false)
	_, _ = <-cs1.NewStepCh(), <-cs1.NewStepCh()

	// we should now be stuck in limbo forever, waiting for more prevotes
	// add one for a different block should cause us to go into prevote wait
	hash := cs1.ProposalBlock.Hash()
	hash[0] = byte(hash[0]+1) % 255
	signAddVoteToFrom(types.VoteTypePrevote, cs1, cs2, hash, cs1.ProposalBlockParts.Header())

	// pass prevote wait
	<-cs1.NewStepCh()

	// NOTE: we have to send the vote for different block first so we don't just go into precommit round right
	// away and ignore more prevotes (and thus fail to slash!)

	// add the conflicting vote
	signAddVoteToFrom(types.VoteTypePrevote, cs1, cs2, cs1.ProposalBlock.Hash(), cs1.ProposalBlockParts.Header())

	// conflicting vote should cause us to broadcast dupeout tx on mempool
	txs := cs1.mempoolReactor.Mempool.GetProposalTxs()
	if len(txs) != 1 {
		t.Fatal("expected to find a transaction in the mempool after double signing")
	}
	dupeoutTx, ok := txs[0].(*types.DupeoutTx)
	if !ok {
		t.Fatal("expected to find DupeoutTx in mempool after double signing")
	}

	if !bytes.Equal(dupeoutTx.Address, cs2.privValidator.Address) {
		t.Fatalf("expected DupeoutTx for %X, got %X", cs2.privValidator.Address, dupeoutTx.Address)
	}

	// TODO: validate the sig
}

func TestSlashingPrecommits(t *testing.T) {
	css, _ := simpleConsensusState(2)
	cs1, cs2 := css[0], css[1]
	cs1.newStepCh = make(chan *RoundState) // so it blocks

	// start round and wait for propose and prevote
	cs1.EnterNewRound(cs1.Height, 0, false)
	_, _ = <-cs1.NewStepCh(), <-cs1.NewStepCh()

	// add prevote from cs2
	signAddVoteToFrom(types.VoteTypePrevote, cs1, cs2, cs1.ProposalBlock.Hash(), cs1.ProposalBlockParts.Header())

	// wait to finish precommit
	<-cs1.NewStepCh()

	// we should now be stuck in limbo forever, waiting for more prevotes
	// add one for a different block should cause us to go into prevote wait
	hash := cs1.ProposalBlock.Hash()
	hash[0] = byte(hash[0]+1) % 255
	signAddVoteToFrom(types.VoteTypePrecommit, cs1, cs2, hash, cs1.ProposalBlockParts.Header())

	// pass prevote wait
	<-cs1.NewStepCh()

	// NOTE: we have to send the vote for different block first so we don't just go into precommit round right
	// away and ignore more prevotes (and thus fail to slash!)

	// add precommit from cs2
	signAddVoteToFrom(types.VoteTypePrecommit, cs1, cs2, cs1.ProposalBlock.Hash(), cs1.ProposalBlockParts.Header())

	// conflicting vote should cause us to broadcast dupeout tx on mempool
	txs := cs1.mempoolReactor.Mempool.GetProposalTxs()
	if len(txs) != 1 {
		t.Fatal("expected to find a transaction in the mempool after double signing")
	}
	dupeoutTx, ok := txs[0].(*types.DupeoutTx)
	if !ok {
		t.Fatal("expected to find DupeoutTx in mempool after double signing")
	}

	if !bytes.Equal(dupeoutTx.Address, cs2.privValidator.Address) {
		t.Fatalf("expected DupeoutTx for %X, got %X", cs2.privValidator.Address, dupeoutTx.Address)
	}

	// TODO: validate the sig

}

//------------------------------------------------------------------------------------------
// CatchupSuite

//------------------------------------------------------------------------------------------
// HaltSuite

// 4 vals.
// we receive a final precommit after going into next round, but others might have gone to commit already!
func TestHalt1(t *testing.T) {
	css, privVals := simpleConsensusState(4)
	cs1, cs2, cs3, cs4 := css[0], css[1], css[2], css[3]
	cs1.newStepCh = make(chan *RoundState) // so it blocks

	timeoutChan := make(chan struct{})
	evsw := events.NewEventSwitch()
	evsw.OnStart()
	evsw.AddListenerForEvent("tester", types.EventStringTimeoutWait(), func(data types.EventData) {
		timeoutChan <- struct{}{}
	})
	cs1.SetFireable(evsw)

	// start round and wait for propose and prevote
	cs1.EnterNewRound(cs1.Height, 0, false)
	_, _ = <-cs1.NewStepCh(), <-cs1.NewStepCh()

	theBlockHash := cs1.ProposalBlock.Hash()

	donePrecommit := make(chan struct{})
	go func() {
		<-cs1.NewStepCh()
		donePrecommit <- struct{}{}
	}()
	signAddVoteToFromMany(types.VoteTypePrevote, cs1, cs1.ProposalBlock.Hash(), cs1.ProposalBlockParts.Header(), cs3, cs4)
	<-donePrecommit

	// the proposed block should now be locked and our precommit added
	validatePrecommit(t, cs1, 0, 0, privVals[0], theBlockHash, theBlockHash)

	donePrecommitWait := make(chan struct{})
	go func() {
		<-cs1.NewStepCh()
		donePrecommitWait <- struct{}{}
	}()
	// add precommits from the rest
	signAddVoteToFrom(types.VoteTypePrecommit, cs1, cs2, nil, types.PartSetHeader{}) // didnt receive proposal
	signAddVoteToFrom(types.VoteTypePrecommit, cs1, cs3, cs1.ProposalBlock.Hash(), cs1.ProposalBlockParts.Header())
	// we receive this later, but cs3 might receive it earlier and with ours will go to commit!
	precommit4 := signVote(cs4, types.VoteTypePrecommit, cs1.ProposalBlock.Hash(), cs1.ProposalBlockParts.Header())
	<-donePrecommitWait

	incrementRound(cs2, cs3, cs4)

	// timeout to new round
	<-timeoutChan

	log.Info("### ONTO ROUND 2")
	/*Round2
	// we timeout and prevote our lock
	// a polka happened but we didn't see it!
	*/

	// go to prevote, prevote for locked block
	_, _ = <-cs1.NewStepCh(), <-cs1.NewStepCh()
	validatePrevote(t, cs1, 0, privVals[0], cs1.LockedBlock.Hash())

	// now we receive the precommit from the previous round
	addVoteToFrom(cs1, cs4, precommit4)

	// receiving that precommit should take us straight to commit
	ensureNewStep(t, cs1)
	log.Warn("done enter commit!")

	// update to state
	ensureNewStep(t, cs1)

	if cs1.Height != 2 {
		t.Fatal("expected height to increment")
	}
}
