package goat

type Message struct {
    Id int
    Message Tuple
    Pred ClosedPredicate
}

type Agent interface{
    GetMessageId() int
    GetFirstMessageId() int
    Inbox() <-chan Message
    Outbox() chan<- Message
    GetComponentId() int
    Start()
}
