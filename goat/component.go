package goat

import (
	"sync"
)

type messagePredicate struct {
	message   string
	predicate ClosedPredicate
	invalid   bool
}

/*
Component represent a component of the AbC system. It represents the pair
environment (Attributes) anc behaviour (Process). The components that are
subscribed to the same infrastructure interact between each other, according
to AbC semantics.
*/
type Component struct {
	attributes          Attributes
	//ncomm               *netCommunication
	agent               Agent
	subscribedProcesses map[*Process]struct{}

	condAttributeChanged *sync.Cond

	chnMessageToSend     chan messagePredicate
	chnGetAttributes     chan *Attributes
	chnSubscribe         chan *Process
	chnUnsubscribe       chan *Process
	chnUpdateEventToProc chan chan struct{}
	nid                  int
	chnEvtMessageSent    chan int
	chnComponentInbox    chan Message
	chnUpdateEvent       chan struct{}
	chnWaitForMid        chan int
	chnClearToSend       chan struct{}
	chnWantsToSend       chan struct{}
	chnComponentStarts   chan struct{}
}

/*
NewComponent defines a new component that interacts with the infrastructure whose
access point is the server URI. Its environment is the empty set.
*/
func NewComponent(agent Agent) *Component {
	return NewComponentWithAttributes(agent, nil)
}

/*
NewComponentWithAttributes defines a new component that interacts with the infrastructure whose
access point is the server URI. The environment is initialized according to attrInit.
*/
func NewComponentWithAttributes(agent Agent, attrInit map[string]interface{}) *Component {
	c := Component{
		attributes:           Attributes{},
		chnMessageToSend:     make(chan messagePredicate),
		subscribedProcesses:  map[*Process]struct{}{},
		condAttributeChanged: sync.NewCond(&sync.Mutex{}),
		chnGetAttributes:     make(chan *Attributes),
		chnSubscribe:         make(chan *Process),
		chnUnsubscribe:       make(chan *Process),
		chnUpdateEventToProc: make(chan chan struct{}),
		chnComponentInbox:    make(chan Message),
		chnEvtMessageSent:    make(chan int),
		chnUpdateEvent:       make(chan struct{}),
		chnWaitForMid:        make(chan int),
		chnClearToSend:       make(chan struct{}),
		chnWantsToSend:       make(chan struct{}),
		chnComponentStarts:   make(chan struct{}),
		agent: agent,
	}
	if attrInit != nil {
		c.attributes.init(attrInit)
	}
	//c.ncomm = netCommunicationInitAndRun(server)
	//c.agent = NewSingleServerAgent(server)
	c.agent.Start()
	dprintln(c.agent.GetComponentId(),"started")
	//c.nid = c.ncomm.firstMessageId
	c.nid = c.agent.GetFirstMessageId()

	go c.readMessageGoroutine()
	go c.componentGoroutine()
	return &c
}

/*
sendMessage forwards messageToSend (whose message id is mid) to be sent to
netCommunication and signals that the message with message id mid has been processed
by this component: indeed, according to the semantics, no process of this component
can receive it.
*/
func (c *Component) sendMessage(messageToSend messagePredicate, mid int) int {
	//if _, isFP := messageToSend.predicate.(False); isFP{
	//	return -1
	//}
	//mid := c.ncomm.getMessageId()
	var msgWithMid Message
	if messageToSend.invalid {
		msgWithMid = Message{
			Message:   NewTuple(),
			Pred:      False(),
			Id:        mid,
		}
	} else {
		msgWithMid = Message{
			Message:   decodeTuple(messageToSend.message),
			Pred: messageToSend.predicate,
			Id:        mid,
		}
	}
	c.chnEvtMessageSent <- mid
	if _, ok := msgWithMid.Pred.(_false); !ok {
		dprintln("Sending", c.agent.GetComponentId(), "->", msgWithMid.Message, "[", msgWithMid.Id, "]")
	}
	//c.ncomm.chnOutbox <- msgWithMid
	c.agent.Outbox() <- msgWithMid
	return mid
}

//TODO dispatching = mid == c.nid with or without or?
/*
readMessageGoroutine is a goroutine that:
* intercepts the messages coming from the infrastructure and stores them,
* delivers the messages to the processes in order, ensuring that at most one
    process can accept a message,
* keeps track of which messages are sent from this component, so they are skipped
    in the delivery order,
* gives the authorization to send a message from the process that is trying to send
    to the component only if all the messages with message ids lower have been
    already sent or received.
*/
func (c *Component) readMessageGoroutine() {
	msgInbox := map[int]Message{}
	msgOutbox := map[int]bool{}
	midToWait := -1
	componentStarted := false

	for {
		dispatching := false
		for !componentStarted {
			select {
			case msg := <-c.agent.Inbox():
				msgInbox[msg.Id] = msg
				dispatching = msg.Id == c.nid
			case mid := <-c.chnEvtMessageSent:
				msgOutbox[mid] = true
				dispatching = mid == c.nid
			case midToWait = <-c.chnWaitForMid:

			case <-c.chnComponentStarts:
				componentStarted = true
				if c.nid == midToWait {
					close(c.chnClearToSend)
					dprintln(c.agent.GetComponentId(), "CTS", midToWait)
					c.chnClearToSend = make(chan struct{})
					midToWait = -1
				}
			}
		}
		for !dispatching {
			select {
			case msg := <-c.agent.Inbox():
				msgInbox[msg.Id] = msg
				dispatching = msg.Id == c.nid
			case mid := <-c.chnEvtMessageSent:
				msgOutbox[mid] = true
				dispatching = mid == c.nid
			case midToWait = <-c.chnWaitForMid:
				if c.nid == midToWait {
					close(c.chnClearToSend)
					dprintln(c.agent.GetComponentId(), "CTS", midToWait)
					c.chnClearToSend = make(chan struct{})
					midToWait = -1
				}
			}
		}
		for continueDispatching := true; continueDispatching; {
			if msgOutbox[c.nid] {
				//fmt.Println(c.ncomm.componentId,"skipping",c.nid)
				delete(msgOutbox, c.nid)
				//fmt.Println("sent",c.nid)
				c.nid++
				if c.nid == midToWait {
					close(c.chnClearToSend)
					dprintln(c.agent.GetComponentId(), "CTS", midToWait)
					c.chnClearToSend = make(chan struct{})
					midToWait = -1
				}
			} else if inMsg, has := msgInbox[c.nid]; has {
				//fmt.Println(c.ncomm.componentId,"dispatching",c.nid)
				for forwardedMsg := false; !forwardedMsg; {
					select {
					case c.chnComponentInbox <- inMsg:
						//fmt.Println(c.ncomm.componentId,"dispatched",c.nid)
						delete(msgInbox, c.nid)
						//fmt.Println("disp",c.nid)
						forwardedMsg = true
						/*c.nid++
						if c.nid == midToWait {
							close(c.chnClearToSend)
							fmt.Println(c.ncomm.componentId, "CTS", midToWait)
							c.chnClearToSend = make(chan struct{})
							midToWait = -1
						}*/
					case msg := <-c.agent.Inbox():
						msgInbox[msg.Id] = msg
					case mid := <-c.chnEvtMessageSent:
						msgOutbox[mid] = true
					case midToWait = <-c.chnWaitForMid:
						if c.nid == midToWait {
							close(c.chnClearToSend)
							dprintln(c.agent.GetComponentId(), "CTS", midToWait)
							c.chnClearToSend = make(chan struct{})
							midToWait = -1
						}
					}
				}
				for sentMsg := false; !sentMsg; {
					select {
					case msg := <-c.agent.Inbox():
						msgInbox[msg.Id] = msg
					case mid := <-c.chnEvtMessageSent:
						if mid == c.nid {
							c.nid++
							sentMsg = true
							if c.nid == midToWait {
								close(c.chnClearToSend)
								dprintln(c.agent.GetComponentId(), "CTS", midToWait)
								c.chnClearToSend = make(chan struct{})
								midToWait = -1
							}
						} else {
							msgOutbox[mid] = true
						}
					case midToWait = <-c.chnWaitForMid:
						if c.nid == midToWait {
							close(c.chnClearToSend)
							dprintln(c.agent.GetComponentId(), "CTS", midToWait)
							c.chnClearToSend = make(chan struct{})
							midToWait = -1
						}
					}
				}
			} else {
				continueDispatching = false
			}
		}
	}
}

/*
sendMessageToProcesses is called when the component must forward a message to its
processes.
*/
func (c *Component) sendMessageToProcesses(messageToDeliver Message) {
	//fmt.Println(c.ncomm.componentId, "smtp+", messageToDeliver.id)
	recipients := map[*Process]chan struct{}{}
	for p := range c.subscribedProcesses {
		recipients[p] = make(chan struct{})
	}

	for recipient, chnEvtUnsubscribed := range recipients {
		chnRecipientAccepted := make(chan bool)
		go func() {
			//fmt.Println(c.ncomm.componentId,"Forwarding",messageToDeliver.id)
			select {
			case recipient.chnMessage <- attributesInMessage{&c.attributes, messageToDeliver}:
			case <-chnEvtUnsubscribed:
				//fmt.Println(c.ncomm.componentId,"Unsubscribed",messageToDeliver.id)
				chnRecipientAccepted <- false
				return
			}
			//fmt.Println(c.ncomm.componentId,"Forwarded",messageToDeliver.id)
			select {
			case accepted := <-recipient.chnAcceptMessage:
				chnRecipientAccepted <- accepted
			case <-chnEvtUnsubscribed:
				//fmt.Println(c.ncomm.componentId,"A_Unsubscribed",messageToDeliver.id)
				chnRecipientAccepted <- false
			}
			//fmt.Println(c.ncomm.componentId,"Answered",messageToDeliver.id)
		}()
		for reply := false; !reply; {
			select {
			case p := <-c.chnSubscribe:
				c.onSubscribe(p)
				//c.subscribedProcesses[p] = struct{}{}
			case p := <-c.chnUnsubscribe:
				delete(c.subscribedProcesses, p)
				close(recipients[p])
			case rAccepted := <-chnRecipientAccepted:
				reply = true
				if rAccepted {
					anyUpdates := c.attributes.commit()
					c.chnEvtMessageSent <- messageToDeliver.Id
					dprintln(c.attributes)
					dprintln("Accepted", c.agent.GetComponentId(), "<-", messageToDeliver.Message, "[", messageToDeliver.Id, "]")
					if anyUpdates {
						close(c.chnUpdateEvent)
						c.chnUpdateEvent = make(chan struct{})
					}
					//fmt.Println(c.ncomm.componentId, "smtp-", messageToDeliver.id)
					return
				} else {
					c.attributes.rollback()
				}
			}
		}
	}
	c.chnEvtMessageSent <- messageToDeliver.Id
	//fmt.Println(c.ncomm.componentId, "smtp-", messageToDeliver.id)
}

/*
sendMessageFromProcess is called when a process wants to send a message. It
waits for its turn before sending the message (all messages whose id is lower
are sent/received)
*/
func (c *Component) sendMessageFromProcess() {
	//fmt.Println(c.ncomm.componentId, "smfp*")
	cts := c.chnClearToSend
	msgId := c.agent.GetMessageId()
	c.chnWaitForMid <- msgId

	for readyToSend := false; !readyToSend; {
		select {
		case p := <-c.chnSubscribe:
			c.onSubscribe(p)
			//c.subscribedProcesses[p] = struct{}{}
		case p := <-c.chnUnsubscribe:
			delete(c.subscribedProcesses, p)
		case messageToDeliver := <-c.chnComponentInbox:
			c.sendMessageToProcesses(messageToDeliver)
		case <-cts:
			readyToSend = true
			dprintln(c.attributes)
		}
	}
	//fmt.Println(c.ncomm.componentId, "smfp+")
	c.chnGetAttributes <- &c.attributes
	//fmt.Println(c.ncomm.componentId, "smfpR")
	for {
		select {
		case p := <-c.chnSubscribe:
			c.onSubscribe(p)
			//				c.subscribedProcesses[p] = struct{}{}
		case p := <-c.chnUnsubscribe:
			delete(c.subscribedProcesses, p)
		case messageToSend := <-c.chnMessageToSend:
			if !messageToSend.invalid {
				c.sendMessage(messageToSend, msgId)
				/*
					if mid, hmd :=  c.attributes.Get("mid"); hmd {
						fmt.Println("m", mid, "sent", messageToSend.message, msgId)
					}
					if wid, hmd :=  c.attributes.Get("wid"); hmd {
						fmt.Println("w", wid, "sent", messageToSend.message, msgId)
					}*/

				anyUpdates := c.attributes.commit()
				if anyUpdates {
					close(c.chnUpdateEvent)
					c.chnUpdateEvent = make(chan struct{})
				}
			} else {
				_ = c.sendMessage(messageToSend, msgId)
				c.attributes.rollback()
				c.chnUpdateEventToProc <- c.chnUpdateEvent
			}
			//fmt.Println(c.ncomm.componentId, "smfp-")
			return
		}
	}
}

/*
componentGoroutine is the main goroutine that coordinates the processes requests.
It calls the specific methods to handle the different situations.
*/
func (c *Component) componentGoroutine() {
	for {
		select {
		case p := <-c.chnSubscribe:
			c.onSubscribe(p)
			//c.subscribedProcesses[p] = struct{}{}
		case p := <-c.chnUnsubscribe:
			delete(c.subscribedProcesses, p)
		case messageToDeliver := <-c.chnComponentInbox:
			c.sendMessageToProcesses(messageToDeliver)
		//case c.chnGetAttributes <- &c.attributes:
		case <-c.chnWantsToSend:
			c.sendMessageFromProcess()
		}
	}
}

/*
onSubscribe handles the subscription of a new process to this component.
Subscription happens when:
* the component is started with the behaviour of p;
* a process p' is a parallel composition of p and other processes.
*/
func (c *Component) onSubscribe(p *Process) {
	select {
	case <-c.chnComponentStarts:
	default:
		close(c.chnComponentStarts)
	}
	c.subscribedProcesses[p] = struct{}{}
}
