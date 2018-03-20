package goat

type inProcess struct {
    chnRply *unboundChanInt
    chnData *unboundChanMessage
    chnFirstMid chan int
    chnNext chan struct{}
    nid int
    inMessages map[int]Message
    inMids map[int]struct{}
    
    chnFreshMid *unboundChanInt
    chnMessage *unboundChanMessage
}

func newInProcess(chnRply *unboundChanInt, chnData *unboundChanMessage) *inProcess {
    ip := inProcess {chnRply: chnRply,
        chnData: chnData,
        chnFirstMid: make(chan int),
        chnNext: make(chan struct{}),
        nid: -1,
        inMessages: map[int]Message{},
        inMids: map[int]struct{}{},
        chnFreshMid: newUnboundChanInt(),
        chnMessage: newUnboundChanMessage()}
    go func(){ip.goroutine()}()
    return &ip
}

func (ip *inProcess) goroutine() {
    for{
        select{
            case mid := <- ip.chnRply.Out:
                ip.inMids[mid] = struct{}{}
            
            case msg := <- ip.chnData.Out:
                ip.inMessages[msg.Id] = msg
                
            case ip.nid = <- ip.chnFirstMid:
            
            case <- ip.chnNext:
                dprintln("N!", ip.nid+1)
                delete(ip.inMids, ip.nid)
                delete(ip.inMessages, ip.nid)
                ip.nid++
        }
        
        if msg, has := ip.inMessages[ip.nid]; has {
                delete(ip.inMessages, ip.nid)
            dprintln("Serving <-",ip.nid)
            ip.chnMessage.In <- msg
        } else if _, has = ip.inMids[ip.nid]; has {
                delete(ip.inMids, ip.nid)
            dprintln("Serving ->",ip.nid)
            ip.chnFreshMid.In <- ip.nid
        }
    }
}
