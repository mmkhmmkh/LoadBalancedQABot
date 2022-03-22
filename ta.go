package main

import (
	"github.com/albrow/zoom"
	_ "gopkg.in/telebot.v3"
	tele "gopkg.in/telebot.v3"
	"math/rand"
)

type TA struct {
	UserID      string
	Name        string
	Probability int
	CurrentQAID string
	Questions   []string
}

var (
	TAs *zoom.Collection
)

func (ta *TA) ModelID() string {
	return ta.UserID
}

func (ta *TA) SetModelID(modelID string) {
	ta.UserID = modelID
}

func newTA(userid string) *TA {
	return &TA{
		UserID:      userid,
		Name:        "",
		Probability: 100,
		CurrentQAID: "",
		Questions:   []string{},
	}
}

func CreateTAs() {
	_TAs, err := pool.NewCollectionWithOptions(&TA{},
		zoom.DefaultCollectionOptions.WithIndex(true))
	if err != nil {
		// handle error
		panic(err)
	}

	TAs = _TAs
}

func DispatchTA() (*TA, error) {
	count, err := TAs.Count()
	if err != nil {
		return nil, err
	}

	index := rand.Intn(count)
	var tas []*TA
	query := TAs.NewQuery().Offset(uint(index)).Limit(1)
	if err := query.Run(&tas); err != nil || len(tas) != 1 {
		return nil, err
	}
	selectedTA := tas[0]
	for rand.Intn(100) > selectedTA.Probability {
		index = rand.Intn(count)
		query = TAs.NewQuery().Offset(uint(index)).Limit(1)
		if err := query.Run(&tas); err != nil || len(tas) != 1 {
			return nil, err
		}
		selectedTA = tas[0]
	}

	return selectedTA, nil
}

func TryGetTA(c tele.Context, menu *tele.ReplyMarkup) (*TA, error) {
	userid := HexID(c.Sender())
	ta := newTA(userid)
	if err := TAs.Find(userid, ta); err != nil {
		return nil, c.Send("Error!", menu)
	}
	return ta, nil
}

func GetTA(id string) *TA {
	ta := newTA(id)
	if err := TAs.Find(id, ta); err != nil {
		return nil
	}
	return ta
}

func IsTA(c tele.Context) bool {
	userid := HexID(c.Sender())
	ta := newTA(userid)
	if err := TAs.Find(userid, ta); err != nil {
		return false
	}
	return ta != nil
}
