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
    server string
    listeningPort int
    listener net.Listener
    chnMids *unboundChanInt
    chnMessagesIn *unboundChanMessage
    chnMessagesOut chan Message
    chnGetMid *unboundChanUnit
    inStrings *unboundChanString
    
    serverOutConn net.Conn
    serverInConn *bufio.Reader
}


func NewSingleServerAgent(serverAddress string) *SingleServerAgent{
    ssa := SingleServerAgent{
        chnGetMid: newUnboundChanUnit(),
        chnMids: newUnboundChanInt(),
        //chnOutbox: make(chan Message, 5),
        //chnInbox: make(chan Message, 5),
        server: serverAddress,
        
        //inStrings: newUnboundChanString(),
        chnMessagesIn: newUnboundChanMessage(),
        chnMessagesOut: make(chan Message),
        inStrings: newUnboundChanString(),
    }
    
    return &ssa
}

func (ssa *SingleServerAgent) Start(){
    ssa.listener, _ = net.Listen("tcp", ":0")
    myAddressPort := ssa.listener.Addr().String()
    portIndex := strings.LastIndex(myAddressPort, ":")
    ssa.listeningPort = atoi(myAddressPort[portIndex+1:])
    
    chnRegistered := make(chan bool, 1)
    
    go func(){ssa.doIncomingProcess(chnRegistered)}()
    go func(){ssa.doOutcomingProcess()}()
    <- chnRegistered
}

func (ssa *SingleServerAgent) GetComponentId() int{
    return ssa.componentId
}

func (ssa *SingleServerAgent) GetFirstMessageId() int{
    return ssa.firstMessageId
}

func (ssa *SingleServerAgent) doIncomingProcess(chnRegistered chan<- bool) {
    /*go func(){
        for {
            fmt.Println(ssa.componentId, "?")
            conn, err := ssa.listener.Accept()
            fmt.Println(ssa.componentId, "!")
            _ = err
            if err != nil {
                panic(err)
            } else {
                dprintln(ssa.componentId, "P?")
                //for{
                    serverMsg, err := bufio.NewReader(conn).ReadString('\n')
                    //ioutil.ReadAll(conn)
                    
                    dprintln(ssa.componentId, "P!")
                    if err != nil {
                        //fmt.Println(err)
                    //    break
                    } else {
                        fmt.Println(ssa.componentId,"|",serverMsg)
                        ssa.inStrings.In <- serverMsg
                        fmt.Println(ssa.componentId,"!|",serverMsg)
                    }
                //}
                conn.Close()
            }
        }
    }()*/
    conn, _ := ssa.listener.Accept()
    ssa.serverInConn = bufio.NewReader(conn)
    for {
        dprintln(ssa.componentId,"IP+")
        cmd, params := ssa.receiveFromServer()
        dprintln(ssa.componentId,"IP-")
        switch cmd {
            case "Registered":
                ssa.componentId = atoi(params[0])
                ssa.firstMessageId = atoi(params[1])
                close(chnRegistered)
            case "RPLY":
                mid := atoi(params[0])
                dprintln(itoa(ssa.componentId), "got MID",mid)
                dprintln(ssa.componentId,"M+")
                ssa.chnMids.In <- mid
                dprintln(ssa.componentId,"M-")
                
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
                dprintln("<-", mid)
                dprintln(ssa.componentId,"D+")
                ssa.chnMessagesIn.In <- inMsg
                dprintln(ssa.componentId,"D-")
        }
    }
}

func (ssa *SingleServerAgent) doOutcomingProcess() {
    //dprintln("Try dialing:", escTokens)
    conn, _ := net.Dial("tcp", ssa.server)
    ssa.serverOutConn = conn
    //Register
    ssa.sendToServer("Register", itoa(ssa.listeningPort))

    //Work
    for {
        dprintln("Ready!")
        select {
        	// TODO: send only when nid >= msg.id
            case msgToSend := <- ssa.chnMessagesOut:
                dprintln("OutMsg",msgToSend)
                ssa.sendToServer("DATA", itoa(msgToSend.Id), itoa(ssa.componentId), msgToSend.Pred.String(), msgToSend.Message.encode() )
            case <- ssa.chnGetMid.Out:
                dprintln(itoa(ssa.componentId), "asking for MID")
                ssa.sendToServer("REQ", itoa(ssa.componentId))
        }
    }
}

func (ssa *SingleServerAgent) GetMessageId() int{
    ssa.chnGetMid.In <- struct{}{}
    //return <- ssa.chnMids.Out
    return -1
}

func (ssa *SingleServerAgent) sendToServer(tokens... string) {
    escTokens := make([]string, len(tokens))
    for i, tok:= range tokens {
        escTokens[i] = escape(tok)
    }
    /*dprintln("Try dialing:", escTokens)
    conn, err := net.Dial("tcp", ssa.server)*/
    dprintln("Try:", escTokens)
    if n, err := fmt.Fprintf(ssa.serverOutConn, "%s\n", strings.Join(escTokens," ")); err != nil{
        panic(err)
    } else {
        dprintln("Conn:",n)
    }
}   

func (ssa *SingleServerAgent) SendMessage(msg Message) {
    ssa.chnMessagesOut <- msg
}

func (ssa *SingleServerAgent) AskMid(){
    ssa.chnGetMid.In <- struct{}{}
}

func (ssa *SingleServerAgent) GetRplyChan() *unboundChanInt{
    return ssa.chnMids
    
}
func (ssa *SingleServerAgent) GetDataChan() *unboundChanMessage {
    return ssa.chnMessagesIn
}

func (ssa *SingleServerAgent) receiveFromServer() (string, []string) {
    /*conn, err := ssa.listener.Accept()
    _ = err
    if err != nil {
        panic(err)
    }
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
    }*/
  
    //serverMsg := <- ssa.inStrings.Out
    serverMsg := ""
    for serverMsg == ""{
        dprintln("?")
        var err error
        serverMsg, err = ssa.serverInConn.ReadString('\n')
        if err != nil {
            panic(err)
        }
    }
    dprintln(serverMsg)
    escTokens := strings.Split(serverMsg[:len(serverMsg)-1], " ")
    tokens := make([]string, len(escTokens))
    for i, escTok := range escTokens {
        tokens[i], _ = unescape(escTok, 0)
    }
    return tokens[0], tokens[1:]
}
