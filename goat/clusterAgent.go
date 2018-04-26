package goat

import(
    "net"
    "time"
    "sync"
    //"fmt"
)

type ClusterAgent struct{
    messageQueueAddress string
    registrationAddress string
    componentId int
    firstMessageId int
    chnGetMid *unboundChanUnit
    chnMids *unboundChanInt
    chnMessagesIn *unboundChanMessage
    chnMessagesOut chan Message
    listeningPort int
    listener net.Listener
    maxMid int
    chnReceiveTime *unboundChanMT
    chnSendTime *unboundChanMT
    lockST *sync.Mutex
    receiveTime map[int]int64
    sendTime map[int]int64
}

func NewClusterAgent(messageQueueAddress string, registrationAddress string) *ClusterAgent{
    ca := ClusterAgent{
        messageQueueAddress: messageQueueAddress, 
        registrationAddress: registrationAddress,
        chnGetMid: newUnboundChanUnit(),
        chnMids: newUnboundChanInt(),
        chnMessagesIn: newUnboundChanMessage(),
        chnMessagesOut: make(chan Message),
        maxMid: -1,
        firstMessageId: -1,
        chnReceiveTime: newUnboundChanMT(),
        chnSendTime: newUnboundChanMT(),
        lockST: &sync.Mutex{},
        receiveTime: map[int]int64{},
        sendTime: map[int]int64{},
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
                //fmt.Println("Got RPLY", params[0])
                mid := atoi(params[0])
                ca.chnMids.In <- mid
                
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
                    rtime := time.Now().UnixNano()
                    ca.lockST.Lock()
                    if mid > ca.maxMid{
                        ca.maxMid = mid
                    } 
                    ca.lockST.Unlock()
                    ca.chnReceiveTime.In <- msgTime{mid, rtime}
                    ca.chnMessagesIn.In <- inMsg
                }
        }
    }
}

func (ca *ClusterAgent) doOutcomingProcess() {

    //Work
    for {
        select {
            case msgToSend := <- ca.chnMessagesOut:
                stime := time.Now().UnixNano()
                sendTo(ca.messageQueueAddress, "add", "DATA", itoa(msgToSend.Id), itoa(ca.componentId), msgToSend.Pred.String(), msgToSend.Message.encode() )
                ca.lockST.Lock()
                if msgToSend.Id >= ca.maxMid {
                    ca.maxMid = msgToSend.Id
                }
                ca.lockST.Unlock()
                ca.chnSendTime.In <- msgTime{msgToSend.Id, stime}
            case <- ca.chnGetMid.Out:
                sendTo(ca.messageQueueAddress, "add", "REQ", itoa(ca.componentId))
        }
    }
}

func (ca *ClusterAgent) GetReceiveTime() map[int]int64{
    ca.chnReceiveTime.Close()
    return toMapIntInt64(&ca.receiveTime, ca.chnReceiveTime)
}

func (ca *ClusterAgent) GetSendTime() map[int]int64{
    ca.chnSendTime.Close()
    return toMapIntInt64(&ca.sendTime, ca.chnSendTime)
}

func (ca *ClusterAgent) GetMaxMid() int {
    ca.lockST.Lock()
    out := ca.maxMid
    ca.lockST.Unlock()
    return out 
}

func (ca *ClusterAgent) SendMessage(msg Message){
    ca.chnMessagesOut <- msg
}
func (ca *ClusterAgent) AskMid(){
    ca.chnGetMid.In <- struct{}{}
}
func (ca *ClusterAgent) GetRplyChan() *unboundChanInt {
    return ca.chnMids
}
func (ca *ClusterAgent) GetDataChan() *unboundChanMessage {
    return ca.chnMessagesIn
}
