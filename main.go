package main

import (
	"fmt"
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
	//CreateCategories()
	CreateStudents()
	CreateTAs()
	CreateQuestions()

	pref := tele.Settings{
		Token:     os.Getenv("TOKEN"),
		Poller:    &tele.LongPoller{Timeout: 10 * time.Second},
		ParseMode: tele.ModeMarkdownV2,
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

		btnHome2 = tasMenu.Text("Home")
		btnNext  = tasMenu.Text("Fetch Next")
	)

	tasMenu.Reply(
		tasMenu.Row(btnNext),
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

	b.Handle("/skip", func(c tele.Context) error {
		if IsTA(c) {
			return botSkipAnswer(c, tasMenu)
		} else {
			return c.Send("Error\\!", studentsMenu)
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

	b.Handle(tele.OnMedia, func(c tele.Context) error {
		if IsTA(c) {
			return botAddToAnswer(c, tasMenu)
		} else {
			return botAddToQuestion(c, studentsMenu)
		}
	})

	b.Handle(tele.OnText, func(c tele.Context) error {
		if IsTA(c) {
			if c.Get("tbfsm").(*TBFSM).State == StateRegistrationGetName {
				return botGetNameTAs(c, tasMenu)
			} else if c.Get("tbfsm").(*TBFSM).State == StateRegistrationGetProb {
				return botGetProbTAs(c, tasMenu)
			} else {
				return botAddToAnswer(c, tasMenu)
			}
		} else {
			if c.Get("tbfsm").(*TBFSM).State == StateRegistrationGetName {
				return botGetNameStudents(c, studentsMenu)
			} else if c.Get("tbfsm").(*TBFSM).State == StateRegistrationGetCode {
				return botGetCodeStudents(c, studentsMenu)
			} else {
				return botAddToQuestion(c, studentsMenu)
			}
		}
	})

	b.Start()
}

func botGetCodeStudents(c tele.Context, menu *tele.ReplyMarkup) error {

	userid := HexID(c.Sender())
	student := GetStudent(userid)
	if student == nil {
		return c.Send("Error\\!")
	}

	student.Code = strings.TrimSpace(c.Message().Text)

	if err := Students.SaveFields([]string{"Code"}, student); err != nil {
		return err
	}

	return botStartStudents(c, menu)

}

func botGetProbTAs(c tele.Context, menu *tele.ReplyMarkup) error {

	userid := HexID(c.Sender())
	ta := GetTA(userid)
	if ta == nil {
		return c.Send("Error\\!")
	}

	prob, _ := strconv.ParseInt(strings.TrimSpace(c.Message().Text), 10, 32)

	if prob <= 0 || prob > 100 {
		return c.Send("Error\\!")
	}

	ta.Probability = int(prob)

	if err := TAs.SaveFields([]string{"Probability"}, ta); err != nil {
		return err
	}

	return botStartTAs(c, menu)
}

func botGetNameStudents(c tele.Context, menu *tele.ReplyMarkup) error {
	if err := c.Get("tbfsm").(*TBFSM).SetTBFSMState(StateRegistrationGetCode); err != nil {
		return err
	}

	userid := HexID(c.Sender())
	student := GetStudent(userid)
	if student == nil {
		return c.Send("Error\\!")
	}

	student.Name = strings.TrimSpace(c.Message().Text)

	if err := Students.SaveFields([]string{"Name"}, student); err != nil {
		return err
	}

	return c.Send("Enter your student id\\.")

}

func botGetNameTAs(c tele.Context, menu *tele.ReplyMarkup) error {
	if err := c.Get("tbfsm").(*TBFSM).SetTBFSMState(StateRegistrationGetProb); err != nil {
		return err
	}

	userid := HexID(c.Sender())
	ta := GetTA(userid)
	if ta == nil {
		return c.Send("Error\\!")
	}

	ta.Name = strings.TrimSpace(c.Message().Text)

	if err := TAs.SaveFields([]string{"Name"}, ta); err != nil {
		return err
	}

	return c.Send("Enter your dispatch probability percentage \\(from 1\\-100\\)\\.")
}

func botStartStudents(c tele.Context, menu *tele.ReplyMarkup) error {
	if err := c.Get("tbfsm").(*TBFSM).SetTBFSMState(StateMainMenu); err != nil {
		return err
	}

	userid := HexID(c.Sender())
	student := GetStudent(userid)
	if student == nil {
		t := newStudent(userid)
		//t.Name = strings.TrimSpace(c.Sender().FirstName + " " + c.Sender().LastName)
		err := Students.Save(t)
		if err != nil {
			return err
		}
		student = t
	}

	if student.Name == "" {
		// Lets register!
		if err := c.Get("tbfsm").(*TBFSM).SetTBFSMState(StateRegistrationGetName); err != nil {
			return err
		}
		return c.Send("Enter your complete name\\.")
	}

	return c.Send("Select one option\\.", menu)
}

func botStartTAs(c tele.Context, menu *tele.ReplyMarkup) error {
	if err := c.Get("tbfsm").(*TBFSM).SetTBFSMState(StateMainMenu); err != nil {
		return err
	}

	userid := HexID(c.Sender())
	ta := GetTA(userid)
	if ta == nil {
		t := newTA(userid)
		//t.Name = strings.TrimSpace(c.Sender().FirstName + " " + c.Sender().LastName)
		err := TAs.Save(t)
		if err != nil {
			return err
		}
		ta = t
	}

	if ta.Name == "" {
		// Lets register!
		if err := c.Get("tbfsm").(*TBFSM).SetTBFSMState(StateRegistrationGetName); err != nil {
			return err
		}
		return c.Send("Enter your complete name\\.")
	}

	return c.Send("Select one option\\.", menu)
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
	return c.Send("Enter your question context\\.")
}

func botAddToQuestion(c tele.Context, menu *tele.ReplyMarkup) error {
	userid := HexID(c.Sender())
	t, err := TryGetStudent(c, menu)
	if t == nil || err != nil {
		return err
	}
	if t.CurrentQAID == "" {
		return c.Send("Error\\!", menu)
	}
	q := newQuestion()
	q.StudentID = userid
	q.SetModelID(t.CurrentQAID)
	if err := Questions.Find(q.ModelID(), q); err != nil {
		return c.Send("Error\\!", menu)
	}

	if c.Get("tbfsm").(*TBFSM).State == StateNewQuestionStart {
		if err := c.Get("tbfsm").(*TBFSM).SetTBFSMState(StateNewQuestionContinue); err != nil {
			return err
		}
		q.Context = strings.TrimSpace(c.Message().Text)

		if err := Questions.SaveFields([]string{"Context"}, q); err != nil {
			return err
		}

		return c.Send("Now, Send your question\\.")
	} else {
		q.Messages = append(q.Messages, c.Message())

		if err := Questions.SaveFields([]string{"Messages"}, q); err != nil {
			return err
		}

		return c.Send("Send more, or send /done to finish\\.\\.\\.")
	}

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
		return c.Send("Error\\!", menu)
	}
	q := newQuestion()
	q.StudentID = userid
	q.SetModelID(ta.CurrentQAID)
	if err := Questions.Find(q.ModelID(), q); err != nil {
		return c.Send("Error\\!", menu)
	}

	q.Answers = append(q.Answers, c.Message())

	if err := Questions.SaveFields([]string{"Answers"}, q); err != nil {
		return err
	}

	return c.Send("Send more, or send /done to finish\\.\\.\\.")
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
		if err := c.Send("No more questions available\\."); err != nil {
			return err
		}
		return botStartTAs(c, menu)
	}

	var nextQ string
	nextQ, ta.Questions = ta.Questions[0], ta.Questions[1:]
	q := newQuestion()
	q.SetModelID(nextQ)
	if err := Questions.Find(q.ModelID(), q); err != nil {
		return c.Send("Error\\!", menu)
	}

	ta.CurrentQAID = nextQ

	if err := TAs.SaveFields([]string{"CurrentQAID", "Questions"}, ta); err != nil {
		return err
	}

	student := q.GetStudent()
	if err := c.Send(fmt.Sprintf("_From:_ %v \\(%v\\)\n_Context:_ %v", student.Name, student.Code, q.Context)); err != nil {
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

	return c.Send("Answer this question now, or send /skip to pass this question to another TA\\!")

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
		return c.Send("Error\\!", menu)
	}
	q := newQuestion()
	q.StudentID = userid
	q.SetModelID(t.CurrentQAID)
	if err := Questions.Find(q.ModelID(), q); err != nil {
		return c.Send("Error\\!", menu)
	}

	// Now, time to dispatch this QA!
	ta, _, err := DispatchTA()
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
	ta.AssignedCount++

	if err := Students.SaveFields([]string{"Questions", "CurrentQAID"}, t); err != nil {
		return err
	}
	if err := TAs.SaveFields([]string{"Questions", "AssignedCount"}, ta); err != nil {
		return err
	}

	taIntID, err := c.Bot().ChatByID(IntID(ta.UserID))
	if err != nil {
		return err
	}

	if _, err := c.Bot().Send(taIntID, "A student is waiting for your answer\\."); err != nil {
		return err
	}

	if err := c.Send("Done\\!"); err != nil {
		return err
	}

	return botStartStudents(c, menu)
}

func botSkipAnswer(c tele.Context, menu *tele.ReplyMarkup) error {
	if err := c.Get("tbfsm").(*TBFSM).SetTBFSMState(StateAnswerSkip); err != nil {
		return err
	}

	ta, err := TryGetTA(c, menu)
	if ta == nil || err != nil {
		return err
	}
	if ta.CurrentQAID == "" {
		return c.Send("Error\\!", menu)
	}
	q := newQuestion()
	q.SetModelID(ta.CurrentQAID)
	if err := Questions.Find(q.ModelID(), q); err != nil {
		return c.Send("Error\\!", menu)
	}

	// Skip this QA for current TA...
	ta.CurrentQAID = ""
	if err := TAs.SaveFields([]string{"CurrentQAID"}, ta); err != nil {
		return err
	}

	// Now, time to redispatch this QA!
	newTA, count, err := DispatchTA()
	if err != nil {
		return err
	}

	// Make sure the newly selected TA is not the current one!
	for count > 1 && newTA.UserID == ta.UserID {
		newTA, _, err = DispatchTA()
		if err != nil {
			return err
		}
	}

	q.TAID = newTA.UserID

	if err := Questions.SaveFields([]string{"TAID"}, q); err != nil {
		return err
	}

	newTA.Questions = append(newTA.Questions, q.ModelID())
	newTA.AssignedCount++

	if err := TAs.SaveFields([]string{"Questions", "AssignedCount"}, newTA); err != nil {
		return err
	}

	newTAIntID, err := c.Bot().ChatByID(IntID(newTA.UserID))
	if err != nil {
		return err
	}

	if _, err := c.Bot().Send(newTAIntID, "A student is waiting for your answer\\."); err != nil {
		return err
	}

	if err := c.Send("Skipped\\!"); err != nil {
		return err
	}

	return botStartTAs(c, menu)

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
		return c.Send("Error\\!", menu)
	}
	q := newQuestion()
	q.SetModelID(ta.CurrentQAID)
	if err := Questions.Find(q.ModelID(), q); err != nil {
		return c.Send("Error\\!", menu)
	}

	// Now, time to dispatch this answer to its questioner!
	ta.CurrentQAID = ""
	ta.AnsweredCount++
	senderIntID, err := c.Bot().ChatByID(IntID(q.StudentID))
	if err != nil {
		return err
	}

	if _, err := c.Bot().Send(senderIntID, fmt.Sprintf("A TA has been answered your question:\n_Context:_ %v", q.Context)); err != nil {
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

	if err := TAs.SaveFields([]string{"CurrentQAID", "AnsweredCount"}, ta); err != nil {
		return err
	}

	if err := c.Send("Done\\!"); err != nil {
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
