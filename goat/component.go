package goat

type Component struct {
    agent Agent
    midHandler *midHandler
    attributes *Attributes
    messageDispatcher *messageDispatcher
    inProcess *inProcess
    chnSubscribe chan []*Process
    chnUnsubscribe chan *Process
}

/*
NewComponentWithAttributes defines a new component that interacts with the infrastructure whose
access point is the server URI. The environment is initialized according to attrInit.
*/
func NewComponentWithAttributes(agent Agent, attrInit map[string]interface{}) *Component {
    chnSubscribe := make(chan []*Process)
    chnUnsubscribe := make(chan *Process)
    attributes := NewAttributes()
    inProcess := newInProcess(agent.GetRplyChan(), agent.GetDataChan())
    midHandler := NewMidHandler(inProcess.chnFreshMid, agent, attributes, inProcess.chnNext)
    messageDispatcher := newMessageDispatcher(inProcess.chnMessage, chnSubscribe, chnUnsubscribe, inProcess.chnNext, attributes)
    
	c := Component{
		attributes: attributes,
		agent: agent,
		midHandler: midHandler,
		messageDispatcher: messageDispatcher,
        inProcess: inProcess,
        chnSubscribe: chnSubscribe,
        chnUnsubscribe: chnUnsubscribe,
	}
	if attrInit != nil {
		c.attributes.init(attrInit)
	}
	//c.ncomm = netCommunicationInitAndRun(server)
	//c.agent = NewSingleServerAgent(server)
	c.agent.Start()
	dprintln(c.agent.GetComponentId(),"started")
	//c.nid = c.ncomm.firstMessageId
	fMid := c.agent.GetFirstMessageId()
	inProcess.chnFirstMid <- fMid
	dprintln(c.agent.GetComponentId(),"'s first mid is",fMid)

	return &c
}

func NewComponent(agent Agent, attrInit map[string]interface{}) *Component {
    return NewComponentWithAttributes(agent, attrInit)
}

func (c *Component) Start(procFncs ...func(p *Process)) {
    NewProcess(c).Run(procFncs...)
}

func (c *Component) OnMid(mid int) chan struct{} {
    chnEvt := make(chan struct{})
    c.midHandler.OnMid(mid, chnEvt)
    c.messageDispatcher.OnMid(mid, chnEvt)
    return chnEvt
}

func (c *Component) GetAgent() Agent {
    return c.agent
}
