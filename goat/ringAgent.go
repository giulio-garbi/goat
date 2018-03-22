package goat

import "time"
import "sync"

type msgTime struct {
    id int
    tm int64
}

type RingAgent struct{
    registrationAddress string
    componentId int
    firstMessageId int
    maxMid int
    listeningPort int
    listener *unboundChanConn
    chnMids *unboundChanInt
    chnMessagesIn *unboundChanMessage
    chnMessagesOut chan Message
    receiveTime map[int]int64
    sendTime map[int]int64
    chnReceiveTime *unboundChanMT
    chnSendTime *unboundChanMT
    lockST *sync.Mutex
    chnGetMid *unboundChanUnit
}

func NewRingAgent(registrationAddress string) *RingAgent{
    ca := RingAgent{
        registrationAddress: registrationAddress,
        chnMids: newUnboundChanInt(),
        chnMessagesIn: newUnboundChanMessage(),
        chnMessagesOut: make(chan Message),
        firstMessageId: -1,
        receiveTime: map[int]int64{},
        sendTime: map[int]int64{},
        chnReceiveTime: newUnboundChanMT(),
        chnSendTime: newUnboundChanMT(),
        lockST: &sync.Mutex{},
        chnGetMid: newUnboundChanUnit(),
    }
    return &ca
}

func (ca *RingAgent) Start(){
    var chnReady chan(struct{})
    ca.listener, chnReady, ca.listeningPort = listenerRandomPort()
    <-chnReady 
    
    connReg := connectWith(ca.registrationAddress)
    connReg.Send("Register", itoa(ca.listeningPort))
    
    connNode := <- ca.listener.Out
    _, params := connNode.Receive()
    ca.componentId = atoi(params[0])
    ca.firstMessageId = atoi(params[1])
    ca.maxMid = ca.firstMessageId
    dprintln("Starting at mid", ca.firstMessageId)
    
    go func() {
        for {
            cmd, params := connNode.Receive()
            switch(cmd) {
                case "RPLY":
                    mid := atoi(params[0])
                    ca.chnMids.In <- mid
                    dprintln("r",mid,ca.componentId)
                    
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
                        dprintln(inMsg, ca.componentId)
                    }
            }
        }
    }()
    go func(){
        for {
            select {
                case msgToSend := <- ca.chnMessagesOut:
                    stime := time.Now().UnixNano()
                    connNode.Send("DATA", itoa(msgToSend.Id), itoa(ca.componentId), msgToSend.Pred.String(), msgToSend.Message.encode() )
                    dprintln("+", msgToSend)
                    ca.lockST.Lock()
                    if msgToSend.Id > ca.maxMid{
                        ca.maxMid = msgToSend.Id
                    } 
                    ca.lockST.Unlock()
                    ca.chnSendTime.In <- msgTime{msgToSend.Id, stime}
                case <- ca.chnGetMid.Out:
                    connNode.Send("REQ", itoa(ca.componentId))
                    dprintln("R?")
            }
        }
    }()
}

func (ca *RingAgent) SendMessage(msg Message){
    ca.chnMessagesOut <- msg
}
func (ca *RingAgent) AskMid(){
    ca.chnGetMid.In <- struct{}{}
}
func (ca *RingAgent) GetRplyChan() *unboundChanInt {
    return ca.chnMids
}
func (ca *RingAgent) GetDataChan() *unboundChanMessage {
    return ca.chnMessagesIn
}

func (ca *RingAgent) GetComponentId() int{
    return ca.componentId
}

func (ca *RingAgent) GetFirstMessageId() int{
    return ca.firstMessageId
}


func toMapIntInt64(m *map[int]int64, c *unboundChanMT) map[int]int64 {
    for quit:=false; !quit; {
        select {
            case mt, stillOpen := <-c.Out :
                if stillOpen{
                    (*m)[mt.id] = mt.tm
                } else {
                    quit = true
                }
        }
    }
    return *m
}

func (ca *RingAgent) GetReceiveTime() map[int]int64{
    ca.chnReceiveTime.Close()
    return toMapIntInt64(&ca.receiveTime, ca.chnReceiveTime)
}

func (ca *RingAgent) GetSendTime() map[int]int64{
    ca.chnSendTime.Close()
    return toMapIntInt64(&ca.sendTime, ca.chnSendTime)
}

func (ca *RingAgent) GetMaxMid() int {
    ca.lockST.Lock()
    out := ca.maxMid
    ca.lockST.Unlock()
    return out 
}
