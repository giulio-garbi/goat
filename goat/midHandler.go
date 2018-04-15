package goat

type midHandler struct {
    chnFreshMid *unboundChanInt
    chnMsgFromProc chan messagePredicate
    chnRetry chan struct{}
    chnNewStop chan chan struct{}
    chnNewSend chan chan struct{}
    chnTimeToAskMid chan struct{}
    askMidPolicy askMidPol
    agent Agent
    attributes *Attributes
    chnNext chan struct{}
    evtMid int
    chnEvtMid chan struct{}
}

type askMidPol int

const (
    ampNone askMidPol = iota
    ampOnUpdate askMidPol = iota
    ampUnconditional askMidPol = iota
)

func NewMidHandler(chnFreshMid *unboundChanInt, agent Agent, attributes *Attributes, chnNext chan struct{}) *midHandler{
    mh := midHandler{ chnFreshMid: chnFreshMid,
        chnMsgFromProc: make(chan messagePredicate),
        chnRetry: make(chan struct{}),
        chnNewStop: make(chan chan struct{}),
        chnNewSend: make(chan chan struct{}),
        chnTimeToAskMid: make(chan struct{}),
        askMidPolicy: ampNone,
        agent: agent,
        attributes: attributes,
        chnNext: chnNext,
        evtMid: -1}
    go func(){mh.start()}()
    return &mh
}

func (mh *midHandler) OnMid(mid int, chnEvt chan struct{}) {
    mh.chnEvtMid = chnEvt
    mh.evtMid = mid
}

func (mh *midHandler) StopMids(incomingMids chan struct{}) {
    mh.chnNewStop <- incomingMids
}
func (mh *midHandler) SendMessage(msg messagePredicate, incomingMids chan struct{}){
    mh.chnMsgFromProc <- msg
}
func (mh *midHandler) AskMids(incomingMids chan struct{}) {
    mh.chnNewSend <- incomingMids
}
func (mh *midHandler) RetryLater(incomingMids chan struct{}) {
    mh.chnRetry <- struct{}{}
}

func (mh *midHandler) start() {
    sendingChans := map[chan struct{}]struct{}{}
    mh.chnTimeToAskMid = make(chan struct{})
    mh.askMidPolicy = ampNone
    for{
        select {
            case <- mh.chnTimeToAskMid:
                dprintln("askmid")
                mh.chnTimeToAskMid = make(chan struct{})
                mh.askMidPolicy = ampNone
                mh.agent.AskMid()
                
            case mid := <- mh.chnFreshMid.Out:
                dprintln("Prepare a send", mid)
                stoppedChans := map[chan struct{}]struct{}{}
                toBeAddedChans := map[chan struct{}]struct{}{}
                midConsumed := false
                messageToSend := messagePredicate{invalid: true}
                for chn := range sendingChans {
                    if _,has := stoppedChans[chn]; !midConsumed && !has{
                        withdraw := false
                        for quit:= false;!quit;{
                            select {
                            case chn <- struct{}{}:
                                quit = true
                            case csnd := <- mh.chnNewSend:
                                toBeAddedChans[csnd] = struct{}{}
                            case cstop := <- mh.chnNewStop:
                                if cstop == chn {
                                    quit = true
                                    withdraw = true
                                }
                                stoppedChans[cstop] = struct{}{}
                            }
                        }
                        for quit:=false;!withdraw && !quit; {
                            select{
                            case cstop := <- mh.chnNewStop:
                                if cstop == chn {
                                    withdraw = true
                                    mh.attributes.rollback()
                                    quit = true
                                }
                                stoppedChans[cstop] = struct{}{}
                            
                            case messageToSend = <-mh.chnMsgFromProc:
                                midConsumed = true
                                mh.attributes.commit()
                                stoppedChans[chn] = struct{}{}
                                quit = true
                            
                            case csnd := <- mh.chnNewSend:
                                toBeAddedChans[csnd] = struct{}{}
                                
                            case <-mh.chnRetry:
                                quit = true
                                mh.attributes.rollback()
                            }
                        }
                    }
                }
                
                mh.agent.SendMessage(makeMessage(messageToSend, mid))
                if mh.evtMid == mid {
                    close(mh.chnEvtMid)
                }
                dprintln("X Serving ->", mid)
                
                hasFreshChans := false
                for chn := range toBeAddedChans {
                    if _, has := stoppedChans[chn]; !has {
                        hasFreshChans = true
                        sendingChans[chn] = struct{}{}
                    }
                }
                for chn := range stoppedChans {
                    delete(sendingChans, chn)
                }
                dprintln("Y Serving ->", mid)
                
                if hasFreshChans || (midConsumed && len(sendingChans) > 0){
                    mh.chnTimeToAskMid = make(chan struct{})
                    mh.askMidPolicy = ampUnconditional
                    close(mh.chnTimeToAskMid)
                } else if len(sendingChans) > 0 {
                    mh.chnTimeToAskMid = mh.attributes.onUpdate.Get()
                    mh.askMidPolicy = ampOnUpdate
                }
                dprintln("V Serving ->", mid)
                mh.chnNext <- struct{}{}
                dprintln("VN Serving ->", mid)
                
            case cstop := <- mh.chnNewStop:
                delete(sendingChans, cstop)
            
            case csnd := <- mh.chnNewSend:
                sendingChans[csnd] = struct{}{}
                //if len(sendingChans) == 1 {
                    if mh.askMidPolicy != ampUnconditional{
                        mh.chnTimeToAskMid = make(chan struct{})
                        mh.askMidPolicy = ampUnconditional
                        close(mh.chnTimeToAskMid)
                    }
                //} 
        }
    }
}

