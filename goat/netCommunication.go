package goat

import(
    "net"
    "fmt"
    "strings"
    "bufio"
)

type outMessage struct {
    id int
    message string
    predicate ClosedPredicate
}

type inMessage struct {
    id int
    message string
    predicate ClosedPredicate
    componentId int
}

type netCommunication struct {
    componentId int
    firstMessageId int
    chnGetMid chan bool
    chnMids chan int
    chnOutbox chan outMessage
    chnInbox chan inMessage
    server string
    listeningPort int
    listener net.Listener
}

func (nc *netCommunication) sendToServer(tokens... string) {
    escTokens := make([]string, len(tokens))
    for i, tok:= range tokens {
        escTokens[i] = escape(tok)
    }
    conn, err := net.Dial("tcp", nc.server)
    if err == nil{
        fmt.Fprintf(conn, "%s\n", strings.Join(escTokens," "))
    }
}   

func (nc *netCommunication) receiveFromServer() (string, []string) {
    conn, err := nc.listener.Accept()
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

func netCommunicationInitAndRun(serverAddress string) *netCommunication{
    nc := netCommunication{
        chnGetMid: make(chan bool),
        chnMids: make(chan int, 5),
        chnOutbox: make(chan outMessage, 5),
        chnInbox: make(chan inMessage, 5),
        server: serverAddress,
    }
    
    nc.listener, _ = net.Listen("tcp", ":0")
    myAddressPort := nc.listener.Addr().String()
    portIndex := strings.LastIndex(myAddressPort, ":")
    nc.listeningPort = atoi(myAddressPort[portIndex+1:])
    
    chnRegistered := make(chan bool, 1)
    
    go nc.doIncomingProcess(chnRegistered)
    go nc.doOutcomingProcess()
    <- chnRegistered
    
    return &nc
}

func (nc *netCommunication) doIncomingProcess(chnRegistered chan<- bool) {
    for {
        cmd, params := nc.receiveFromServer()
        switch cmd {
            case "Registered":
                nc.componentId = atoi(params[0])
                nc.firstMessageId = atoi(params[1])
                close(chnRegistered)
            case "RPLY":
                mid := atoi(params[0])
                nc.chnMids <- mid
                
            case "DATA":
                pred, _ := ToPredicate(params[2])
                mid := atoi(params[0])
                cid := atoi(params[1])
                inMsg := inMessage {
                    id: mid,
                    componentId: cid,
                    predicate: pred,
                    message: params[3],
                }
                nc.chnInbox <- inMsg
        }
    }
}

func (nc *netCommunication) doOutcomingProcess() {
    //Register
    nc.sendToServer("Register", itoa(nc.listeningPort))

    //Work
    for {
        select {
        	// TODO: send only when nid >= msg.id
            case msgToSend := <- nc.chnOutbox:
                nc.sendToServer("DATA", itoa(msgToSend.id), itoa(nc.componentId), msgToSend.predicate.String(), msgToSend.message )
            case <- nc.chnGetMid:
                nc.sendToServer("REQ", itoa(nc.componentId))
        }
    }
}

func (nc *netCommunication) getMessageId() int{
    nc.chnGetMid <- true
    return <- nc.chnMids
}
