package goat

import (
    "net"
    "sync/atomic"
)

type ClusterCounter struct {
    listener net.Listener
    count int
    infrMessagesFromAgents uint64
    infrMessagesSent uint64
    perfTest bool
}

func NewClusterCounter(port int) *ClusterCounter{
    return NewClusterCounterPerf(false, port)
}

func NewClusterCounterPerf(perfTest bool, port int) *ClusterCounter{
    return &ClusterCounter{
        listener: listenToPort(port),
        perfTest: perfTest,
        infrMessagesFromAgents: 0,
        infrMessagesSent: 0,
    }
}

func (tn *ClusterCounter) onInfrMsgAgent() {
    if tn.perfTest {
        atomic.AddUint64(&tn.infrMessagesFromAgents, 1)
    }
}

func (tn *ClusterCounter) onInfrMsgSent() {
    if tn.perfTest {
        atomic.AddUint64(&tn.infrMessagesSent, 1)
    }
}

func (tn *ClusterCounter) GetInfrMsgAgent() uint64 {
    if tn.perfTest {
        return atomic.LoadUint64(&tn.infrMessagesFromAgents)
    } else {
        panic("No performance tests required!")
    }
}

func (tn *ClusterCounter) GetInfrMsgSent() uint64 {
    if tn.perfTest {
        return atomic.LoadUint64(&tn.infrMessagesSent)
    } else {
        panic("No performance tests required!")
    }
}

func (cc *ClusterCounter) Work(timeout int64, timedOut chan<- struct{}){
    hasTimedOut := false
    for {
        cmd, params, srcAddr := receiveWithAddressTimeout(cc.listener, timeout, &hasTimedOut)
        if hasTimedOut {
            close(timedOut)
            return
        }
        switch cmd {
            case "read": // ask, it will not be used
                rplPort := params[0]
                rplAddress := netAddress{srcAddr.Host, rplPort}
                cc.onInfrMsgSent()
                sendToAddress(rplAddress, "count", itoa(cc.count))
            case "inc": // it will be assigned to a message
                rplPort := params[0]
                rplAddress := netAddress{srcAddr.Host, rplPort}
                cc.onInfrMsgSent()
                sendToAddress(rplAddress, "count", itoa(cc.count))
                cc.count++
        }    
    }
}

func (cc *ClusterCounter) Terminate(){
    cc.listener.Close()
}
