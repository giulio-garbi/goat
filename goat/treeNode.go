package goat

import (
    "net"
    "fmt"
    "strings"
)

// Contains the set of registered agents, and assigns them to the nodes
type TreeAgentRegistration = RingAgentRegistration
func NewTreeAgentRegistration(port int, nodesAddresses []string) *TreeAgentRegistration{
    return NewRingAgentRegistration(port, nodesAddresses)
}

type TreeNode struct{
    counter int //only for the root
    listener net.Listener
    agents map[int]netAddress
    port string
    messages map[int]tnMessageToForward
    nid int
    parentAddress string //except the root, which has ""
    childNodesAddresses []string
}

type tnMessageToForward struct{
    message []string
    fromTheParent bool
    sourceAgent int
    sourceDescendant string
}

func NewTreeNode(port int, parentAddress string, childNodesAddresses []string) *TreeNode {
    return &TreeNode{
        counter: 0,
        listener: listenToPort(port),
        agents: map[int]netAddress{},
        port: itoa(port),
        messages: map[int]tnMessageToForward{},
        nid: 0,
        parentAddress: parentAddress,
        childNodesAddresses: childNodesAddresses,
    }
}

func (tn *TreeNode) getMessageToForward(srcAddr netAddress, params []string) tnMessageToForward{
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
            sourceDescendant: "",
        }
    } else {
        na := netAddress{srcAddr.Host, src[1:]}
        sourceReplyAddress := na.String()
        if sourceReplyAddress == tn.parentAddress {
            return tnMessageToForward{
                message: append([]string{"DATA"}, params...),
                fromTheParent: true,
                sourceAgent: -1,
                sourceDescendant: "",
            }
        } else {
            return tnMessageToForward{
                message: append([]string{"DATA"}, params...),
                fromTheParent: false,
                sourceAgent: -1,
                sourceDescendant: sourceReplyAddress,
            }
        }
    }
}

func (tn *TreeNode) prepareMessageForInfrastructure(msg tnMessageToForward)[]string{
    msg.message[2] = fmt.Sprintf("P%s", tn.port)
    return msg.message
}

func (tn *TreeNode) prepareMessageForAgent(msg tnMessageToForward)[]string{
    msg.message[2] = "0"
    return msg.message
}

func (tn *TreeNode) amRoot() bool{
    return tn.parentAddress == ""
}

func (tn *TreeNode) resolveLastAddress(seq []string) (netAddress, []string){
    //For info see in Work()
    if len(seq) == 1{ // is a component id
        return tn.agents[atoi(seq[0])], []string{}
    } else {
        return netAddress{seq[len(seq)-1], seq[len(seq)-2]}, seq[:len(seq)-2]
    }
}

func (tn *TreeNode) Work(timeout int64, timedOut chan<- struct{}){
    hasTimedOut := false
    //reqFrom := make([]int,0) //contains the agents id that sent the reqs
    for{
        cmd, params, srcAddr := receiveWithAddressTimeout(tn.listener, timeout, &hasTimedOut)
        if hasTimedOut {
            close(timedOut)
            return
        }
        switch cmd {
            case "REQ": // REQ compId port0 addr0 port1 addr1 ... port_n-1 addr_n-1 port_n
                        // NB: I am addr_n+1
                /*
                    node       gets
                    leaf node: REQ compId
                    parent " : REQ compId port0
                    parent " : REQ compId port0 addr0 port1
                    root     : REQ compId port0 addr0 port1 addr1 port2 
                    ===> NB it will be given to resolveLastAddress with addr2 appended at the end, unless it is a compId
                    child "  : RPLY mid compId port0 addr0 port1 addr1
                    child "  : RPLY mid compId port0 addr0
                    child "  : RPLY mid compId 
                    compId   : RPLY mid
                */
                path := params
                var corrPath []string
                if len(path) == 1{
                    corrPath = path
                } else {
                    corrPath = append(path, srcAddr.Host)
                }
                    
                if tn.amRoot(){
                    assMid := itoa(tn.counter)
                    tn.counter++
                    childAddress, remainder := tn.resolveLastAddress(corrPath)
                    sendToAddress(childAddress, append([]string{"RPLY", assMid}, remainder...)...)
                } else {
                    if len(path) == 1{
                        sendTo(tn.parentAddress, append([]string{"REQ"}, append(path, tn.port)...)...)
                    } else {
                        sendTo(tn.parentAddress, append([]string{"REQ"}, append(path, srcAddr.Host, tn.port)...)...)
                    }
                }
                
            case "RPLY": // RPLY mid compId port0 addr0 port1 addr1 ... port_n-1 addr_n-1 port_n addr_n
                         // NB: I am addr_n+1
                assMid := params[0]
                path := params[1:]
                childAddress, remainder := tn.resolveLastAddress(path)
                sendToAddress(childAddress,append([]string{"RPLY", assMid}, remainder...)...)

            case "DATA": // DATA mid src pred msg
                msg := tn.getMessageToForward(srcAddr,params)
                
                if !tn.amRoot() && !msg.fromTheParent {
                    sendTo(tn.parentAddress, tn.prepareMessageForInfrastructure(msg)...)
                }
                
                msgId := atoi(params[0])
                tn.messages[msgId] = msg
                
                for{
                    if mFwd, has := tn.messages[tn.nid]; has{
                        delete(tn.messages, tn.nid)
                        
                        mFwdAgent := tn.prepareMessageForAgent(mFwd)
                        for agentId, agentAddr := range tn.agents {
                            if agentId != mFwd.sourceAgent{
                                sendToAddress(agentAddr, mFwdAgent...)
                            }
                        }
                        
                        mFwdInfr := tn.prepareMessageForInfrastructure(mFwd)
                        for _, node := range tn.childNodesAddresses {
                            if node != mFwd.sourceDescendant{
                                sendTo(node, mFwdInfr...)
                            }
                        }
                        tn.nid++
                    } else {
                        break
                    }
                }
                
            case "newAgent": // a new agent arrived
                agCompId := params[0]
                agAddr := params[1]
                tn.agents[atoi(agCompId)] = newNetAddress(agAddr)
                sendTo(agAddr, "Registered", agCompId, itoa(tn.nid), tn.port)
        }    
    }
}

func (tn *TreeNode) Terminate(){
    tn.listener.Close()
}
