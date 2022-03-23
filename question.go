package main

import (
	"encoding/gob"
	"github.com/albrow/zoom"
	tele "gopkg.in/telebot.v3"
)

type Question struct {
	zoom.RandomID
	StudentID string
	TAID      string
	Context   string
	Messages  []any
	Answers   []any
}

var (
	Questions *zoom.Collection
)

func newQuestion() *Question {
	return &Question{
		StudentID: "",
		TAID:      "",
		Messages:  []any{},
		Answers:   []any{},
	}
}

func CreateQuestions() {
	gob.Register(tele.Message{})
	_Questions, err := pool.NewCollectionWithOptions(&Question{},
		zoom.DefaultCollectionOptions.WithIndex(true))
	if err != nil {
		// handle error
		panic(err)
	}

	Questions = _Questions
}

func (q *Question) GetStudent() *Student {
	return GetStudent(q.StudentID)
}

func (q *Question) GetTA() *TA {
	return GetTA(q.TAID)
}

//
//func (q *Question) GetCategory() *Category {
//	return GetCategory(q.CategoryID)
//}
