package goat

import (
    "net"
    "math/rand"
    "time"
)

// Contains the set of registered agents, and assigns them to the nodes
type RingAgentRegistration struct {
    listener net.Listener
    nodesAddresses []string
    queuedAgents []netAddress
    port string
    compId int
}

func NewRingAgentRegistration(port int, nodesAddresses []string) *RingAgentRegistration{
    return &RingAgentRegistration{
        listener: listenToPort(port),
        nodesAddresses: nodesAddresses,
        queuedAgents: make([]netAddress, 0),
        port: itoa(port),
    }
}

func (rar *RingAgentRegistration) Work(timeout int64, timedOut chan<- struct{}){
    hasTimedOut := false
    src := rand.NewSource(time.Now().Unix())
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
            ndAddr := rar.nodesAddresses[rand.New(src).Intn(len(rar.nodesAddresses))]
            sendTo(ndAddr, "newAgent", agCompId, agAddr.String())
            rar.queuedAgents = rar.queuedAgents[1:]
        }
    }
}

func (rar *RingAgentRegistration) Terminate(){
    rar.listener.Close()
}

///////////

type RingNode struct{
    counterAddress string
    listener net.Listener
    agents map[int]string
    port string
    messages map[int][]string
    nid int
    nextNodeAddress string
}

func NewRingNode(port int, counterAddress string, nextNodeAddress string) *RingNode {
    return &RingNode{
        counterAddress: counterAddress,
        listener: listenToPort(port),
        agents: map[int]string{},
        port: itoa(port),
        messages: map[int][]string{},
        nid: 0,
        nextNodeAddress: nextNodeAddress,
    }
}

func (rn *RingNode) Work(timeout int64, timedOut chan<- struct{}){
    hasTimedOut := false
    reqFrom := make([]int,0) //contains the agents id that sent the reqs
    for{
        cmd, params := receiveWithTimeout(rn.listener, timeout, &hasTimedOut)
        if hasTimedOut {
            close(timedOut)
            return
        }
        switch cmd {
            case "REQ":
                reqFrom = append(reqFrom, atoi(params[0]))
                sendTo(rn.counterAddress, "inc", rn.port)

            case "DATA":
                msgId := atoi(params[0])
                if msgId >= rn.nid{
                    rn.messages[msgId] = append([]string{cmd}, params...)
                    
                    for ; len(rn.messages[rn.nid])>0; rn.nid++{
                        mParams := rn.messages[rn.nid][1:]
                        sender := atoi(mParams[1])
                        mParams[1] = "0" //anonimity
                        for agentId, agentAddr := range rn.agents {
                            if agentId != sender{
                                sendTo(agentAddr, rn.messages[rn.nid]...)
                            }
                        }
                        mParams[1] = itoa(sender) // reset before forwarding
                        sendTo(rn.nextNodeAddress, rn.messages[rn.nid]...)
                        delete(rn.messages, rn.nid)
                    }
                }
                
            case "newAgent": // a new agent arrived
                agCompId := params[0]
                agAddr := params[1]
                rn.agents[atoi(agCompId)] = agAddr
                sendTo(agAddr, "Registered", agCompId, itoa(rn.nid), rn.port)
                
            case "count": // a message count => a REQ was filed and I must reply with this mid
                mid := params[0]
                sendTo(rn.agents[reqFrom[0]], "RPLY", mid)
                reqFrom = reqFrom[1:]
        }    
    }
}

func (rn *RingNode) Terminate(){
    rn.listener.Close()
}
