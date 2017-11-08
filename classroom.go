package main

import (
	"fmt"
	"github.com/goat-package/goat/goat"
	"math/rand"
)

func startInfrastructure(){
	goat.RunCentralServer(17000, make(chan struct{}), 0)
}

////////////////////////////////////////

type Teacher struct{
    comp *goat.Component
    subject string
}

func lessonPart(subject string, progr int) string{
	return fmt.Sprint(subject,progr)
}

func holdLesson(subject string) func (*goat.Process){
    return func (proc *goat.Process){
	    listeningToMe := goat.Equals(goat.Receiver("listening"), goat.Comp("subject"))
	    for i:=0; ; i++{
		    proc.WaitUntilTrue(func(attr *goat.Attributes) bool {
			    return attr.GetValue("questions") == 0
		    })
		    proc.Send(goat.NewTuple("lesson", lessonPart(subject, i)), listeningToMe)
	    }
	}
}

func answerQuestion(question string, asker int) func(proc *goat.Process) {
	return func(proc *goat.Process){
		answer := fmt.Sprint("Answer to "+question)
		isAsker := goat.Equals(goat.Receiver("id"), asker)
		proc.SendUpd(goat.NewTuple("answer", answer), isAsker, 
		    func(attr *goat.Attributes){
		        attr.Set("questions", attr.GetValue("questions").(int) - 1)
	        })
	}
}

func listenQuestions(proc *goat.Process){
	for{
		question := proc.Receive(func(attr *goat.Attributes, msg goat.Tuple) bool {
			if msg.IsLong(3) && msg.Get(0) == "question" {
				attr.Set("questions", attr.GetValue("questions").(int) + 1)
				return true
			}
			return false
		})
		questionTxt := question.Get(1).(string)
		asker := question.Get(2).(int)
		proc.Spawn(answerQuestion(questionTxt, asker))
	}
}

func createTeacher(subject string) *Teacher{
	environment := map[string]interface{}{
		"subject": subject,
		"questions": 0,
	}
	agent := goat.NewSingleServerAgent("127.0.0.1:17000")
	return &Teacher{goat.NewComponentWithAttributes(agent, environment), subject}
}

func (t *Teacher) start(){
	goat.NewProcess(t.comp).Run(func(proc *goat.Process){
		proc.Spawn(holdLesson(t.subject))
		proc.Call(listenQuestions)
	})
}

////////////////////////////////////////

type Student struct {
    id int
    comp *goat.Component
}

func listen(proc *goat.Process){
	for{
		lessonPart := proc.Receive(func(attr *goat.Attributes, msg goat.Tuple) bool {
			return msg.IsLong(2) && msg.Get(0) == "lesson"
		})
		fmt.Println("New lesson part: ", lessonPart.Get(1))
	}
}

func askQuestions(proc *goat.Process){
	for {
	    // Time to generate a question
	    proc.Sleep(int(rand.ExpFloat64() * 5000))
	    question := "question"
		myTeacher := goat.Equals(goat.Receiver("subject"), goat.Comp("listening"))
		proc.Send(goat.NewTuple("question", question, goat.Comp("id")), myTeacher)
		myId := 0
		answer := proc.Receive(func(attr *goat.Attributes, msg goat.Tuple) bool {
		    myId = attr.GetValue("id").(int)
			return msg.IsLong(2) && msg.Get(0) == "answer"
		})
		fmt.Printf("%d I asked: question and got: %s\n", myId, answer.Get(1).(string))
	}
}

func changeSubject(proc *goat.Process){
	for {
	    // Time to change subject
	    proc.Sleep(int(rand.ExpFloat64() * 10000))
	    var newSubject string
	    switch(rand.Intn(3)){
	        case 0:
	            newSubject = "chemistry"
            case 1:
                newSubject = "physics"
            case 2:
                newSubject = "math"
	    }
	    proc.WaitUntilTrue(func(attr *goat.Attributes) bool{
	        attr.Set("listening", newSubject)
	        return true
	    })
	}
}

func createStudent(id int, subject string) *Student{
	environment := map[string]interface{}{
		"listening": subject,
		"id": id,
	}
	agent := goat.NewSingleServerAgent("127.0.0.1:17000")
	return &Student{id, goat.NewComponentWithAttributes(agent, environment)}
}

func (s *Student) start(){
	goat.NewProcess(s.comp).Run(func(proc *goat.Process){
		proc.Spawn(listen)
		proc.Spawn(changeSubject)
		proc.Call(askQuestions)
	})
}

////////////////////////////////////////

func main(){
	startInfrastructure()
	teachers := []*Teacher{createTeacher("chemistry"), 
	    createTeacher("physics"), 
	    createTeacher("math")}
    students := []*Student{createStudent(0, "chemistry"),
        createStudent(1, "physics"),
        createStudent(2, "physics"),
        createStudent(3, "chemistry"),
        createStudent(4, "math")}
    
    for _, t := range teachers {
        t.start()
    }
    for _, s := range students {
        s.start()
    }
    
	<-make(chan struct{})
}
