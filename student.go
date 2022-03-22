package main

import (
	"github.com/albrow/zoom"
	_ "gopkg.in/telebot.v3"
	tele "gopkg.in/telebot.v3"
)

type Student struct {
	UserID      string
	Code        string
	Name        string
	CurrentQAID string
	Questions   []string
}

var (
	Students *zoom.Collection
)

func (student *Student) ModelID() string {
	return student.UserID
}

func (student *Student) SetModelID(modelID string) {
	student.UserID = modelID
}

func newStudent(userid string) *Student {
	return &Student{
		UserID:      userid,
		Code:        "",
		Name:        "",
		CurrentQAID: "",
		Questions:   []string{},
	}
}

func CreateStudents() {
	_Students, err := pool.NewCollectionWithOptions(&Student{},
		zoom.DefaultCollectionOptions.WithIndex(true))
	if err != nil {
		// handle error
		panic(err)
	}

	Students = _Students
}

func TryGetStudent(c tele.Context, menu *tele.ReplyMarkup) (*Student, error) {
	userid := HexID(c.Sender())
	t := newStudent(userid)
	if err := Students.Find(userid, t); err != nil {
		return nil, c.Send("Error!", menu)
	}
	return t, nil
}

func GetStudent(id string) *Student {
	t := newStudent(id)
	if err := Students.Find(id, t); err != nil {
		return nil
	}
	return t
}

func IsStudent(c tele.Context) bool {
	userid := HexID(c.Sender())
	t := newStudent(userid)
	if err := Students.Find(userid, t); err != nil {
		return false
	}
	return t != nil
}
