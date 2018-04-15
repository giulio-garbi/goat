package goat

import (
    "sync"
    "sync/atomic"
)

// Contains the set of registered agents, and assigns them to the nodes
type TreeAgentRegistration = RingAgentRegistration
func NewTreeAgentRegistration(port int, nodesAddresses []string) *TreeAgentRegistration{
    return NewRingAgentRegistration(port, nodesAddresses)
}
func NewTreeAgentRegistrationPolicy(port int, nodesAddresses []string, policy func(*RingAgentRegistration, []CandidateNode)int) *TreeAgentRegistration{
    return NewRingAgentRegistrationPolicyPerf(false, port, nodesAddresses, policy)
}

type TreeNode struct{
    counter int //only for the root
    agents map[int]*duplexConn
    port int
    messages map[int]tnMessageToForward
    nid int
    parentAddress string //except the root, which has ""
    childNodesAddresses []string
    parentConn *duplexConn //except the root, which has nil
    childNodesConn []*duplexConn
    lock *sync.Mutex
    registrationAddress string
    
    infrMessagesFromAgents uint64
    infrMessagesSent uint64
    perfTest bool
}

type tnMessageToForward struct{
    message []string
    fromTheParent bool
    sourceAgent int
    sourceDescendant int
}

func NewTreeNode(port int, parentAddress string, registrationAddress string, childNodesAddresses []string) *TreeNode {
    return NewTreeNodePerf(false, port, parentAddress, registrationAddress, childNodesAddresses)
}

func NewTreeNodePerf(perfTest bool, port int, parentAddress string, registrationAddress string, childNodesAddresses []string) *TreeNode {
    return &TreeNode{
        counter: 0,
        agents: map[int]*duplexConn{},
        port: port,
        messages: map[int]tnMessageToForward{},
        nid: 0,
        parentAddress: parentAddress,
        childNodesAddresses: childNodesAddresses,
        lock: &sync.Mutex{},
        registrationAddress: registrationAddress,
        perfTest: perfTest,
        infrMessagesFromAgents: 0,
        infrMessagesSent: 0,
    }
}

func (tn *TreeNode) onInfrMsgAgent() {
    if tn.perfTest {
        atomic.AddUint64(&tn.infrMessagesFromAgents, 1)
    }
}

func (tn *TreeNode) onInfrMsgSent() {
    if tn.perfTest {
        atomic.AddUint64(&tn.infrMessagesSent, 1)
    }
}

func (tn *TreeNode) GetInfrMsgAgent() uint64 {
    if tn.perfTest {
        return atomic.LoadUint64(&tn.infrMessagesFromAgents)
    } else {
        panic("No performance tests required!")
    }
}

func (tn *TreeNode) GetInfrMsgSent() uint64 {
    if tn.perfTest {
        return atomic.LoadUint64(&tn.infrMessagesSent)
    } else {
        panic("No performance tests required!")
    }
}
/*
func (tn *TreeNode) getMessageToForward(params []string, childSource int) tnMessageToForward{
    // DATA mid src pred msg 
    //idx    0   1   2   3..
    src := params[1]
    // src = [0-9]+  => it's an agent
    // src = P[0-9]+ => it's a node
    if !strings.HasPrefix(src,"P"){
        return tnMessageToForward{
            message: append([]string{"DATA"}, params...),
            fromTheParent: false,
            sourceAgent: atoi(src),
            sourceDescendant: -1,
        }
    } else {
        if sourceReplyAddress == tn.parentAddress {
            return tnMessageToForward{
                message: append([]string{"DATA"}, params...),
                fromTheParent: true,
                sourceAgent: -1,
                sourceDescendant: -1,
            }
        } else {
            return tnMessageToForward{
                message: append([]string{"DATA"}, params...),
                fromTheParent: false,
                sourceAgent: -1,
                sourceDescendant: childSource,
            }
        }
    }
}*/

func (tn *TreeNode) prepareMessageForInfrastructure(msg tnMessageToForward)[]string{
    msg.message[2] = "-1"//fmt.Sprintf("P%s", tn.port)
    return msg.message
}

func (tn *TreeNode) prepareMessageForAgent(msg tnMessageToForward)[]string{
    msg.message[2] = "-1"
    return msg.message
}

func (tn *TreeNode) amRoot() bool{
    return tn.parentAddress == ""
}

func (tn *TreeNode) resolveLastAddress(seq []string) (*duplexConn, []string){
    // REQ compId port0 addr0 port1 addr1 ... port_n-1 addr_n-1 port_n
    // NB: I am addr_n+1
    /*
        node       gets
        leaf node: REQ compId
        parent " : REQ compId idxLeaf1
        parent " : REQ compId idxLeaf1 idxLeaf2
        root     : REQ compId port0 addr0 port1 addr1 port2 
        ===> NB it will be given to resolveLastAddress with addr2 appended at the end, unless it is a compId
        child "  : RPLY mid compId port0 addr0 port1 addr1
        child "  : RPLY mid compId port0 addr0
        child "  : RPLY mid compId 
        compId   : RPLY mid
    */
    if atoi(seq[len(seq)-1]) >= len(tn.childNodesAddresses){ // is a component id
        return tn.agents[atoi(seq[len(seq)-1])-len(tn.childNodesAddresses)], []string{}
    } else {
        return tn.childNodesConn[atoi(seq[len(seq)-1])], seq[:len(seq)-1]
    }
}

func (tn *TreeNode) serveParent() {
    for{
        cmd, params := tn.parentConn.Receive()
        switch(cmd) {
        case "RPLY": 
                assMid := params[0]
                path := params[1:]
                childConn, remainder := tn.resolveLastAddress(path)
                //fmt.Println(tn.childNodesConn)
                childConn.Send(append([]string{"RPLY", assMid}, remainder...)...)
                tn.onInfrMsgSent()
                dprintln("sent rply", append([]string{"RPLY", assMid}, remainder...))
        case "DATA": // DATA mid src pred msg
                msg := tnMessageToForward{
                    message: append([]string{"DATA"}, params...),
                    fromTheParent: true,
                    sourceAgent: -1,
                    sourceDescendant: -1,
                }
                msgId := atoi(params[0])
                tn.lock.Lock()
                if _, has := tn.messages[msgId]; has {
                    panic("Got msg "+itoa(msgId)+" twice!")
                }
                tn.messages[msgId] = msg
                tn.dispatch()
                tn.lock.Unlock()
        }
    }
}

func (tn *TreeNode) serveChild(childConn *duplexConn, idx int) {
    amANode := idx < len(tn.childNodesConn)
    if !amANode {
        dprintln(idx - len(tn.childNodesConn), "with", tn.port)
    }
    for{
        cmd, params,err := childConn.ReceiveErr()
        if err != nil {
            if amANode {
                panic(err) 
            } else {
                return
            }
        }
        if !amANode {
            tn.onInfrMsgAgent()
        }
        switch(cmd) {
        case "REQ": 
                //fmt.Println("got req")
                var corrPath []string
                if amANode {
                    path := params
                    corrPath = append(path, itoa(idx))
                } else {
                    corrPath = []string{itoa(idx)}
                }
                    
                if tn.amRoot(){
                    tn.lock.Lock()
                    assMid := itoa(tn.counter)
                    tn.counter++
                    //fmt.Println("fwding",corrPath)
                    childC, remainder := tn.resolveLastAddress(corrPath)
                    if childC != childConn {
                        panic("Whoops!")
                    }
                    tn.lock.Unlock()
                    childC.Send(append([]string{"RPLY", assMid}, remainder...)...)
                    tn.onInfrMsgSent()
                    dprintln("sent rply",append([]string{"RPLY", assMid}, remainder...))
                } else {
                    tn.parentConn.Send(append([]string{"REQ"}, corrPath...)...)
                    tn.onInfrMsgSent()
                    //fmt.Println("sent req",append([]string{"REQ"}, corrPath...))
                }
        case "DATA": // DATA mid src pred msg
                //msg := tn.getMessageToForward(srcAddr,params,idx)
                msg := tnMessageToForward{
                    message: append([]string{"DATA"}, params...),
                    fromTheParent: false,
                }
                if amANode {
                    msg.sourceDescendant = idx
                    msg.sourceAgent = -1
                } else {
                    msg.sourceDescendant = -1
                    msg.sourceAgent = idx - len(tn.childNodesConn)//atoi(params[1])
                }
                if !tn.amRoot() {
                    tn.parentConn.Send(tn.prepareMessageForInfrastructure(msg)...)
                    tn.onInfrMsgSent()
                }
                msgId := atoi(params[0])
                tn.lock.Lock()
                dprintln("got", msgId)
                if _, has := tn.messages[msgId]; has {
                    panic("Got msg "+itoa(msgId)+" twice!")
                }
                tn.messages[msgId] = msg
                tn.dispatch()
                tn.lock.Unlock()
        }
    }
}

func (tn *TreeNode) handleAgent(idx int, conn *duplexConn) {
    tn.lock.Lock()
    conn.Send("Registered", itoa(idx), itoa(tn.nid))
    tn.onInfrMsgSent()
    //fmt.Println("Agent", idx, "started at mid",tn.nid, len(tn.childNodesAddresses))
    tn.lock.Unlock()
    tn.serveChild(conn, idx + len(tn.childNodesAddresses))
}

func (tn *TreeNode) regConnHandlerIn(regConn *duplexConn) {
    for {
        cmd, params := regConn.Receive()
        if cmd == "newAgent"{ // a new agent arrived
            tn.lock.Lock()
            agCompId := params[0]
            agAddr := params[1]
            agConn := connectWith(agAddr)
            tn.agents[atoi(agCompId)] = agConn
            go func(idx int, conn *duplexConn){tn.handleAgent(idx, conn)}(atoi(agCompId), agConn)
            tn.lock.Unlock()
        }
    }
}

func (tn *TreeNode) dispatch() {
    for{
        if mFwd, has := tn.messages[tn.nid]; has{
            delete(tn.messages, tn.nid)
            dprintln("disp", tn.nid, tn.port)
            
            mFwdAgent := tn.prepareMessageForAgent(mFwd)
            for agentId, agentConn := range tn.agents {
                if agentId != mFwd.sourceAgent{
                    agentConn.Send(mFwdAgent...)
                    tn.onInfrMsgSent()
                }
            }
            
            mFwdInfr := tn.prepareMessageForInfrastructure(mFwd)
            for nodeId, nodeConn := range tn.childNodesConn {
                if nodeId != mFwd.sourceDescendant{
                    nodeConn.Send(mFwdInfr...)
                    tn.onInfrMsgSent()
                }
            }
            
            tn.nid++
        } else {
            break
        }
    }
}

func (tn *TreeNode) WorkLoop() {
    tn.Work(0, make(chan struct{}))
}

func (tn *TreeNode) Work(timeout int64, timedOut chan<- struct{}){
    listenerConns, chnReady := listener(tn.port)
    regConn := connectWith(tn.registrationAddress)
    <-chnReady
    if len(tn.childNodesAddresses) > 0 {
        regConn.Send("ready")
        tn.onInfrMsgSent()
    } else {
        regConn.Send("ready","leaf")
        tn.onInfrMsgSent()
    }
    for canConnectParent := false; !canConnectParent;{
        cmd, _ := regConn.Receive()
        canConnectParent = cmd == "connNext"
    }
    chnConnParent := make(chan struct{})
    if tn.parentAddress == "" {
        tn.parentConn = nil
        close(chnConnParent)
    } else {
        go func() {
            tn.parentConn = connectWith(tn.parentAddress)
            close(chnConnParent)
        }()
    }
    tn.childNodesConn = make([]*duplexConn, len(tn.childNodesAddresses))
    for i := range tn.childNodesAddresses {
        tn.childNodesConn[i] = <- listenerConns.Out
    }
    <-chnConnParent
    go func(){tn.regConnHandlerIn(regConn)}()
    if !tn.amRoot() {
        go func(){tn.serveParent()}()
    }
    for idx,nd := range tn.childNodesConn{
        go func(n *duplexConn, i int){tn.serveChild(n, i)}(nd, idx)
    }
}

func (tn *TreeNode) Terminate(){
    //tn.listener.Close()
}
