package models

import "fmt"

type Todo struct {
	Do     string
	Object any
}

func (t *Todo) ToString() string {
	return "Do: " + t.Do + " Object: " + fmt.Sprint(t.Object)
}
