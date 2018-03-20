package goat

type Agent interface{
    //GetMessageId() int
    //
    //Inbox() <-chan Message
    //Outbox() chan<- Message
    
    GetComponentId() int
    Start()
    GetFirstMessageId() int
    SendMessage(Message)
    AskMid()
    GetRplyChan() *unboundChanInt
    GetDataChan() *unboundChanMessage
    GetMaxMid() int
    GetSendTime() map[int]int64
    GetReceiveTime() map[int]int64
}
