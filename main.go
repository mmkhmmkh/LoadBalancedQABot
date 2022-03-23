package main

import (
	"github.com/albrow/zoom"
	tele "gopkg.in/telebot.v3"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

var pool *zoom.Pool

func main() {
	pool = zoom.NewPool("localhost:6379")
	defer func() {
		if err := pool.Close(); err != nil {
			// handle error
			panic(err)
		}
	}()

	CreateTBFSMs()
	CreateCategories()
	CreateStudents()
	CreateTAs()
	CreateQuestions()

	pref := tele.Settings{
		Token:  os.Getenv("TOKEN"),
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		log.Fatal(err)
		return
	}

	var (
		studentsMenu = &tele.ReplyMarkup{ResizeKeyboard: true}
		tasMenu      = &tele.ReplyMarkup{ResizeKeyboard: true}

		btnHome = studentsMenu.Text("Home")
		btnNew  = studentsMenu.Text("New Question")

		btnHome2             = tasMenu.Text("Home")
		btnNext              = tasMenu.Text("Fetch Next")
		btnCategoriesManager = tasMenu.Text("Manage Categories")
	)

	tasMenu.Reply(
		tasMenu.Row(btnNext),
		tasMenu.Row(btnCategoriesManager),
		tasMenu.Row(btnHome2),
	)

	studentsMenu.Reply(
		studentsMenu.Row(btnNew),
		studentsMenu.Row(btnHome),
	)

	b.Use(func(hf tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			tbfsm, err := ForceGetTBFSM(c)
			if err != nil {
				return err
			}
			c.Set("tbfsm", tbfsm)
			return hf(c)
		}
	})

	b.Handle("/start", func(c tele.Context) error {
		if c.Message().Payload == os.Getenv("TA_PAYLOAD") || IsTA(c) {
			return botStartTAs(c, tasMenu)
		} else {
			return botStartStudents(c, studentsMenu)
		}
	})

	b.Handle("/done", func(c tele.Context) error {
		if IsTA(c) {
			return botCommitAnswer(c, tasMenu)
		} else {
			return botCommitQuestion(c, studentsMenu)
		}
	})

	b.Handle(&btnHome, func(c tele.Context) error {
		if c.Message().Payload == os.Getenv("TA_PAYLOAD") || IsTA(c) {
			return botStartTAs(c, tasMenu)
		} else {
			return botStartStudents(c, studentsMenu)
		}
	})

	b.Handle(&btnNew, func(c tele.Context) error {
		return botNewQuestion(c, studentsMenu)
	})

	b.Handle(&btnNext, func(c tele.Context) error {
		return botNextAnswer(c, tasMenu)
	})

	b.Handle(&btnCategoriesManager, func(c tele.Context) error {
		return botCategoriesManager(c, tasMenu)
	})

	b.Handle(tele.OnMedia, func(c tele.Context) error {
		if IsTA(c) {
			return botAddToAnswer(c, tasMenu)
		} else {
			return botAddToQuestion(c, studentsMenu)
		}
	})

	b.Handle(tele.OnText, func(c tele.Context) error {
		if IsTA(c) {
			return botAddToAnswer(c, tasMenu)
		} else {
			return botAddToQuestion(c, studentsMenu)
		}
	})

	b.Start()
}

func botCategoriesManager(c tele.Context, menu *tele.ReplyMarkup) error {
	return nil
}

func botStartStudents(c tele.Context, menu *tele.ReplyMarkup) error {
	if err := c.Get("tbfsm").(*TBFSM).SetTBFSMState(StateStart); err != nil {
		return err
	}

	userid := HexID(c.Sender())
	t := newStudent(userid)
	if _, err := Students.Delete(userid); err != nil {
		return err
	}
	t.Name = strings.TrimSpace(c.Sender().FirstName + " " + c.Sender().LastName)
	err := Students.Save(t)
	if err != nil {
		return err
	}

	return c.Send("Select one option.", menu)
}

func botStartTAs(c tele.Context, menu *tele.ReplyMarkup) error {
	if err := c.Get("tbfsm").(*TBFSM).SetTBFSMState(StateStart); err != nil {
		return err
	}

	userid := HexID(c.Sender())

	t := newTA(userid)
	if _, err := TAs.Delete(userid); err != nil {
		return err
	}
	t.Name = strings.TrimSpace(c.Sender().FirstName + " " + c.Sender().LastName)
	err := TAs.Save(t)
	if err != nil {
		return err
	}

	return c.Send("Select one option.", menu)
}

func botNewQuestion(c tele.Context, menu *tele.ReplyMarkup) error {
	if err := c.Get("tbfsm").(*TBFSM).SetTBFSMState(StateNewQuestionStart); err != nil {
		return err
	}

	userid := HexID(c.Sender())
	t, err := TryGetStudent(c, menu)
	if t == nil || err != nil {
		return err
	}
	q := newQuestion()
	q.StudentID = userid
	if err := Questions.Save(q); err != nil {
		return err
	}
	t.CurrentQAID = q.ModelID()
	if err := Students.SaveFields([]string{"CurrentQAID"}, t); err != nil {
		return err
	}
	return c.Send("Enter your question.", menu)
}

func botAddToQuestion(c tele.Context, menu *tele.ReplyMarkup) error {
	if err := c.Get("tbfsm").(*TBFSM).SetTBFSMState(StateNewQuestionContinue); err != nil {
		return err
	}

	userid := HexID(c.Sender())
	t, err := TryGetStudent(c, menu)
	if t == nil || err != nil {
		return err
	}
	if t.CurrentQAID == "" {
		return c.Send("Error!", menu)
	}
	q := newQuestion()
	q.StudentID = userid
	q.SetModelID(t.CurrentQAID)
	if err := Questions.Find(q.ModelID(), q); err != nil {
		return c.Send("Error!", menu)
	}

	q.Messages = append(q.Messages, c.Message())

	if err := Questions.SaveFields([]string{"Messages"}, q); err != nil {
		return err
	}

	return c.Send("Send more, or send /done to finish...")
}

func botAddToAnswer(c tele.Context, menu *tele.ReplyMarkup) error {
	if err := c.Get("tbfsm").(*TBFSM).SetTBFSMState(StateAnswerContinue); err != nil {
		return err
	}

	userid := HexID(c.Sender())
	ta, err := TryGetTA(c, menu)
	if ta == nil || err != nil {
		return err
	}
	if ta.CurrentQAID == "" {
		return c.Send("Error!", menu)
	}
	q := newQuestion()
	q.StudentID = userid
	q.SetModelID(ta.CurrentQAID)
	if err := Questions.Find(q.ModelID(), q); err != nil {
		return c.Send("Error!", menu)
	}

	q.Answers = append(q.Answers, c.Message())

	if err := Questions.SaveFields([]string{"Answers"}, q); err != nil {
		return err
	}

	return c.Send("Send more, or send /done to finish...")
}

func botNextAnswer(c tele.Context, menu *tele.ReplyMarkup) error {
	if err := c.Get("tbfsm").(*TBFSM).SetTBFSMState(StateAnswerStart); err != nil {
		return err
	}

	ta, err := TryGetTA(c, menu)
	if ta == nil || err != nil {
		return err
	}

	if len(ta.Questions) == 0 {
		if err := c.Send("No more questions available."); err != nil {
			return err
		}
		return botStartTAs(c, menu)
	}

	var nextQ string
	nextQ, ta.Questions = ta.Questions[0], ta.Questions[1:]
	q := newQuestion()
	q.SetModelID(nextQ)
	if err := Questions.Find(q.ModelID(), q); err != nil {
		return c.Send("Error!", menu)
	}

	ta.CurrentQAID = nextQ

	if err := TAs.SaveFields([]string{"CurrentQAID", "Questions"}, ta); err != nil {
		return err
	}

	for _, m := range q.Messages {
		switch obj := m.(type) {
		case *tele.Message:
			if err := c.Forward(obj); err != nil {
				return err
			}
		case tele.Message:
			if err := c.Forward(&obj); err != nil {
				return err
			}
		}

	}

	return c.Send("Answer this question now!")

}

func botCommitQuestion(c tele.Context, menu *tele.ReplyMarkup) error {
	if err := c.Get("tbfsm").(*TBFSM).SetTBFSMState(StateNewQuestionCommit); err != nil {
		return err
	}

	userid := HexID(c.Sender())
	t, err := TryGetStudent(c, menu)
	if t == nil || err != nil {
		return err
	}
	if t.CurrentQAID == "" {
		return c.Send("Error!", menu)
	}
	q := newQuestion()
	q.StudentID = userid
	q.SetModelID(t.CurrentQAID)
	if err := Questions.Find(q.ModelID(), q); err != nil {
		return c.Send("Error!", menu)
	}

	// Now, time to dispatch this QA!
	ta, err := DispatchTA()
	if err != nil {
		return err
	}
	q.TAID = ta.UserID

	if err := Questions.SaveFields([]string{"TAID"}, q); err != nil {
		return err
	}

	t.Questions = append(t.Questions, q.ModelID())
	t.CurrentQAID = ""
	ta.Questions = append(ta.Questions, q.ModelID())

	if err := Students.SaveFields([]string{"Questions", "CurrentQAID"}, t); err != nil {
		return err
	}
	if err := TAs.SaveFields([]string{"Questions"}, ta); err != nil {
		return err
	}

	if err := c.Send("Done!"); err != nil {
		return err
	}

	return botStartStudents(c, menu)
}

func botCommitAnswer(c tele.Context, menu *tele.ReplyMarkup) error {
	if err := c.Get("tbfsm").(*TBFSM).SetTBFSMState(StateAnswerCommit); err != nil {
		return err
	}

	ta, err := TryGetTA(c, menu)
	if ta == nil || err != nil {
		return err
	}
	if ta.CurrentQAID == "" {
		return c.Send("Error!", menu)
	}
	q := newQuestion()
	q.SetModelID(ta.CurrentQAID)
	if err := Questions.Find(q.ModelID(), q); err != nil {
		return c.Send("Error!", menu)
	}

	// Now, time to dispatch this answer to its questioner!
	ta.CurrentQAID = ""
	senderIntID, err := c.Bot().ChatByID(IntID(q.StudentID))
	if err != nil {
		return err
	}
	for _, m := range q.Answers {
		switch obj := m.(type) {
		case *tele.Message:
			if _, err := c.Bot().Copy(senderIntID, obj); err != nil {
				return err
			}
		case tele.Message:
			if _, err := c.Bot().Copy(senderIntID, &obj); err != nil {
				return err
			}
		}
	}

	if err := TAs.SaveFields([]string{"CurrentQAID"}, ta); err != nil {
		return err
	}

	if err := c.Send("Done!"); err != nil {
		return err
	}

	return botStartTAs(c, menu)
}

func HexID(u *tele.User) string {
	return strconv.FormatInt(u.ID, 16)
}

func IntID(id string) int64 {
	parseInt, err := strconv.ParseInt(id, 16, 64)
	if err != nil {
		panic(err)
	}
	return parseInt
}
