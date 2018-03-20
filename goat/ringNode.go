package goat

import (
    "math/rand"
    "time"
    "sync"
    "fmt"
    "net"
)

// Contains the set of registered agents, and assigns them to the nodes
type RingAgentRegistration struct {
    nodesAddresses []string
    port int
    compId int
    policy func(*RingAgentRegistration, []CandidateNode)int
    lock *sync.Mutex
    listenerConns *unboundChanConn
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
    return NewRingAgentRegistrationPolicy(port, nodesAddresses, RingSequentialPolicy())
}

func NewRingAgentRegistrationPolicy(port int, nodesAddresses []string,policy func(*RingAgentRegistration, []CandidateNode)int) *RingAgentRegistration{
    listenerConns, chnReady := listener(port)
    <-chnReady
    return &RingAgentRegistration{
        nodesAddresses: nodesAddresses,
        port: port,
        policy: policy,
        lock: &sync.Mutex{},
        listenerConns: listenerConns,
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
}

func NewRingNode(port int, counterAddress string, nextNodeAddress string, registrationAddress string) *RingNode {
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
        switch(cmd) {
            case "REQ":
                rn.counterConn.Send("inc")
                go func(){
                    mid := <- rn.chnMids.Out
                    conn.Send("RPLY", itoa(mid))
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
                            }
                        }
                        mParams[1] = itoa(sender) // reset before forwarding
                        rn.nextNodeConn.Send(rn.messages[rn.nid]...)
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
}

func NewRingCounter(port int) *RingCounter {
    listenerConns, chnReady := listener(port)
    <-chnReady
    return &RingCounter{
        mid: 0,
        port: port,
        lock: &sync.Mutex{},
        listenerConns: listenerConns,
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
        }
    }
}

func (rc *RingCounter) Work(timeout int64, timedOut chan<- struct{}){
    
    for {
        conn := <- rc.listenerConns.Out
        go func(c *duplexConn){rc.handleConn(c)}(conn)
    }
}

