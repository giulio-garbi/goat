package goat

import (
	"time"
)

/*
Process represents the behaviour of the system. It is associated to a Component.
*/
type Process struct {
	Comp *Component

	chnAcceptMessage chan bool
	chnMessage       chan attributesInMessage
}

/*
NewProcess returns the placeholder process of the behaviour of the component c.
Note that it does not define now how c behaves.
*/
func NewProcess(c *Component) *Process {
	p := Process{
		Comp: c,

		chnAcceptMessage: make(chan bool),
		chnMessage:       make(chan attributesInMessage),
	}
	return &p
}

func (p *Process) unsubscribe() {
	//close(p.chnAcceptMessage)
	p.Comp.chnUnsubscribe <- p
}

/*
Run defines that the wrapped component must behave like procFnc, and starts the
component behaviour. Note that each component behaves as only one process (that
it could well be a parallel composition).
*/
func (p *Process) Run(procFnc func(p *Process)) {
	chnSubscribed := make(chan struct{})
	go func() {
		p.Comp.chnSubscribe <- p
		close(chnSubscribed)
		p.Call(procFnc)
		p.unsubscribe()
		//fmt.Println("Unsubscribed")
	}()
	<-chnSubscribed
}

/*
Call makes the process to behave as procFnc.
*/
func (p *Process) Call(procFnc func(p *Process)) {
	procFnc(p)
}

/*
Spawn creates a new process that behaves like procFnc running on the same component.
*/
func (p *Process) Spawn(procFnc func(p *Process)) {
	chnSubscribed := make(chan struct{})
	go func() {
		subProc := NewProcess(p.Comp)
		subProc.Comp.chnSubscribe <- subProc
		close(chnSubscribed)
		subProc.Call(procFnc)
		subProc.unsubscribe()
		//fmt.Println("Unsubscribed")
	}()
	<-chnSubscribed
}

type attributesInMessage struct {
	attribs *Attributes
	inMsg   Message
}

/*
Receive blocks the execution of p until an acceptable message is received.
A message is acceptable if the attributes satisfy the aware condition, and the
message satisfies the accept condition. aware and accept can alter the attributes
(attr), but if the message is not accepted any change to them will be lost.
*/
func (p *Process) Receive(accept func(attr *Attributes, msg Tuple) bool) Tuple {
	return p.sendrec(
		func(attr *Attributes, receiving bool) SendReceive {
			if receiving {
				return ThenReceive(accept)
			} else {
				return ThenFail()
			}
		}, true)
}

type srAction int

const (
	sendAction    srAction = iota
	receiveAction srAction = iota
)

type SendReceive struct {
	action  srAction
	msg     string
	msgPred ClosedPredicate
	valid   bool
	accept  func(*Attributes, Tuple) bool
	updFnc  func(*Attributes)
}

/*
ThenSend signals the intention to sent the message msg to the components that
satisfy the predicate pred.
*/
func ThenSend(msg Tuple, msgPred ClosedPredicate) SendReceive {
	return ThenSendUpdate(msg, msgPred, func(*Attributes){})
}

/*
ThenSendUpdate signals the intention to sent the message msg to the components that
satisfy the predicate pred, and then alters the environment
*/
func ThenSendUpdate(msg Tuple, msgPred ClosedPredicate, updFnc func(*Attributes)) SendReceive {
	return SendReceive{
		action:  sendAction,
		msg:     msg.encode(),
		msgPred: msgPred,
		valid:   true,
		updFnc:  updFnc,
	}
}

/*
ThenFail signals the intention to retry the Send/Receive when a new message arrives
or the attributes of the component change.
*/
func ThenFail() SendReceive {
	return SendReceive{
		action: sendAction,
		valid:  false,
	}
}

/*
ThenReceive signals the intention to accept the first received message that satisfies the
accept condition.
*/
func ThenReceive(accept func(*Attributes, Tuple) bool) SendReceive {
	return SendReceive{
		action: receiveAction,
		accept: accept,
	}
}

/*
Sleep pauses the process p for msec milliseconds. Any message received during
this timeframe is rejected.
*/
func (p *Process) Sleep(msec int) {
	timeout := time.After(time.Duration(msec) * time.Millisecond)
	for {
		select {
		case <-p.chnMessage:
			p.chnAcceptMessage <- false
		case <-timeout:
			return
		}
	}
}

func (p *Process) sendrec(chooseFnc func(attr *Attributes, receiving bool) SendReceive, onlyReceive bool) Tuple {
	chnTryASend := make(chan struct{})
	chnGetAttributes := make(chan *Attributes)
	chnFailTheSend := make(chan struct{})
	if !onlyReceive {
		close(chnTryASend)
	}
	for {
		select {
		case attrsIM := <-p.chnMessage:
			attrs := attrsIM.attribs
			inMsg := attrsIM.inMsg
			nextAction := chooseFnc(attrs, true)
			if nextAction.action == receiveAction &&
				attrs.Satisfy(inMsg.Pred) &&
				nextAction.accept(attrs, inMsg.Message) {
				p.chnAcceptMessage <- true
				close(chnFailTheSend)
				return inMsg.Message
			} else {
				p.chnAcceptMessage <- false
			}

		case <-chnTryASend:
			chnTryASend = make(chan struct{})
			go func(ga chan *Attributes, cmp *Component) {
				cmp.chnWantsToSend <- struct{}{}
				at := <-cmp.chnGetAttributes
				select {
				case ga <- at:
				case <-chnFailTheSend:
					{
						p.Comp.chnMessageToSend <- messagePredicate{invalid: true}
						<-p.Comp.chnUpdateEventToProc
					}
				}
			}(chnGetAttributes, p.Comp)

		case attrs := <-chnGetAttributes:
			if onlyReceive {
				p.Comp.chnMessageToSend <- messagePredicate{invalid: true}
				<-p.Comp.chnUpdateEventToProc
			} else {
				nextAction := chooseFnc(attrs, false)
				if nextAction.action == sendAction {
					msg := nextAction.msg
					msgPred := nextAction.msgPred
					valid := nextAction.valid
					if valid {
						p.Comp.chnMessageToSend <- messagePredicate{msg, msgPred, false}
						return NewTuple()
					}
				}
				p.Comp.chnMessageToSend <- messagePredicate{invalid: true}
				chnTryASend = <-p.Comp.chnUpdateEventToProc
			}
		}
	}
}

/*
SendOrReceive allows to implement either a Send or a Receive according to the value
of the attributes. The condition is encoded into chooseFnc. chooseFnc must return
a call to ThenSend or ThenFail if receiving is false, or a call to ThenReceive or
ThenFail otherwise. chooseFnc is allowed to modify the attributes, but any change
is lost if a message is not actually received or sent.
Deprecated: this is a low level API call that can be avoided. It should be used only 
by the code generator.
*/
func (p *Process) SendOrReceive(chooseFnc func(attr *Attributes, receiving bool) SendReceive) {
	p.sendrec(chooseFnc, false)
}

/*
Send sends a message to other components. msgFnc defines which message will be
sent, according to the return values:
* if a message must be sent (according to the attributes), msgFnc must return
	- the message
	- the predicate that a component must satisfy in order to be able to read it
	- the true value
* otherwise, msgFnc must return a string, a predicate and the false value.
Note that msgFnc can alter the attributes, but if the message is not sent any
change to them will be lost.
Deprecated: this is a low level API call that can be avoided. It should be used only 
by the code generator.
*/
func (p *Process) SendFunc(msgFnc func(attr *Attributes) (Tuple, Predicate, bool)) {
	p.sendrec(func(attr *Attributes, receiving bool) SendReceive {
		if receiving {
			return ThenFail()
		}
		msg, msgPred, valid := msgFnc(attr)
		if valid {
			return ThenSend(msg, msgPred.CloseUnder(attr))
		} else {
			return ThenFail()
		}
	}, false)
}

/*
Send sends a message to other components. msg contains the message to be sent,
pr states the property a component must satisfy to receive msg.
*/
func (p *Process) Send(msg Tuple, pr Predicate){
    p.SendUpd(msg, pr, func(*Attributes){})
}

/*
SendUpd sends a message to other components. msg contains the message to be sent,
pr states the property a component must satisfy to receive msg. After sending the
message, the upd function can alter the set of attributes.
*/
func (p *Process) SendUpd(msg Tuple, pr Predicate, upd func(*Attributes)){
    p.sendrec(func(attr *Attributes, receiving bool) SendReceive {
		if receiving {
			return ThenFail()
		} else {
		    cmsg := msg.CloseUnder(attr)
		    cpr := pr.CloseUnder(attr)
		    upd(attr)
		    return ThenSend(cmsg, cpr)
		}
	}, false)
}

type selectcase struct{
    pred Predicate
    action SendReceive
    then func(*Process)
}

func Nothing() {}

/*
ZeroProcess represents a process that cannot do anything (hence it behaves as 0).
*/
func ZeroProcess(*Process) {}

/*
Case represent a case in a Select call. pred is the predicate to be satisfied to
enter the case. action is the action to perform in this case (send/receive/set/fail).
If the action fails, the Select will be retried; otherwise the process continues as
then.
*/
func Case(pred Predicate, action SendReceive, then func(*Process)) selectcase{
    return selectcase{pred, action, then}
}

/*
Select is a statement that allows the process to evolve differently according to
the environment. Each possible evolution is stated in a case. Cases are considered
sequentially in the order they are given. If none of the case is satisfied, the Select
statement is repeated as soon as the environment changes.
*/
func (p *Process) Select(cases ...selectcase){
    var caseN int
    p.sendrec(func(attr *Attributes, receiving bool) SendReceive {
        for i, casei := range cases{
            if casei.pred.CloseUnder(attr).Satisfy(attr){
                wantsToReceive := casei.action.action == sendAction
		        if receiving != wantsToReceive {
			        return ThenFail()
		        } else {
		            caseN = i
		            return casei.action
		        }
		    }
	    }
	    return ThenFail()
	}, false)
	p.Call(cases[caseN].then)
}

/*
NoPre generates an always true precondition.
*/
/*func NoPre() func(attr *Attributes) bool {
	return func(*Attributes) bool {
		return true
	}
}*/

/*
WaitUntilTrue blocks p until the todo condition is true. Any message received
in the meantime is rejected.
*/
func (p *Process) WaitUntilTrue(todo func(attr *Attributes) bool) {
	p.SendFunc(func(attr *Attributes) (Tuple, Predicate, bool){
	    return NewTuple(), False(), todo(attr)
	})
}

/*
SetIf waits for pred to be satisfied, then changes the environment according
to setup. Any message received during this call is rejected.
*/
func (p *Process) SetIf(pred Predicate, setup func(attr *Attributes)) {
	p.SendFunc(func(attr *Attributes) (Tuple, Predicate, bool){
	    if pred.CloseUnder(attr).Satisfy(attr) {
	        setup(attr)
	        return NewTuple(), False(), true
	    } else {
	        return NewTuple(), False(), false
        }
	})
}

/*
Set changes the environment according to setup. Any message received during 
this call is rejected.
*/
func (p *Process) Set(setup func(attr *Attributes)) {
	p.SetIf(True(), setup)
}
