package goat

import(
    "net"
)

type RingAgent struct{
    nodeAddress netAddress
    registrationAddress string
    componentId int
    firstMessageId int
    chnGetMid chan bool
    chnMids chan int
    chnOutbox chan Message
    chnInbox chan Message
    listeningPort int
    listener net.Listener
}

func NewRingAgent(registrationAddress string) *RingAgent{
    ca := RingAgent{
        registrationAddress: registrationAddress,
        chnGetMid: make(chan bool),
        chnMids: make(chan int, 5),
        chnOutbox: make(chan Message, 5),
        chnInbox: make(chan Message, 5),
        firstMessageId: -1,
    }
    return &ca
}

func (ca *RingAgent) Start(){
    ca.listener, ca.listeningPort = listenToRandomPort()
    
    chnRegistered := make(chan struct{}, 1) // TODO remove 1
    
    //Register
    sendTo(ca.registrationAddress, "Register", itoa(ca.listeningPort))
    go ca.doIncomingProcess(chnRegistered)
    <- chnRegistered
    go ca.doOutcomingProcess()
}

func (ca *RingAgent) Inbox() <-chan Message{
    return ca.chnInbox
}

func (ca *RingAgent) Outbox() chan<- Message{
    return ca.chnOutbox
}

func (ca *RingAgent) GetComponentId() int{
    return ca.componentId
}

func (ca *RingAgent) GetFirstMessageId() int{
    return ca.firstMessageId
}

func (ca *RingAgent) doIncomingProcess(chnRegistered chan<- struct{}) {
    for {
        cmd, params, srcAddr := receiveWithAddress(ca.listener)
        switch cmd {
            case "Registered":
                ca.componentId = atoi(params[0])
                ca.firstMessageId = atoi(params[1])
                nodePort := params[2]
                ca.nodeAddress = netAddress{srcAddr.Host, nodePort}
                close(chnRegistered)
            case "RPLY":
                mid := atoi(params[0])
                ca.chnMids <- mid
                
            case "DATA":
                pred, _ := ToPredicate(params[2])
                mid := atoi(params[0])
                if ca.firstMessageId >= 0 && mid >= ca.firstMessageId {
                    //cid := atoi(params[1])
                    inMsg := Message {
                        Id: mid,
                        //componentId: cid,
                        Pred: pred,
                        Message: decodeTuple(params[3]),
                    }
                    ca.chnInbox <- inMsg
                }
        }
    }
}

func (ca *RingAgent) doOutcomingProcess() {

    //Work
    for {
        select {
            case msgToSend := <- ca.chnOutbox:
                sendToAddress(ca.nodeAddress, "DATA", itoa(msgToSend.Id), itoa(ca.componentId), msgToSend.Pred.String(), msgToSend.Message.encode() )
            case <- ca.chnGetMid:
                sendToAddress(ca.nodeAddress, "REQ", itoa(ca.componentId))
        }
    }
}

func (ca *RingAgent) GetMessageId() int{
    ca.chnGetMid <- true
    mid := <- ca.chnMids
    if ca.firstMessageId < 0{
        panic("Got a mid before getting the first message id!")
    }
    if mid < ca.firstMessageId {
        panic("Got a mid < first message id!")
    }
    return mid
}
