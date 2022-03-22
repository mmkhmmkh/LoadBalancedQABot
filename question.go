package main

import (
	"encoding/gob"
	"github.com/albrow/zoom"
	tele "gopkg.in/telebot.v3"
)

type Question struct {
	zoom.RandomID
	SenderID string
	TAID     string
	Messages []any
	Answers  []any
}

var (
	Questions *zoom.Collection
)

func newQuestion() *Question {
	return &Question{
		SenderID: "",
		TAID:     "",
		Messages: []any{},
		Answers:  []any{},
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

func (q *Question) GetSender() *Student {
	return GetStudent(q.SenderID)
}

func (q *Question) GetTA() *TA {
	return GetTA(q.TAID)
}
