package goat

import(
    "net"
    "fmt"
    "strings"
    "bufio"
)

type SingleServerAgent struct{
    componentId int
    firstMessageId int
    chnGetMid chan bool
    chnMids chan int
    chnOutbox chan Message
    chnInbox chan Message
    server string
    listeningPort int
    listener net.Listener
}

func NewSingleServerAgent(serverAddress string) *SingleServerAgent{
    ssa := SingleServerAgent{
        chnGetMid: make(chan bool),
        chnMids: make(chan int, 5),
        chnOutbox: make(chan Message, 5),
        chnInbox: make(chan Message, 5),
        server: serverAddress,
    }
    
    return &ssa
}

func (ssa *SingleServerAgent) Start(){
    ssa.listener, _ = net.Listen("tcp", ":0")
    myAddressPort := ssa.listener.Addr().String()
    portIndex := strings.LastIndex(myAddressPort, ":")
    ssa.listeningPort = atoi(myAddressPort[portIndex+1:])
    
    chnRegistered := make(chan bool, 1)
    
    go ssa.doIncomingProcess(chnRegistered)
    go ssa.doOutcomingProcess()
    <- chnRegistered
}

func (ssa *SingleServerAgent) Inbox() <-chan Message{
    return ssa.chnInbox
}

func (ssa *SingleServerAgent) Outbox() chan<- Message{
    return ssa.chnOutbox
}

func (ssa *SingleServerAgent) GetComponentId() int{
    return ssa.componentId
}

func (ssa *SingleServerAgent) GetFirstMessageId() int{
    return ssa.firstMessageId
}

func (ssa *SingleServerAgent) doIncomingProcess(chnRegistered chan<- bool) {
    for {
        cmd, params := ssa.receiveFromServer()
        switch cmd {
            case "Registered":
                ssa.componentId = atoi(params[0])
                ssa.firstMessageId = atoi(params[1])
                close(chnRegistered)
            case "RPLY":
                mid := atoi(params[0])
                ssa.chnMids <- mid
                
            case "DATA":
                pred, _ := ToPredicate(params[2])
                mid := atoi(params[0])
                //cid := atoi(params[1])
                inMsg := Message {
                    Id: mid,
                    //componentId: cid,
                    Pred: pred,
                    Message: decodeTuple(params[3]),
                }
                ssa.chnInbox <- inMsg
        }
    }
}

func (ssa *SingleServerAgent) doOutcomingProcess() {
    //Register
    ssa.sendToServer("Register", itoa(ssa.listeningPort))

    //Work
    for {
        select {
        	// TODO: send only when nid >= msg.id
            case msgToSend := <- ssa.chnOutbox:
                ssa.sendToServer("DATA", itoa(msgToSend.Id), itoa(ssa.componentId), msgToSend.Pred.String(), msgToSend.Message.encode() )
            case <- ssa.chnGetMid:
                ssa.sendToServer("REQ", itoa(ssa.componentId))
        }
    }
}

func (ssa *SingleServerAgent) GetMessageId() int{
    ssa.chnGetMid <- true
    return <- ssa.chnMids
}

func (ssa *SingleServerAgent) sendToServer(tokens... string) {
    escTokens := make([]string, len(tokens))
    for i, tok:= range tokens {
        escTokens[i] = escape(tok)
    }
    conn, err := net.Dial("tcp", ssa.server)
    if err == nil{
        fmt.Fprintf(conn, "%s\n", strings.Join(escTokens," "))
    }
}   

func (ssa *SingleServerAgent) receiveFromServer() (string, []string) {
    conn, err := ssa.listener.Accept()
    _ = err
    serverMsg, err := bufio.NewReader(conn).ReadString('\n')
    if err == nil {
        escTokens := strings.Split(serverMsg[:len(serverMsg)-1], " ")
        tokens := make([]string, len(escTokens))
        for i, escTok := range escTokens {
            tokens[i], _ = unescape(escTok, 0)
        }
        return tokens[0], tokens[1:]
    } else {
        //TODO Error
        return "", []string{}
    }
}
