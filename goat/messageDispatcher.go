package goat

type messageDispatcher struct {
    chnMessage *unboundChanMessage
    chnSubscribe chan []*Process
    chnUnsubscribe chan *Process
    chnNext chan struct{}
    chnAcceptMessage chan bool
    attributes *Attributes
}

func newMessageDispatcher(chnMessageIn *unboundChanMessage, chnSubscribe chan []*Process, chnUnsubscribe chan *Process, chnNext chan struct{}, attributes *Attributes)  *messageDispatcher {
    md := messageDispatcher{chnMessage: chnMessageIn,
        chnSubscribe: chnSubscribe,
        chnUnsubscribe: chnUnsubscribe,
        chnNext: chnNext,
        chnAcceptMessage: make(chan bool),
        attributes: attributes}
    go func(){md.goroutine()}()
    return &md
}

func (md *messageDispatcher) goroutine() {
    subscribedProcs := map[*Process]struct{}{}
    
    for {
        select{
            case msg := <- md.chnMessage.Out:
                unsubscribedProcs := map[*Process]struct{}{}
                accepted := false
                i := 1
                for p := range subscribedProcs {
                    //fmt.Println("Serving",msg.Id,"to",i,"/",len(subscribedProcs))
                    i++
                    if _, uns := unsubscribedProcs[p]; !accepted && !uns {
                        withdraw := false
                        for quit := false; !quit; {
                            select{
                            case p.chnMessage <- msg:
                                quit = true
                            case prs := <- md.chnSubscribe:
                                for _, pr := range prs{
                                    subscribedProcs[pr] = struct{}{}
                                }
                            case pr := <- md.chnUnsubscribe:
                                unsubscribedProcs[pr] = struct{}{}
                                withdraw = (p == pr)
                                if withdraw {
                                    quit = true
                                    //md.attributes.rollback()
                                }
                            }
                        }
                        for quit := false; !withdraw && !quit; {
                            select {
                                case accepted = <- md.chnAcceptMessage:
                                    /*if accepted {
                                        md.attributes.commit()
                                    } else {
                                        md.attributes.rollback()
                                    }*/
                                    quit = true
                                case prs := <- md.chnSubscribe:
                                    for _, pr := range prs{
                                        subscribedProcs[pr] = struct{}{}
                                    }
                                case pr := <- md.chnUnsubscribe:
                                    unsubscribedProcs[pr] = struct{}{}
                                    quit = (p == pr)
                                    if quit {
                                        //md.attributes.rollback()
                                    }
                            }
                        }
                    }
                }
                for p := range unsubscribedProcs{
                    delete(subscribedProcs, p)
                }
                for quit := false; !quit;{
                    select{
                    case md.chnNext <- struct{}{}:
                        dprintln("V Serving <-", msg)
                        quit = true
                    case prs := <- md.chnSubscribe:
                        for _, pr := range prs{
                            subscribedProcs[pr] = struct{}{}
                        }
                    case pr := <- md.chnUnsubscribe:
                        delete(subscribedProcs, pr)
                    }
                }
                        
            case prs := <- md.chnSubscribe:
                for _, pr := range prs{
                    subscribedProcs[pr] = struct{}{}
                }
            case pr := <- md.chnUnsubscribe:
                delete(subscribedProcs, pr) 
        }
    }
}