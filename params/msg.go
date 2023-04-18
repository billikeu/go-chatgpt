package params

import (
	"container/list"
	"errors"
	"sync"
)

type Msg interface {
	ID() string
}

type MsgQueue struct {
	queue *list.List
	sync.Mutex
}

func NewMsgQueue() *MsgQueue {
	return &MsgQueue{queue: list.New()}
}

func (mq *MsgQueue) Enqueue(msg Msg) {
	mq.Lock()
	defer mq.Unlock()

	mq.queue.PushBack(msg)
}

func (mq *MsgQueue) Len() int {
	mq.Lock()
	defer mq.Unlock()

	return mq.queue.Len()
}

func (mq *MsgQueue) Dequeue() (Msg, error) {
	mq.Lock()
	defer mq.Unlock()

	if mq.queue.Len() == 0 {
		return nil, errors.New("queue is empty")
	}
	elem := mq.queue.Front()
	mq.queue.Remove(elem)
	return elem.Value.(Msg), nil
}
