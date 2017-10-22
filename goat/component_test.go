// component_test.go
package goat

import (
	"testing"
	"fmt"
	"math/rand"
	"encoding/gob"
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


type testRingInfrastructure struct{
    nodes []*RingNode
    agents []*RingAgent
    registration *RingAgentRegistration
    counter *ClusterCounter
    terms []chan struct{}
}

func (tri *testRingInfrastructure) initTest(timeout int64, ringSize int, componentNbr int) {
	// Launching a clustered infrastructure
	tri.terms = make([]chan struct{}, 2 + ringSize)
	
	counterAddr := "127.0.0.1:17998"
	registrationAddr := "127.0.0.1:17997"
	nodesAddr := make([]string, ringSize)
	for i:=0; i<ringSize; i++{
	    nodesAddr[i] = fmt.Sprintf("127.0.0.1:%d",18000+i)
	    tri.terms[i+2] = make(chan struct{})
	}
	tri.terms[0] = make(chan struct{})
	tri.terms[1] = make(chan struct{})
	
	tri.counter = NewClusterCounter(17998)
	tri.registration = NewRingAgentRegistration(17997, nodesAddr)
	tri.nodes = make([]*RingNode, ringSize)
	for i:=0; i<ringSize; i++{
	    tri.nodes[i] = NewRingNode(18000+i, counterAddr, nodesAddr[(i+1)%ringSize])
	}
    
    go tri.counter.Work(timeout, tri.terms[0])
    go tri.registration.Work(timeout, tri.terms[1])
    
    for i:=0; i<ringSize; i++{
	    go tri.nodes[i].Work(timeout, tri.terms[2+i])
	}
    
    tri.agents = make([]*RingAgent, componentNbr)
    for i:=0; i<componentNbr; i++{
        tri.agents[i] = NewRingAgent(registrationAddr)
	}
}

func (tri *testRingInfrastructure) teardownTest(){
    for _, chnTO := range tri.terms{
        <- chnTO
    }
    tri.counter.Terminate()
    tri.registration.Terminate()
    for _, nd := range tri.nodes{
        nd.Terminate()
    }
} 

type testTreeInfrastructure struct{
    nodes []*TreeNode
    agents []*TreeAgent
    registration *TreeAgentRegistration
    terms []chan struct{}
}

type treeInfrBuilder struct {
    myAddress string
    parentAddress string
    child []*treeInfrBuilder
}

func createTree(treeDepth int, maxChild int, portCount int) (*treeInfrBuilder, int) {
    childs := 0
    if maxChild > 0 && treeDepth > 0 {
        childs = 1 + rand.Intn(maxChild)
    }
    out := treeInfrBuilder{
        myAddress: fmt.Sprintf("127.0.0.1:%d", portCount),
        child: make([]*treeInfrBuilder, childs),
    }
    portCount++
    for i:=0; i<childs; i++{
        out.child[i], portCount = createTree(treeDepth-1, maxChild, portCount)
        out.child[i].parentAddress = out.myAddress
    }
    return &out, portCount
}

func (tib *treeInfrBuilder) getParentsChild(parents *map[string]string, childs *map[string][]string){
    (*parents)[tib.myAddress] = tib.parentAddress
    (*childs)[tib.myAddress] = make([]string, len(tib.child))
    for i, tr := range tib.child{
        ((*childs)[tib.myAddress])[i] = tr.myAddress
        tr.getParentsChild(parents, childs)
    }
}

func (tti *testTreeInfrastructure) initTest(timeout int64, treeDepth int, maxChild int, componentNbr int) {
    tree, nextPort := createTree(treeDepth, maxChild, 18000)
    treeSize := nextPort-18000

	// Launching a clustered infrastructure
	tti.terms = make([]chan struct{}, 1 + treeSize)
	
	registrationAddr := "127.0.0.1:17997"
	nodesAddr := make([]string, treeSize)
	for i:=0; i<treeSize; i++{
	    nodesAddr[i] = fmt.Sprintf("127.0.0.1:%d",18000+i)
	    tti.terms[i+1] = make(chan struct{})
	}
	tti.terms[0] = make(chan struct{})
	
	tti.registration = NewTreeAgentRegistration(17997, nodesAddr)
	tti.nodes = make([]*TreeNode, treeSize)

    parents := map[string]string{}
    childs := map[string][]string{}
    tree.getParentsChild(&parents, &childs)
    
	for i:=0; i<treeSize; i++{
	    tti.nodes[i] = NewTreeNode(18000+i, parents[nodesAddr[i]], childs[nodesAddr[i]])
	}
    
    go tti.registration.Work(timeout, tti.terms[0])
    
    for i:=0; i<treeSize; i++{
	    go tti.nodes[i].Work(timeout, tti.terms[1+i])
	}
    
    tti.agents = make([]*TreeAgent, componentNbr)
    for i:=0; i<componentNbr; i++{
        tti.agents[i] = NewTreeAgent(registrationAddr)
	}
}

func (tti *testTreeInfrastructure) teardownTest(){
    for _, chnTO := range tti.terms{
        <- chnTO
        fmt.Println("X", len(tti.terms))
    }
    tti.registration.Terminate()
    for _, nd := range tti.nodes{
        nd.Terminate()
    }
} 

func TestComponentEmpty(t *testing.T) {
    tst := testTreeInfrastructure{}
    tst.initTest(2000, 2, 4, 1)
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
    tst := testTreeInfrastructure{}
    tst.initTest(2000, 2, 4, 2)
	
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
	tst := testTreeInfrastructure{}
    tst.initTest(2000, 2, 4, 2)
	sendOb := Foo{
	    Dog : "bark",
	    Cat : "meoww",
	    Fish : "",
	    Rat: []string{"squit"},
	    Monkey: 5,
	}
	gob.Register(sendOb) //Needed to exchange non-standard objects
	sent := false
	received := false
	comp1 := NewComponent(tst.agents[0])
	comp2 := NewComponent(tst.agents[1])
	NewProcess(comp1).Run(func(p *Process) {
		p.Send(NewTuple(sendOb), True())
		sent = true
	})
	NewProcess(comp2).Run(func(p *Process) {
		p.Receive(func(attr *Attributes, t Tuple) bool {
		    if !t.IsLong(1){
		        return false
		    }
		    recOb := t.Get(0).(Foo)
			return recOb.Dog == sendOb.Dog && recOb.Cat == sendOb.Cat && 
			    recOb.Fish == sendOb.Fish && recOb.Monkey == sendOb.Monkey &&
			    len(recOb.Rat) == 1 && recOb.Rat[0] == sendOb.Rat[0]  
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

func TestSendReceive(t *testing.T) {
	tst := testTreeInfrastructure{}
    tst.initTest(2000, 2, 4, 2)
    
	sent := false
	received := false
	comp1 := NewComponent(tst.agents[0])
	comp2 := NewComponent(tst.agents[1])
	NewProcess(comp1).Run(func(p *Process) {
		p.Send(NewTuple("Ciao"), True())
		sent = true
	})
	NewProcess(comp2).Run(func(p *Process) {
		p.Receive(func(attr *Attributes, msg Tuple) bool {
		    if !msg.IsLong(1){
		        return false
		    }
			return msg.Get(0) == "Ciao"
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
	tst := testTreeInfrastructure{}
    tst.initTest(2000, 2, 4, 3)
	sent := false
	received2 := false
	received3 := false
	comp1 := NewComponent(tst.agents[0])
	comp2 := NewComponent(tst.agents[1])
	comp3 := NewComponent(tst.agents[2])
	NewProcess(comp1).Run(func(p *Process) {
		p.Send(NewTuple("Ciao"), True())
		sent = true
	})
	NewProcess(comp2).Run(func(p *Process) {
		p.Receive(func(attr *Attributes, msg Tuple) bool {
		    if !msg.IsLong(1){
		        return false
		    }
			return msg.Get(0) == "Ciao"
		})
		received2 = true
	})
	NewProcess(comp3).Run(func(p *Process) {
		p.Receive(func(attr *Attributes, msg Tuple) bool {
		    if !msg.IsLong(1){
		        return false
		    }
			return msg.Get(0) == "Ciao"
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
    fmt.Println("---")
	tst := testTreeInfrastructure{}
    tst.initTest(2000, 2, 4, 3)
	sent := false
	received2 := false
	received3 := false
	comp1 := NewComponent(tst.agents[0])
	comp2 := NewComponent(tst.agents[1])
	comp3 := NewComponent(tst.agents[2])
	NewProcess(comp1).Run(func(p *Process) {
		p.Send(NewTuple("Ciao"), True())
		sent = true
	})
	NewProcess(comp2).Run(func(p *Process) {
		p.Receive(func(attr *Attributes, t Tuple) bool {
		    if !t.IsLong(1){
		        return false
		    }
			return t.Get(0) == "Ciao"
		})
		received2 = true
		p.Send(NewTuple("Ciaone"), True())
	})
	NewProcess(comp3).Run(func(p *Process) {
		p.Receive(func(attr *Attributes, msg Tuple) bool {
		    if !msg.IsLong(1){
		        return false
		    }
			return msg.Get(0) == "Ciaone"
		})
		if !received2 {
			t.Error("comp2 must have received before me!")
		}
		received3 = true
	})
	defer func() {
		tst.teardownTest()
		fmt.Println(sent, received2, received3)
		if !sent || !received2 || !received3 {
			t.Fail()
		}
	}()
}
