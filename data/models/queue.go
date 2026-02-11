package models

import (
	"container/list"
)

type Queue struct {
	v *list.List
}

func NewQueue() *Queue {
	return &Queue{list.New()}
}

func (q *Queue) Push(v interface{}) {
	q.v.PushBack(v)
}

func (q *Queue) Pop() interface{} {
	front := q.v.Front()
	if front == nil {
		return nil
	}

	return q.v.Remove(front)
}

func (q *Queue) GetQueueData() *list.List {
	return q.v
}

func (q *Queue) Length() int {
	return q.v.Len()
}

func (q *Queue) IsEmpty() bool {
	return q.v.Len() == 0
}

func (q *Queue) Clear() {
	q.v.Init()
}

func (q *Queue) ToSlice() []any {
	slice := make([]any, q.v.Len())
	i := 0
	for e := q.v.Front(); e != nil; e = e.Next() {
		slice[i] = e.Value
		i++
	}
	return slice
}
