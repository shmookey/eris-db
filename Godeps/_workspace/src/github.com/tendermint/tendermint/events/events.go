package events

import (
	"sync"

	. "github.com/shmookey/eris-db/Godeps/_workspace/src/github.com/tendermint/tendermint/common"
	"github.com/shmookey/eris-db/Godeps/_workspace/src/github.com/tendermint/tendermint/types"
)

// reactors and other modules should export
// this interface to become eventable
type Eventable interface {
	SetFireable(Fireable)
}

// an event switch or cache implements fireable
type Fireable interface {
	FireEvent(event string, data types.EventData)
}

type EventSwitch struct {
	BaseService

	mtx        sync.RWMutex
	eventCells map[string]*eventCell
	listeners  map[string]*eventListener
}

func NewEventSwitch() *EventSwitch {
	evsw := &EventSwitch{}
	evsw.BaseService = *NewBaseService(log, "EventSwitch", evsw)
	return evsw
}

func (evsw *EventSwitch) OnStart() error {
	evsw.BaseService.OnStart()
	evsw.eventCells = make(map[string]*eventCell)
	evsw.listeners = make(map[string]*eventListener)
	return nil
}

func (evsw *EventSwitch) OnStop() {
	evsw.BaseService.OnStop()
	evsw.eventCells = nil
	evsw.listeners = nil
}

func (evsw *EventSwitch) AddListenerForEvent(listenerID, event string, cb eventCallback) {
	// Get/Create eventCell and listener
	evsw.mtx.Lock()
	eventCell := evsw.eventCells[event]
	if eventCell == nil {
		eventCell = newEventCell()
		evsw.eventCells[event] = eventCell
	}
	listener := evsw.listeners[listenerID]
	if listener == nil {
		listener = newEventListener(listenerID)
		evsw.listeners[listenerID] = listener
	}
	evsw.mtx.Unlock()

	// Add event and listener
	eventCell.AddListener(listenerID, cb)
	listener.AddEvent(event)
}

func (evsw *EventSwitch) RemoveListener(listenerID string) {
	// Get and remove listener
	evsw.mtx.RLock()
	listener := evsw.listeners[listenerID]
	delete(evsw.listeners, listenerID)
	evsw.mtx.RUnlock()

	if listener == nil {
		return
	}

	// Remove callback for each event.
	listener.SetRemoved()
	for _, event := range listener.GetEvents() {
		evsw.RemoveListenerForEvent(event, listenerID)
	}
}

func (evsw *EventSwitch) RemoveListenerForEvent(event string, listenerID string) {
	// Get eventCell
	evsw.mtx.Lock()
	eventCell := evsw.eventCells[event]
	evsw.mtx.Unlock()

	if eventCell == nil {
		return
	}

	// Remove listenerID from eventCell
	numListeners := eventCell.RemoveListener(listenerID)

	// Maybe garbage collect eventCell.
	if numListeners == 0 {
		// Lock again and double check.
		evsw.mtx.Lock()      // OUTER LOCK
		eventCell.mtx.Lock() // INNER LOCK
		if len(eventCell.listeners) == 0 {
			delete(evsw.eventCells, event)
		}
		eventCell.mtx.Unlock() // INNER LOCK
		evsw.mtx.Unlock()      // OUTER LOCK
	}
}

func (evsw *EventSwitch) FireEvent(event string, data types.EventData) {
	// Get the eventCell
	evsw.mtx.RLock()
	eventCell := evsw.eventCells[event]
	evsw.mtx.RUnlock()

	if eventCell == nil {
		return
	}

	// Fire event for all listeners in eventCell
	eventCell.FireEvent(data)
}

//-----------------------------------------------------------------------------

// eventCell handles keeping track of listener callbacks for a given event.
type eventCell struct {
	mtx       sync.RWMutex
	listeners map[string]eventCallback
}

func newEventCell() *eventCell {
	return &eventCell{
		listeners: make(map[string]eventCallback),
	}
}

func (cell *eventCell) AddListener(listenerID string, cb eventCallback) {
	cell.mtx.Lock()
	cell.listeners[listenerID] = cb
	cell.mtx.Unlock()
}

func (cell *eventCell) RemoveListener(listenerID string) int {
	cell.mtx.Lock()
	delete(cell.listeners, listenerID)
	numListeners := len(cell.listeners)
	cell.mtx.Unlock()
	return numListeners
}

func (cell *eventCell) FireEvent(data types.EventData) {
	cell.mtx.RLock()
	for _, listener := range cell.listeners {
		listener(data)
	}
	cell.mtx.RUnlock()
}

//-----------------------------------------------------------------------------

type eventCallback func(data types.EventData)

type eventListener struct {
	id string

	mtx     sync.RWMutex
	removed bool
	events  []string
}

func newEventListener(id string) *eventListener {
	return &eventListener{
		id:      id,
		removed: false,
		events:  nil,
	}
}

func (evl *eventListener) AddEvent(event string) {
	evl.mtx.Lock()
	defer evl.mtx.Unlock()

	if evl.removed {
		return
	}
	evl.events = append(evl.events, event)
}

func (evl *eventListener) GetEvents() []string {
	evl.mtx.RLock()
	defer evl.mtx.RUnlock()

	events := make([]string, len(evl.events))
	copy(events, evl.events)
	return events
}

func (evl *eventListener) SetRemoved() {
	evl.mtx.Lock()
	defer evl.mtx.Unlock()
	evl.removed = true
}
