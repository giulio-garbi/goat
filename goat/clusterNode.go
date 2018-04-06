package goat

import (
    "net"
    "sync/atomic"
)

// Contains the set of registered agents, and informs the nodes about their arrival
type ClusterAgentRegistration struct {
    listener net.Listener
    counterAddress string
    nodesAddresses []string
    agentAddresses map[netAddress]struct{}
    queuedAgents []netAddress
    port string
    compId int
    messagesExchanged int
    infrMessagesFromAgents uint64
    infrMessagesSent uint64
    perfTest bool
}

func NewClusterAgentRegistration(port int, counterAddress string, nodesAddresses []string) *ClusterAgentRegistration {
    return NewClusterAgentRegistrationPerf(false, port, counterAddress, nodesAddresses)
}

func NewClusterAgentRegistrationPerf(perfTest bool, port int, counterAddress string, nodesAddresses []string) *ClusterAgentRegistration{
    return &ClusterAgentRegistration{
        listener: listenToPort(port),
        counterAddress: counterAddress,
        nodesAddresses: nodesAddresses,
        agentAddresses: map[netAddress]struct{}{},
        queuedAgents: make([]netAddress, 0),
        messagesExchanged: 0,
        port: itoa(port),
        perfTest: perfTest,
        infrMessagesFromAgents: 0,
        infrMessagesSent: 0,
    }
}

func (tn *ClusterAgentRegistration) onInfrMsgAgent() {
    if tn.perfTest {
        atomic.AddUint64(&tn.infrMessagesFromAgents, 1)
    }
}

func (tn *ClusterAgentRegistration) onInfrMsgSent() {
    if tn.perfTest {
        atomic.AddUint64(&tn.infrMessagesSent, 1)
    }
}

func (tn *ClusterAgentRegistration) GetInfrMsgAgent() uint64 {
    if tn.perfTest {
        return atomic.LoadUint64(&tn.infrMessagesFromAgents)
    } else {
        panic("No performance tests required!")
    }
}

func (tn *ClusterAgentRegistration) GetInfrMsgSent() uint64 {
    if tn.perfTest {
        return atomic.LoadUint64(&tn.infrMessagesSent)
    } else {
        panic("No performance tests required!")
    }
}

func (tn *ClusterAgentRegistration) WorkLoop() {
    tn.Work(0, make(chan struct{}))
}

func (car *ClusterAgentRegistration) Work(timeout int64, timedOut chan<- struct{}){
    hasTimedOut := false
    for {
        if len(car.queuedAgents) == 0 {
            cmd, params, srcAddr := receiveWithAddressTimeout(car.listener, timeout, &hasTimedOut)
            if hasTimedOut {
                close(timedOut)
                return
            }
            switch cmd {
                case "Register":
                    car.onInfrMsgAgent()
                    agPort := params[0]
                    agAddr := netAddress{srcAddr.Host, agPort}
                    car.queuedAgents = append(car.queuedAgents, agAddr)
                case "newAgentKnown":
                    panic("no agent is being announced!")
            }
        } else {
            agAddr := car.queuedAgents[0]
            agCompId := itoa(car.compId)
            car.compId++
            dprintln("Registering component", agCompId)
            for _, ndAddr := range car.nodesAddresses {
                car.onInfrMsgSent()
                sendTo(ndAddr, "newAgent", agCompId, agAddr.String())
            }
            
            for nodesToReply := len(car.nodesAddresses); nodesToReply > 0; {
                cmd, params, srcAddr := receiveWithAddressTimeout(car.listener, timeout, &hasTimedOut)
                if hasTimedOut {
                    close(timedOut)
                    return
                }
                switch cmd {
                    case "Register":
                        car.onInfrMsgAgent()
                        nagPort := params[0]
                        nagAddr := netAddress{srcAddr.Host, nagPort}
                        car.queuedAgents = append(car.queuedAgents, nagAddr)
                    case "newAgentKnown":
                        nodesToReply--
                }
            }
            
            car.onInfrMsgSent()
            sendTo(car.counterAddress, "read", car.port)
            // get current count
            msgCnt := ""
            for msgCnt == "" {
                cmd, params, srcAddr := receiveWithAddressTimeout(car.listener, timeout, &hasTimedOut)
                if hasTimedOut {
                    close(timedOut)
                    return
                }
                switch cmd {
                    case "Register":
                        car.onInfrMsgAgent()
                        nagPort := params[0]
                        nagAddr := netAddress{srcAddr.Host, nagPort}
                        car.queuedAgents = append(car.queuedAgents, nagAddr)
                    case "newAgentKnown":
                        panic("no agent is being announced!")
                    case "count":
                        msgCnt = params[0]
                }
            }
            
            car.onInfrMsgSent()
            sendToAddress(agAddr, "Registered", agCompId, msgCnt)
            car.queuedAgents = car.queuedAgents[1:]
        }
    }
}

func (car *ClusterAgentRegistration) Terminate(){
    car.listener.Close()
}

///////////

type ClusterMessageQueue struct{
    listener net.Listener
    messages [][]string
    queued []netAddress
    infrMessagesFromAgents uint64
    infrMessagesSent uint64
    perfTest bool
}

func NewClusterMessageQueue(port int) *ClusterMessageQueue {
    return NewClusterMessageQueuePerf(false, port)
}

func NewClusterMessageQueuePerf(perfTest bool, port int) *ClusterMessageQueue {
    return &ClusterMessageQueue{
        listener: listenToPort(port),
        messages: make([][]string, 0),
        queued: make([]netAddress, 0),
        infrMessagesFromAgents: 0,
        infrMessagesSent: 0,
        perfTest: perfTest,
    }
}

func (tn *ClusterMessageQueue) WorkLoop() {
    tn.Work(0, make(chan struct{}))
}

func (cmq *ClusterMessageQueue) Work(timeout int64, timedOut chan<- struct{}){
    hasTimedOut := false
    for{
        cmd, params, srcAddr := receiveWithAddressTimeout(cmq.listener, timeout, &hasTimedOut)
        if hasTimedOut {
            close(timedOut)
            return
        }
        switch cmd {
            case "add":
                dprintln("New Message:", params)
                cmq.messages = append(cmq.messages, params)
            case "get":
                srcPort := params[0]
                cmq.queued = append(cmq.queued, netAddress{srcAddr.Host, srcPort})
        }
        if len(cmq.messages) > 0 && len(cmq.queued) > 0 {
            dprintln("Message to be served:", cmq.messages[0])
            cmq.onInfrMsgSent()
            sendToAddress(cmq.queued[0], append([]string{"msg"}, cmq.messages[0]...)...)
            cmq.messages = cmq.messages[1:]
            cmq.queued = cmq.queued[1:]
        }
    }
}

func (tn *ClusterMessageQueue) onInfrMsgAgent() {
    if tn.perfTest {
        atomic.AddUint64(&tn.infrMessagesFromAgents, 1)
    }
}

func (tn *ClusterMessageQueue) onInfrMsgSent() {
    if tn.perfTest {
        atomic.AddUint64(&tn.infrMessagesSent, 1)
    }
}

func (tn *ClusterMessageQueue) GetInfrMsgAgent() uint64 {
    if tn.perfTest {
        return atomic.LoadUint64(&tn.infrMessagesFromAgents)
    } else {
        panic("No performance tests required!")
    }
}

func (tn *ClusterMessageQueue) GetInfrMsgSent() uint64 {
    if tn.perfTest {
        return atomic.LoadUint64(&tn.infrMessagesSent)
    } else {
        panic("No performance tests required!")
    }
}

func (cmq *ClusterMessageQueue) Terminate(){
    cmq.listener.Close()
}

///////////

type ClusterNode struct{
    messageQueueAddress string
    counterAddress string
    registrationAddress string
    listener net.Listener
    agents map[int]string
    port string
    infrMessagesFromAgents uint64
    infrMessagesSent uint64
    perfTest bool
}

func NewClusterNode(port int, messageQueueAddress string, counterAddress string, registrationAddress string) *ClusterNode {
    return NewClusterNodePerf(false, port, messageQueueAddress, counterAddress, registrationAddress)
}

func NewClusterNodePerf(perfTest bool, port int, messageQueueAddress string, counterAddress string, registrationAddress string) *ClusterNode {
    return &ClusterNode{
        messageQueueAddress: messageQueueAddress,
        counterAddress: counterAddress,
        registrationAddress: registrationAddress,
        listener: listenToPort(port),
        agents: map[int]string{},
        port: itoa(port),
        infrMessagesFromAgents: 0,
        infrMessagesSent: 0,
        perfTest: perfTest,
    }
}

func (tn *ClusterNode) WorkLoop() {
    tn.Work(0, make(chan struct{}))
}

func (cn *ClusterNode) Work(timeout int64, timedOut chan<- struct{}){
    hasTimedOut := false
    for{
        cn.onInfrMsgSent()
        sendTo(cn.messageQueueAddress, "get", cn.port)
        var reqFrom int //contains the agent id that sent the req
        for deliveredMessage := false; !deliveredMessage; {
            cmd, params := receiveWithTimeout(cn.listener, timeout, &hasTimedOut)
            if hasTimedOut {
                close(timedOut)
                return
            }
            switch cmd {
                case "msg": // a new message arrived
                    // "msg" cmd [params]
                    msgCmd := params[0]
                    msgParams := params[1:]
                    if msgCmd == "REQ" {
                        cn.onInfrMsgAgent()
                        reqFrom = atoi(msgParams[0])
                        cn.onInfrMsgSent()
                        sendTo(cn.counterAddress, "inc", cn.port)
                    } else {
                        sender := atoi(msgParams[1])
                        msgParams[1] = "0"
                        for agentId, agentAddr := range cn.agents {
                            if agentId != sender{
                                cn.onInfrMsgSent()
                                sendTo(agentAddr, params...)
                            }
                        }
                        deliveredMessage = true
                    }
                    
                case "newAgent": // a new agent arrived
                    agCompId := params[0]
                    agAddr := params[1]
                    cn.agents[atoi(agCompId)] = agAddr
                    cn.onInfrMsgSent()
                    sendTo(cn.registrationAddress, "newAgentKnown")
                    
                case "count": // a message count => a REQ was filed and I must reply with this mid
                    mid := params[0]
                    cn.onInfrMsgSent()
                    sendTo(cn.agents[reqFrom], "RPLY", mid)
                    deliveredMessage = true
            }    
        }
    }
}
func (tn *ClusterNode) onInfrMsgAgent() {
    if tn.perfTest {
        atomic.AddUint64(&tn.infrMessagesFromAgents, 1)
    }
}

func (tn *ClusterNode) onInfrMsgSent() {
    if tn.perfTest {
        atomic.AddUint64(&tn.infrMessagesSent, 1)
    }
}

func (tn *ClusterNode) GetInfrMsgAgent() uint64 {
    if tn.perfTest {
        return atomic.LoadUint64(&tn.infrMessagesFromAgents)
    } else {
        panic("No performance tests required!")
    }
}

func (tn *ClusterNode) GetInfrMsgSent() uint64 {
    if tn.perfTest {
        return atomic.LoadUint64(&tn.infrMessagesSent)
    } else {
        panic("No performance tests required!")
    }
}

func (cn *ClusterNode) Terminate(){
    cn.listener.Close()
}
