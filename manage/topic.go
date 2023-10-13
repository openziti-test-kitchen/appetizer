package manage

import "github.com/sirupsen/logrus"

type Notifiable[T any] interface {
	Notify(msg T)
	Id() string
}

type TopicEntries[T any] struct {
	Id    string
	Entry Notifiable[T]
}

type TopicActions int

const (
	NOTIFY TopicActions = iota
	ADD
	REMOVE
)

type TopicAction[T any] struct {
	action TopicActions
	entry  Notifiable[T]
	msg    T
}

type Topic[T any] struct {
	entries []Notifiable[T]
	actions chan TopicAction[T]
}

var isRunning = false

func (t *Topic[T]) Start() {
	t.actions = make(chan TopicAction[T], 16)
	t.entries = make([]Notifiable[T], 0, 64)

	if !isRunning {
		go t.processActions()
		isRunning = true
	}
}

func (t *Topic[T]) processActions() {
	for {
		action := <-t.actions
		switch action.action {
		case ADD:
			logrus.Debugf("adding entry: %s", action.entry.Id())
			t.entries = append(t.entries, action.entry)
			logrus.Infof("added entry: %s. size now: %d", action.entry.Id(), len(t.entries))
			break
		case REMOVE:
			logrus.Debugf("removing entry: %s", action.entry.Id())
			t.removeEntry(action.entry)
			logrus.Infof("removed entry: %s. size now: %d", action.entry.Id(), len(t.entries))
			break
		case NOTIFY:
			for _, n := range t.entries {
				n.Notify(action.msg)
			}
			break
		}
	}
}

func (t *Topic[T]) removeEntry(action Notifiable[T]) {
	for a, b := range t.entries {
		if action.Id() == b.Id() {
			t.entries = append(t.entries[:a], t.entries[a+1:]...)
			return
		}
	}
	logrus.Warn("Attempt to remove entry with id %s failed?", action.Id())
}

func (t *Topic[T]) Close() {
	close(t.actions)
	isRunning = false
}

func (t *Topic[T]) AddReceiver(entry Notifiable[T]) {
	ta := TopicAction[T]{
		action: ADD,
		entry:  entry,
	}
	t.actions <- ta
}

func (t *Topic[T]) RemoveReceiver(entry Notifiable[T]) {
	ta := TopicAction[T]{
		action: REMOVE,
		entry:  entry,
	}
	t.actions <- ta
}

func (t *Topic[T]) Notify(m T) {
	ta := TopicAction[T]{
		action: NOTIFY,
		entry:  nil,
		msg:    m,
	}
	t.actions <- ta
}

func (t *Topic[T]) NewEntry(id string) *TopicEntry[T] {
	e := &TopicEntry[T]{
		identifier: id,
		Messages:   make(chan T, 16),
	}
	t.AddReceiver(e)
	return e
}

type TopicEntry[T any] struct {
	identifier string
	Messages   chan T
}

func (e *TopicEntry[T]) Id() string {
	return e.identifier
}

func (e TopicEntry[T]) Notify(msg T) {
	e.Messages <- msg
}
