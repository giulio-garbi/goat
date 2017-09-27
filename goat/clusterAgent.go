package goat

import(
    "net"
)

type ClusterAgent struct{
    messageQueueAddress string
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

func NewClusterAgent(messageQueueAddress string, registrationAddress string) *ClusterAgent{
    ca := ClusterAgent{
        messageQueueAddress: messageQueueAddress, 
        registrationAddress: registrationAddress,
        chnGetMid: make(chan bool),
        chnMids: make(chan int, 5),
        chnOutbox: make(chan Message, 5),
        chnInbox: make(chan Message, 5),
        firstMessageId: -1,
    }
    return &ca
}

func (ca *ClusterAgent) Start(){
    ca.listener, ca.listeningPort = listenToRandomPort()
    
    chnRegistered := make(chan struct{}, 1) // TODO remove 1
    
    //Register
    sendTo(ca.registrationAddress, "Register", itoa(ca.listeningPort))
    go ca.doIncomingProcess(chnRegistered)
    <- chnRegistered
    go ca.doOutcomingProcess()
}

func (ca *ClusterAgent) Inbox() <-chan Message{
    return ca.chnInbox
}

func (ca *ClusterAgent) Outbox() chan<- Message{
    return ca.chnOutbox
}

func (ca *ClusterAgent) GetComponentId() int{
    return ca.componentId
}

func (ca *ClusterAgent) GetFirstMessageId() int{
    return ca.firstMessageId
}

func (ca *ClusterAgent) doIncomingProcess(chnRegistered chan<- struct{}) {
    for {
        cmd, params := receive(ca.listener)
        switch cmd {
            case "Registered":
                ca.componentId = atoi(params[0])
                ca.firstMessageId = atoi(params[1])
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

func (ca *ClusterAgent) doOutcomingProcess() {

    //Work
    for {
        select {
            case msgToSend := <- ca.chnOutbox:
                sendTo(ca.messageQueueAddress, "add", "DATA", itoa(msgToSend.Id), itoa(ca.componentId), msgToSend.Pred.String(), msgToSend.Message.encode() )
            case <- ca.chnGetMid:
                sendTo(ca.messageQueueAddress, "add", "REQ", itoa(ca.componentId))
        }
    }
}

func (ca *ClusterAgent) GetMessageId() int{
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
