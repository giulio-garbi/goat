package goat

import (
    "net"
)

type ClusterCounter struct {
    listener net.Listener
    count int
}

func NewClusterCounter(port int) *ClusterCounter{
    return &ClusterCounter{
        listener: listenToPort(port),
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
                sendToAddress(rplAddress, "count", itoa(cc.count))
            case "inc": // it will be assigned to a message
                rplPort := params[0]
                rplAddress := netAddress{srcAddr.Host, rplPort}
                sendToAddress(rplAddress, "count", itoa(cc.count))
                cc.count++
        }    
    }
}

func (cc *ClusterCounter) Terminate(){
    cc.listener.Close()
}
