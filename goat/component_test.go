// component_test.go
package goat

import (
	"testing"
)

func initTest(timeout int64) (chan struct{}, *CentralServer) {
	// Launching a simple server
	// TODO add support to change server type

	term := make(chan struct{}) //signals when no messages have been exchanged for some timeout
	srv := RunCentralServer(17654, term, timeout)
	return term, srv
}

func teardownTest(t chan struct{}, srv *CentralServer) {
	// waits until the server ends
	<-t
	srv.Terminate()
}

func TestComponentEmpty(t *testing.T) {
	defer teardownTest(initTest(200))
	comp := NewComponent("127.0.0.1:17654")
	run := false
	NewProcess(comp).Run(func(*Process) {
		run = true
	})
	defer func() {
		if !run {
			t.Fail()
		}
	}()
}

func TestTwoComponentEmpty(t *testing.T) {
	chn, srv := initTest(200)
	run1 := false
	comp1 := NewComponent("127.0.0.1:17654")
	comp2 := NewComponent("127.0.0.1:17654")
	NewProcess(comp1).Run(func(*Process) {
		run1 = true
	})
	run2 := false
	NewProcess(comp2).Run(func(*Process) {
		run2 = true
	})
	defer func() {
		teardownTest(chn, srv)
		if !run1 || !run2 {
			t.Fail()
		}
	}()
}

type Foo struct {
    Dog string
    Cat string
    Fish string
    Rat []string
    Monkey int
};

func TestSendReceiveObject(t *testing.T) {
	chn, srv := initTest(200)
	sendOb := Foo{
	    Dog : "bark",
	    Cat : "meoww",
	    Fish : "",
	    Rat: []string{"squit"},
	    Monkey: 5,
	}
	var recOb Foo;
	sent := false
	received := false
	comp1 := NewComponent("127.0.0.1:17654")
	comp2 := NewComponent("127.0.0.1:17654")
	NewProcess(comp1).Run(func(p *Process) {
		p.SendObject(func(*Attributes) (interface{}, Predicate, bool) {
			return sendOb, True{}, true
		})
		sent = true
	})
	NewProcess(comp2).Run(func(p *Process) {
		p.ReceiveObject(NoPre(), func(attr *Attributes) bool {
			return recOb.Dog == sendOb.Dog && recOb.Cat == sendOb.Cat && 
			    recOb.Fish == sendOb.Fish && recOb.Monkey == sendOb.Monkey &&
			    len(recOb.Rat) == 1 && recOb.Rat[0] == sendOb.Rat[0]  
		}, &recOb)
		received = true
	})
	defer func() {
		teardownTest(chn, srv)
		if !sent || !received {
			t.Fail()
		}
	}()
}

func TestSendReceive(t *testing.T) {
	chn, srv := initTest(200)
	sent := false
	received := false
	comp1 := NewComponent("127.0.0.1:17654")
	comp2 := NewComponent("127.0.0.1:17654")
	NewProcess(comp1).Run(func(p *Process) {
		p.Send(func(*Attributes) (string, Predicate, bool) {
			return "Ciao", True{}, true
		})
		sent = true
	})
	NewProcess(comp2).Run(func(p *Process) {
		p.Receive(NoPre(), func(attr *Attributes, msg string) bool {
			return msg == "Ciao"
		})
		received = true
	})
	defer func() {
		teardownTest(chn, srv)
		if !sent || !received {
			t.Fail()
		}
	}()
}

func TestSendTwoReceive(t *testing.T) {
	chn, srv := initTest(200)
	sent := false
	received2 := false
	received3 := false
	comp1 := NewComponent("127.0.0.1:17654")
	comp2 := NewComponent("127.0.0.1:17654")
	comp3 := NewComponent("127.0.0.1:17654")
	NewProcess(comp1).Run(func(p *Process) {
		p.Send(func(*Attributes) (string, Predicate, bool) {
			return "Ciao", True{}, true
		})
		sent = true
	})
	NewProcess(comp2).Run(func(p *Process) {
		p.Receive(NoPre(), func(attr *Attributes, msg string) bool {
			return msg == "Ciao"
		})
		received2 = true
	})
	NewProcess(comp3).Run(func(p *Process) {
		p.Receive(NoPre(), func(attr *Attributes, msg string) bool {
			return msg == "Ciao"
		})
		received3 = true
	})
	defer func() {
		teardownTest(chn, srv)
		if !sent || !received2 || !received3 {
			t.Fail()
		}
	}()
}

func TestSendTwoReceiveOneAcceptThenTheOther(t *testing.T) {
	chn, srv := initTest(200)
	sent := false
	received2 := false
	received3 := false
	comp1 := NewComponent("127.0.0.1:17654")
	comp2 := NewComponent("127.0.0.1:17654")
	comp3 := NewComponent("127.0.0.1:17654")
	NewProcess(comp1).Run(func(p *Process) {
		p.Send(func(*Attributes) (string, Predicate, bool) {
			return "Ciao", True{}, true
		})
		sent = true
	})
	NewProcess(comp2).Run(func(p *Process) {
		p.Receive(NoPre(), func(attr *Attributes, msg string) bool {
			return msg == "Ciao"
		})
		received2 = true
		p.Send(func(*Attributes) (string, Predicate, bool) {
			return "Ciaone", True{}, true
		})
	})
	NewProcess(comp3).Run(func(p *Process) {
		p.Receive(NoPre(), func(attr *Attributes, msg string) bool {
			return msg == "Ciaone"
		})
		if !received2 {
			t.Error("comp2 must have received before me!")
		}
		received3 = true
	})
	defer func() {
		teardownTest(chn, srv)
		if !sent || !received2 || !received3 {
			t.Fail()
		}
	}()
}
