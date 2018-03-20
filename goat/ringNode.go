package goat

import (
    "math/rand"
    "time"
    "sync"
    "fmt"
    "net"
    "sync/atomic"
)

// Contains the set of registered agents, and assigns them to the nodes
type RingAgentRegistration struct {
    nodesAddresses []string
    port int
    compId int
    policy func(*RingAgentRegistration, []CandidateNode)int
    lock *sync.Mutex
    listenerConns *unboundChanConn
    infrMessagesFromAgents uint64
    infrMessagesSent uint64
    perfTest bool
}

type CandidateNode struct {
    IsLeaf bool
    Address net.Addr
}

func RingSequentialPolicy() (func (*RingAgentRegistration, []CandidateNode)int){
    idx := 0
    return func(rar *RingAgentRegistration, agent []CandidateNode)int{
        if idx >= len(agent) {
            idx = 1
        } else {
            idx++
        }
        return idx-1
    }
}

func RingRandomPolicy() (func (*RingAgentRegistration, []CandidateNode)int){
    src := rand.NewSource(time.Now().Unix())
    return func(rar *RingAgentRegistration, agent []CandidateNode)int{
        return rand.New(src).Intn(len(agent))
    }
}

func TreeOnlyLeaf() (func (*RingAgentRegistration, []CandidateNode)int){
    idx := 0
    return func(rar *RingAgentRegistration, agent []CandidateNode)int{
        leafs := []int{}
        for i, ag := range agent {
            if ag.IsLeaf {
                leafs = append(leafs, i)
            }
        }
        if idx >= len(leafs) {
            idx = 1
        } else {
            idx++
        }
        return leafs[idx-1]
    }
}

func NewRingAgentRegistration(port int, nodesAddresses []string) *RingAgentRegistration{
    return NewRingAgentRegistrationPolicyPerf(false, port, nodesAddresses, RingSequentialPolicy())
}

func NewRingAgentRegistrationPolicyPerf(perfTest bool, port int, nodesAddresses []string,policy func(*RingAgentRegistration, []CandidateNode)int) *RingAgentRegistration{
    listenerConns, chnReady := listener(port)
    <-chnReady
    return &RingAgentRegistration{
        nodesAddresses: nodesAddresses,
        port: port,
        policy: policy,
        lock: &sync.Mutex{},
        listenerConns: listenerConns,
        perfTest: perfTest,
        infrMessagesFromAgents: 0,
        infrMessagesSent: 0,
    }
}

func (tn *RingAgentRegistration) onInfrMsgAgent() {
    if tn.perfTest {
        atomic.AddUint64(&tn.infrMessagesFromAgents, 1)
    }
}

func (tn *RingAgentRegistration) onInfrMsgSent() {
    if tn.perfTest {
        atomic.AddUint64(&tn.infrMessagesSent, 1)
    }
}

func (tn *RingAgentRegistration) GetInfrMsgAgent() uint64 {
    if tn.perfTest {
        return atomic.LoadUint64(&tn.infrMessagesFromAgents)
    } else {
        panic("No performance tests required!")
    }
}

func (tn *RingAgentRegistration) GetInfrMsgSent() uint64 {
    if tn.perfTest {
        return atomic.LoadUint64(&tn.infrMessagesSent)
    } else {
        panic("No performance tests required!")
    }
}

func (rar *RingAgentRegistration) Work(timeout int64, timedOut chan<- struct{}){
    conns := []*duplexConn{}
    candNodes := []CandidateNode{}
    readyReceived := 0
    chnStartRegistrations := make(chan struct{})
    for {
        conn := <- rar.listenerConns.Out
        cmd, params, err := conn.ReceiveErr()
        switch (cmd) {
            case "Register":
                if err == nil {
                    rar.onInfrMsgAgent()
                    agPort := params[0]
                    agAddr := netAddress{conn.SrcAddr().Host, agPort}
                    go func(con *duplexConn, addr netAddress){
                        <- chnStartRegistrations
                        rar.lock.Lock()
                        compId := rar.compId
                        rar.compId++
                        which := rar.policy(rar, candNodes)
                        rar.lock.Unlock()
                        conns[which].Send("newAgent", itoa(compId), addr.String())
                        rar.onInfrMsgSent()
                    }(conn, agAddr)
                }
            case "ready":
                if err != nil {
                    panic(err)
                }
                isALeaf := len(params) > 0 && params[0] == "leaf" 
                conns = append(conns, conn)
                candNodes = append(candNodes, CandidateNode{IsLeaf: isALeaf, Address: conn.RemoteAddr()})
                readyReceived++
                if readyReceived == len(rar.nodesAddresses) {
                    for _,con := range conns {
                        con.Send("connNext")
                        rar.onInfrMsgSent()
                    }
                    close(chnStartRegistrations)
                }
        }
    }
}
/*
func (rar *RingAgentRegistration) Work(timeout int64, timedOut chan<- struct{}){
    hasTimedOut := false
    for {
        if len(rar.queuedAgents) == 0 {
            cmd, params, srcAddr := receiveWithAddressTimeout(rar.listener, timeout, &hasTimedOut)
            if hasTimedOut {
                close(timedOut)
                return
            }
            switch cmd {
                case "Register":
                    agPort := params[0]
                    agAddr := netAddress{srcAddr.Host, agPort}
                    rar.queuedAgents = append(rar.queuedAgents, agAddr)
            }
        } else {
            agAddr := rar.queuedAgents[0]
            agCompId := itoa(rar.compId)
            rar.compId++
            dprintln("Registering component", agCompId)
            // Random assignment policy!
            ndAddr := rar.policy(rar, agAddr)
            sendTo(ndAddr, "newAgent", agCompId, agAddr.String())
            rar.queuedAgents = rar.queuedAgents[1:]
        }
    }
}*/

func (rar *RingAgentRegistration) Terminate(){
    //rar.listener.Close()
    //TODO
}

///////////

type RingNode struct{
    counterAddress string
    registrationAddress string
    agents map[int]*duplexConn
    rplys map[int]int
    removedComps map[int]struct{}
    port int
    messages map[int][]string
    nid int
    nextNodeAddress string
    lock *sync.Mutex
    counterConn *duplexConn
    chnMids *unboundChanInt
    nextNodeConn *duplexConn
    infrMessagesFromAgents uint64
    infrMessagesSent uint64
    perfTest bool
}

func NewRingNode(port int, counterAddress string, nextNodeAddress string, registrationAddress string) *RingNode {
    return NewRingNodePerf(false, port, counterAddress, nextNodeAddress, registrationAddress)
}

func NewRingNodePerf(perfTest bool, port int, counterAddress string, nextNodeAddress string, registrationAddress string) *RingNode {
    return &RingNode{
        counterAddress: counterAddress,
        agents: map[int]*duplexConn{},
        rplys: map[int]int{},
        removedComps: map[int]struct{}{},
        port: port,
        messages: map[int][]string{},
        nid: 0,
        nextNodeAddress: nextNodeAddress,
        registrationAddress: registrationAddress,
        lock: &sync.Mutex{},
        perfTest: perfTest,
        infrMessagesFromAgents: 0,
        infrMessagesSent: 0,
    }
}

func (tn *RingNode) onInfrMsgAgent() {
    if tn.perfTest {
        atomic.AddUint64(&tn.infrMessagesFromAgents, 1)
    }
}

func (tn *RingNode) onInfrMsgSent() {
    if tn.perfTest {
        atomic.AddUint64(&tn.infrMessagesSent, 1)
    }
}

func (tn *RingNode) GetInfrMsgAgent() uint64 {
    if tn.perfTest {
        return atomic.LoadUint64(&tn.infrMessagesFromAgents)
    } else {
        panic("No performance tests required!")
    }
}

func (tn *RingNode) GetInfrMsgSent() uint64 {
    if tn.perfTest {
        return atomic.LoadUint64(&tn.infrMessagesSent)
    } else {
        panic("No performance tests required!")
    }
}

func counterConnHandlerIn(counterConn *duplexConn, chnMids *unboundChanInt) {
    for {
        cmd, params := counterConn.Receive()
        if cmd == "counter" {
            chnMids.In <- atoi(params[0])
        }
    }
}

func (rn *RingNode) dispatch(idx int) bool {
    agentFailed := false
    for {
        for ; len(rn.messages[rn.nid])>0; rn.nid++{
            mParams := rn.messages[rn.nid][1:]
            sender := atoi(mParams[1])
            mParams[1] = "0" //anonimity
            idxDead := false
            for agentId, agentConn := range rn.agents {
                if agentId != sender{
                    err := agentConn.Send(rn.messages[rn.nid]...)
                    rn.onInfrMsgSent()
                    if err != nil {
                        delete(rn.agents, agentId)
                        if idx == agentId {
                            idxDead = true
                        }
                    }
                }
            }
            mParams[1] = itoa(sender) // reset before forwarding
            rn.nextNodeConn.Send(rn.messages[rn.nid]...)
            rn.onInfrMsgSent()
            delete(rn.messages, rn.nid)
            if idxDead {
                rn.removedComps[idx] = struct{}{}
                dprintln("Agent", idx, "failed")
                agentFailed = true
            }
        }
        whoRply, hasRply := rn.rplys[rn.nid]
        _, isRemoved := rn.removedComps[whoRply]
        if hasRply && isRemoved {
            delete(rn.rplys, rn.nid)
            rn.nid++
        } else {
            break
        }
    }
    return agentFailed
}

func (rn *RingNode) handleAgent(idx int, conn *duplexConn) {
    rn.lock.Lock()
    conn.Send("Registered", itoa(idx), itoa(rn.nid))
    rn.onInfrMsgSent()
    fmt.Println("Agent", idx, "started at mid",rn.nid)
    rn.lock.Unlock()
    for {
        cmd, params, err := conn.ReceiveErr()
        if err != nil {
            //unsubscribe it
            rn.lock.Lock()
            delete(rn.agents, idx)
            rn.removedComps[idx] = struct{}{}
            rn.lock.Unlock()
            dprintln("Agent", idx, "failed")
            return
        }
        rn.onInfrMsgAgent()
        switch(cmd) {
            case "REQ":
                rn.counterConn.Send("inc")
                rn.onInfrMsgSent()
                go func(){
                    mid := <- rn.chnMids.Out
                    conn.Send("RPLY", itoa(mid))
                    rn.onInfrMsgSent()
                    rn.lock.Lock()
                    rn.rplys[mid] = idx
                    isFailed := rn.dispatch(idx)
                    rn.lock.Unlock()
                    if isFailed {
                        rn.removedComps[idx] = struct{}{}
                    }
                }()

            case "DATA":
                msgId := atoi(params[0])
                rn.lock.Lock()
                delete(rn.rplys, msgId)
                if msgId >= rn.nid{
                    rn.messages[msgId] = append([]string{cmd}, params...)
                    
                    /*for ; len(rn.messages[rn.nid])>0; rn.nid++{
                        mParams := rn.messages[rn.nid][1:]
                        sender := atoi(mParams[1])
                        mParams[1] = "0" //anonimity
                        idxDead := false
                        for agentId, agentConn := range rn.agents {
                            if agentId != sender{
                                err = agentConn.Send(rn.messages[rn.nid]...)
                                if err != nil {
                                    delete(rn.agents, agentId)
                                    if idx == agentId {
                                        idxDead = true
                                    }
                                }
                            }
                        }
                        mParams[1] = itoa(sender) // reset before forwarding
                        rn.nextNodeConn.Send(rn.messages[rn.nid]...)
                        delete(rn.messages, rn.nid)
                        if idxDead {
                            rn.lock.Unlock()
                            dprintln("Agent", idx, "failed")
                            return
                        }
                    }*/
                    isFailed := rn.dispatch(idx)
                    if isFailed {
                        rn.removedComps[idx] = struct{}{}
                        rn.lock.Unlock()
                        return
                    }
                }
                rn.lock.Unlock()
        }
    }
}
func (rn *RingNode) handlePrevNode(conn *duplexConn) {
    for {
        cmd, params := conn.Receive()
        switch(cmd) {
            case "DATA":
                msgId := atoi(params[0])
                rn.lock.Lock()
                if msgId >= rn.nid{
                    rn.messages[msgId] = append([]string{cmd}, params...)
                    
                    for ; len(rn.messages[rn.nid])>0; rn.nid++{
                        mParams := rn.messages[rn.nid][1:]
                        sender := atoi(mParams[1])
                        mParams[1] = "0" //anonimity
                        for agentId, agentConn := range rn.agents {
                            if agentId != sender{
                                agentConn.Send(rn.messages[rn.nid]...)
                                rn.onInfrMsgSent()
                            }
                        }
                        mParams[1] = itoa(sender) // reset before forwarding
                        rn.nextNodeConn.Send(rn.messages[rn.nid]...)
                        rn.onInfrMsgSent()
                        delete(rn.messages, rn.nid)
                    }
                }
                rn.lock.Unlock()
        }
    }
}

func (rn *RingNode) regConnHandlerIn(regConn *duplexConn) {
    for {
        cmd, params := regConn.Receive()
        if cmd == "newAgent"{ // a new agent arrived
            rn.lock.Lock()
            agCompId := params[0]
            agAddr := params[1]
            agConn := connectWith(agAddr)
            rn.agents[atoi(agCompId)] = agConn
            go func(idx int, conn *duplexConn){rn.handleAgent(idx, conn)}(atoi(agCompId), agConn)
            rn.lock.Unlock()
        }
    }
}

func (rn *RingNode) Work(timeout int64, timedOut chan<- struct{}){
    listenerConns, chnReady := listener(rn.port)
    regConn := connectWith(rn.registrationAddress)
    rn.counterConn = connectWith(rn.counterAddress)
    <-chnReady
    regConn.Send("ready")
    rn.onInfrMsgSent()
    for canConnectNext := false; !canConnectNext;{
        cmd, _ := regConn.Receive()
        canConnectNext = cmd == "connNext"
    }
    chnConnNext := make(chan struct{})
    go func() {
        rn.nextNodeConn = connectWith(rn.nextNodeAddress)
        close(chnConnNext)
    }()
    prevNodeConn := <- listenerConns.Out
    <-chnConnNext
    rn.chnMids = newUnboundChanInt()
    
    go func(){counterConnHandlerIn(rn.counterConn, rn.chnMids)}()
    go func(){rn.regConnHandlerIn(regConn)}()
    go func(){rn.handlePrevNode(prevNodeConn)}()
}

func (rn *RingNode) Terminate(){
    //rn.listener.Close()
    //TODO
}

////

type RingCounter struct{
    mid int
    port int
    lock *sync.Mutex
    listenerConns *unboundChanConn
    infrMessagesFromAgents uint64
    infrMessagesSent uint64
    perfTest bool
}

func NewRingCounter(port int) *RingCounter {
    return NewRingCounterPerf(false, port)
}

func NewRingCounterPerf(perfTest bool, port int) *RingCounter {
    listenerConns, chnReady := listener(port)
    <-chnReady
    return &RingCounter{
        mid: 0,
        port: port,
        lock: &sync.Mutex{},
        listenerConns: listenerConns,
        perfTest: perfTest,
        infrMessagesFromAgents: 0,
        infrMessagesSent: 0,
    }
}

func (tn *RingCounter) onInfrMsgAgent() {
    if tn.perfTest {
        atomic.AddUint64(&tn.infrMessagesFromAgents, 1)
    }
}

func (tn *RingCounter) onInfrMsgSent() {
    if tn.perfTest {
        atomic.AddUint64(&tn.infrMessagesSent, 1)
    }
}

func (tn *RingCounter) GetInfrMsgAgent() uint64 {
    if tn.perfTest {
        return atomic.LoadUint64(&tn.infrMessagesFromAgents)
    } else {
        panic("No performance tests required!")
    }
}

func (tn *RingCounter) GetInfrMsgSent() uint64 {
    if tn.perfTest {
        return atomic.LoadUint64(&tn.infrMessagesSent)
    } else {
        panic("No performance tests required!")
    }
}

func (rc *RingCounter) handleConn(conn *duplexConn) {
    for {
        cmd, _ := conn.Receive()
        if cmd == "inc"{
            rc.lock.Lock()
            mid := rc.mid
            rc.mid++
            rc.lock.Unlock()
            conn.Send("counter", itoa(mid))
            rc.onInfrMsgSent()
        }
    }
}

func (rc *RingCounter) Work(timeout int64, timedOut chan<- struct{}){
    
    for {
        conn := <- rc.listenerConns.Out
        go func(c *duplexConn){rc.handleConn(c)}(conn)
    }
}

