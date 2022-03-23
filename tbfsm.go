package main

import (
	"github.com/albrow/zoom"
	tele "gopkg.in/telebot.v3"
)

type State int

const (
	StateStart State = iota
	StateRegistrationGetName
	StateRegistrationGetCode
	StateMainMenu
	StateNewQuestionStart
	StateNewQuestionContinue
	StateNewQuestionCommit
	StateAnswerStart
	StateAnswerContinue
	StateAnswerCommit
	StateFatalError
)

type TBFSM struct {
	UserID string
	State  State
}

var (
	TBFSMs *zoom.Collection
)

func (tbfsm *TBFSM) ModelID() string {
	return tbfsm.UserID
}

func (tbfsm *TBFSM) SetModelID(modelID string) {
	tbfsm.UserID = modelID
}

func newTBFSM(userid string) *TBFSM {
	return &TBFSM{
		UserID: userid,
		State:  StateStart,
	}
}

func CreateTBFSMs() {
	_TBFSMs, err := pool.NewCollectionWithOptions(&TBFSM{},
		zoom.DefaultCollectionOptions.WithIndex(true))
	if err != nil {
		// handle error
		panic(err)
	}

	TBFSMs = _TBFSMs
}

func ForceGetTBFSM(c tele.Context, menu *tele.ReplyMarkup) (*TBFSM, error) {
	userid := HexID(c.Sender())
	tbfsm := newTBFSM(userid)
	if err := TBFSMs.Find(userid, tbfsm); err != nil {
		tbfsm = newTBFSM(userid)
		if err := TBFSMs.Save(tbfsm); err != nil {
			return nil, c.Send("Error!", menu)
		}
		return tbfsm, nil
	}
	return tbfsm, nil
}

func SetTBFSMState(state State, c tele.Context, menu *tele.ReplyMarkup) error {
	tbfsm, err := ForceGetTBFSM(c, menu)
	if err != nil {
		return c.Send("Error!", menu)
	}

	tbfsm.State = state

	if err := TBFSMs.SaveFields([]string{"State"}, tbfsm); err != nil {
		return c.Send("Error!", menu)
	}

	return nil
}

func (tbfsm *TBFSM) SetTBFSMState(state State) error {
	tbfsm.State = state
	if err := TBFSMs.SaveFields([]string{"State"}, tbfsm); err != nil {
		return err
	}
	return nil
}

func GetTBFSM(id string) *TBFSM {
	tbfsm := newTBFSM(id)
	if err := TBFSMs.Find(id, tbfsm); err != nil {
		return nil
	}
	return tbfsm
}
