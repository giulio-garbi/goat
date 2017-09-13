// component_test.go
package goat

import (
	"testing"
	"fmt"
)

func initTestCS(timeout int64) (chan struct{}, *CentralServer) {
	// Launching a simple server
	// TODO add support to change server type

	term := make(chan struct{}) //signals when no messages have been exchanged for some timeout
	srv := RunCentralServer(17654, term, timeout)
	return term, srv
}

func teardownTestCS(t chan struct{}, srv *CentralServer) {
	// waits until the server ends
	<-t
	srv.Terminate()
}

type testClusterInfrastructure struct{
    nodes []*ClusterNode
    agents []*ClusterAgent
    registration *ClusterAgentRegistration
    msgQ *ClusterMessageQueue
    counter *ClusterCounter
    terms []chan struct{}
}

func (tci *testClusterInfrastructure) initTest(timeout int64, clusterSize int, componentNbr int) {
	// Launching a clustered infrastructure
	tci.terms = make([]chan struct{}, 3 + clusterSize)
	
	msgQAddr := "127.0.0.1:17999"
	counterAddr := "127.0.0.1:17998"
	registrationAddr := "127.0.0.1:17997"
	nodesAddr := make([]string, clusterSize)
	for i:=0; i<clusterSize; i++{
	    nodesAddr[i] = fmt.Sprintf("127.0.0.1:%d",18000+i)
	    tci.terms[i+3] = make(chan struct{})
	}
	tci.terms[0] = make(chan struct{})
	tci.terms[1] = make(chan struct{})
	tci.terms[2] = make(chan struct{})
	
	tci.msgQ = NewClusterMessageQueue(17999)
	tci.counter = NewClusterCounter(17998)
	tci.registration = NewClusterAgentRegistration(17997, counterAddr, nodesAddr)
	tci.nodes = make([]*ClusterNode, clusterSize)
	for i:=0; i<clusterSize; i++{
	    tci.nodes[i] = NewClusterNode(18000+i, msgQAddr, counterAddr, registrationAddr)
	}
    
    go tci.counter.Work(timeout, tci.terms[1])
    go tci.msgQ.Work(timeout, tci.terms[0])
    go tci.registration.Work(timeout, tci.terms[2])
    
    for i:=0; i<clusterSize; i++{
	    go tci.nodes[i].Work(timeout, tci.terms[3+i])
	}
    
    tci.agents = make([]*ClusterAgent, componentNbr)
    for i:=0; i<componentNbr; i++{
        tci.agents[i] = NewClusterAgent(msgQAddr, registrationAddr)
	}
}

func (tci *testClusterInfrastructure) teardownTest(){
    for _, chnTO := range tci.terms{
        <- chnTO
    }
    tci.counter.Terminate()
    tci.msgQ.Terminate()
    tci.registration.Terminate()
    for _, nd := range tci.nodes{
        nd.Terminate()
    }
} 

func TestComponentEmpty(t *testing.T) {
    tst := testClusterInfrastructure{}
    tst.initTest(2000, 1, 1)
	defer tst.teardownTest()
	comp := NewComponent(tst.agents[0])
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
    tst := testClusterInfrastructure{}
    tst.initTest(2000, 1, 2)
	
	run1 := false
	comp1 := NewComponent(tst.agents[0])
	comp2 := NewComponent(tst.agents[1])
	NewProcess(comp1).Run(func(*Process) {
		run1 = true
	})
	run2 := false
	NewProcess(comp2).Run(func(*Process) {
		run2 = true
	})
	defer func() {
		tst.teardownTest()
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
	tst := testClusterInfrastructure{}
    tst.initTest(2000, 1, 2)
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
	comp1 := NewComponent(tst.agents[0])
	comp2 := NewComponent(tst.agents[1])
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
		tst.teardownTest()
		if !sent || !received {
			t.Fail()
		}
	}()
}

func TestSendReceive(t *testing.T) {
	tst := testClusterInfrastructure{}
    tst.initTest(2000, 1, 2)
    
	sent := false
	received := false
	comp1 := NewComponent(tst.agents[0])
	comp2 := NewComponent(tst.agents[1])
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
		tst.teardownTest()
		if !sent || !received {
			t.Fail()
		}
	}()
}

func TestSendTwoReceive(t *testing.T) {
	tst := testClusterInfrastructure{}
    tst.initTest(2000, 1, 3)
	sent := false
	received2 := false
	received3 := false
	comp1 := NewComponent(tst.agents[0])
	comp2 := NewComponent(tst.agents[1])
	comp3 := NewComponent(tst.agents[2])
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
		tst.teardownTest()
		if !sent || !received2 || !received3 {
			t.Fail()
		}
	}()
}

func TestSendTwoReceiveOneAcceptThenTheOther(t *testing.T) {
	tst := testClusterInfrastructure{}
    tst.initTest(2000, 1, 3)
	sent := false
	received2 := false
	received3 := false
	comp1 := NewComponent(tst.agents[0])
	comp2 := NewComponent(tst.agents[1])
	comp3 := NewComponent(tst.agents[2])
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
		tst.teardownTest()
		if !sent || !received2 || !received3 {
			t.Fail()
		}
	}()
}
