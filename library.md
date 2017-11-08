# Programming with the GoAt Go library
Using the library, you can write Go programs that take advantage of the attribute-based communication paradigm. 

## How to define a component
The skeleton of a component definition follows:

    package main

    import (
        "github.com/goat-pakage/goat/goat"
    )

    func main(){
        environment := map[string]interface{}{...}
        agent := ...
        comp := goat.NewComponentWithAttributes(agent, environment)
        goat.NewProcess(comp).Run(func(p *goat.Process) {
            ...
	    })
    }

`environment` is a map that contains the set of attributes (and corresponding values) that the component has at startup. Attribute names must be strings, while the associated values can be of any type. Note that associating an attribute to `nil` is not the same as not having that attribute associated to anything. `agent` acts as an adapter between the infrastructure and the component. It can be seen as the access point to the infrastructure from the component. It's constructor differs according to the infrastructure that the component participates to:

Infrastructure | Agent constructor |
---|---|
Single Server | `goat.NewSingleServerAgent(serverAddressAndPort)` |
Cluster | `goat.NewClusterAgent(messageQueueAddressAndPort, registrationAddressAndPort)` |
Ring | `goat.NewRingAgent(registrationAddressAndPort)` |
Tree | `goat.NewTreeAgent(registrationAddressAndPort)` |

`comp` is the component. It contains both the dynamic behaviour (the process) and its state (the set of attributes, called environment). You can create a component by calling:
* `goat.NewComponentWithAttributes(agent, environment)`, to create a new component linked to the infrastructure via the agent `agent`; its environment is set according to the map `environment`;
* `goat.NewComponent`, to create a new component linked to the infrastructure via the agent `agent`; its environment empty.

Note that an agent must be associated with at most one component: that is, for each component you have to instantiate one different agent. 

` goat.NewProcess(comp).Run(func(p *goat.Process) {...})` sets up the behaviour of the component and starts it. The ellipsis must be replaced by the behaviour you want that component to have. `p` represents the `goat.Process` object, that is the access point to the functions that allows the specification of the behaviour.
#### How to define a process with the `goat.Process` API?
The `goat.Process` API was designed to match as much as possible the AbC constructs. In the following, we assume that `proc` is an object of type `goat.Process`.

##### Example: a classroom
To show the primitives in action, we will model step by step a classroom. In a classroom there are two types of agents: teachers and students.

Each teacher holds a lesson of one subject. At the same time, teachers answer questions posed by students.

Each student listens to a lesson. While listening to a lesson, it might ask questions to the corresponding teacher, and it listens for the answer. Each student can also choose to listen to another subject.

##### Send a message
`proc.Send(message, predicate)` sends a message to all components (but this one) that satisfy the predicate `predicate`. A message is a tuple created with the `goat.NewTuple(parts...)` method. In a tuple it is possible to put immediate values or the value associated to an attribute. To put the value of an attribute, it is possible to use `goat.Comp(attribute_name)`.

For example, a message containing "hello", the id of the component and the number 5, it can be created with `goat.NewTuple("hello", goat.Comp("id"), 5)`.

> **Note:** the environment must contain the stated attributes. Otherwise, the system will crash.

The predicate is a boolean expression that can refer to the attributes of both the sending and receiving components. To express a predicate, it is possible to use the following methods (where p1, p2, ... are predicates and v1, v2 values):
* `goat.And(p1, p2, ...)` that is satisfied iff each `pi` is satisfied;
* `goat.Or(p1, p2, ...)` that is satisfied iff at least for an i, `pi` is satisfied;
* `goat.Not(p1)` that is satisfied iff `p1` is not satisfied;
* `goat.Equals(v1, v2)` that is satisfied iff `v1` evaluation and `v2` evaluation are equal;
* `goat.NotEquals(v1, v2)` that is satisfied iff `v1` evaluation and `v2` evaluation aren't equal;
* `goat.GreaterThan(v1, v2)` that is satisfied iff `v1` evaluation is greater than `v2` evaluation;
* `goat.GreaterThanOrEqual(v1, v2)` that is satisfied iff `v1` evaluation is greater than or equal to `v2` evaluation;
* `goat.LessThan(v1, v2)` that is satisfied iff `v1` evaluation is less than `v2` evaluation;
* `goat.LessThanOrEqual(v1, v2)` that is satisfied iff `v1` evaluation is less than or equal to `v2` evaluation;
* `goat.True()` that is always satisfied;
* `goat.False()` that is never satisfied.

Values can be expressed as:
* an immediate value, that evaluates to the same value;
* `goat.Comp(attr_name)` that evaluates to the value associated to the attribute `attr_name` in the sending component; if `attr_name` is not set, the system will crash;
* `goat.Receiver(attr_name)` that evaluates to the value associated to the attribute `attr_name` in the receiving component; if `attr_name` is not set, the (direct) parent predicate is not satisfied.

For example, to send the message to each component whose age is less than 3 and has the same name as the sender component: `goat.And(goat.LessThan(goat.Receiver("age"), 3), goat.Equals(goat.Comp("name"), goat.Receiver("name")))`.

To wrap up, suppose that you want to send your name to everybody in your group. To do this, you call:
```go
proc.Send(goat.NewTuple("hello", goat.Comp("name")), goat.Equals(goat.Comp("group"), goat.Receiver("group")))
```

Instead, if you want to send your name to everybody:
```go
proc.Send(goat.NewTuple("hello", goat.Comp("name")), goat.True())
```

> **Example:** We model a teacher that only holds a lesson. Its process sends continuously a part of the lesson to its students.
> 
```go
listeningToMe := goat.Equals(goat.Receiver("listening"), goat.Comp("subject"))
for i:=0; ; i++{
    proc.Send(goat.NewTuple("lesson", lessonPart(subject, i)), listeningToMe)
}
```
> Note that the predicate (`listeningToMe`) does not name the students. Rather, it references any receiver listening to the teacher's subject. This allows a great level of flexibility, as the set of listeners might vary over time seamlessly.

##### Send a message and update the environment
`proc.SendUpd(message, predicate, updFnc)` is a function that updates the environment after sending a message. The role of the `message` and `predicate` is the same as in `proc.Send`. `updFnc` is a function that gets a reference to the environment and alters it accordingly. The type of `updFnc` is `func(attr *goat.Attributes)`. `attr` supports four methods:
* `value, has := attr.Get(attr_name)` returns:
  - if the attribute `attr_name` is set in the environment, the value associated with the attribute and `true`;
  - otherwise, `nil` and `false`.
* `attr.GetValue(attr_name)` returns the value associated to the attribute `attr_name` if set, otherwise crashes the system.
* `attr.Has(attr_name)` returns `true` iff the attribute `attr_name` if set.
* `attr.Set(attr_name, new_val)` replaces (or creates) the value associated with the attribute `attr_name` with `new_val`.

Remember that the environment is not typed, hence the value returned by `Get` and `GetValue` has the type `interface{}`. To use it, you must do a type cast.

> **Example:** We model a teacher that sends an answer to a question. Then, it decreases the number of questons pending.
```go
answer := fmt.Sprint("Answer to "+question)
isAsker := goat.Equals(goat.Receiver("id"), asker)
answered := func(attr *goat.Attributes){
    attr.Set("questions", attr.GetValue("questions").(int) - 1)
}
proc.SendUpd(goat.NewTuple("answer", answer), isAsker, answered)
```

##### Receive a message
`proc.Receive(acceptFnc)` blocks `proc` until a message is received and accepted. The call returns the message accepted. `acceptFnc` is of type `func(*goat.Attributes, goat.Tuple) bool`, and its return value states whether the message is accepted.

When a component receives a message from some _other_ component, it tests if its predicate is satisfied by the environment. If the test fails, the message is discarded. Otherwise, the component will consider each process that called `Receive` in some (unspecified) order. If for one of those process (say `p`) `acceptFnc` applied on the environment and the message returns `true`, `p` accepts the message and the component stops considering the message. If no process accepts a message or no processes are willing to receive a message, the message is discarded.

The message (of type `goat.Tuple`) can be tested with this API:
* `msg.IsLong(n)` returns if the message has exactly `n` fields;
* `msg.Get(i)` returns the field of index `i` in the message, or crashes if `i` is bigger than or equal the lenght of the message or negative.

For example, suppose that you want to sum a set of numbers until you get a "stop". A possible solution is:

```go
stop := false
accumOrStop := func(attr *goat.Attributes, msg goat.Tuple) bool{
    if msg.IsLong(2) && msg.Get(0) == "number" {
        num := msg.Get(1).(int)
        acc := attr.GetValue("accumulator").(int)
        attr.Set("accumulator", acc + num)
        return true
    }
    if msg.IsLong(1) && msg.Get(0) == "stop" {
        stop = true
        return true
    }
    return false
}
for !stop {
    proc.Receive(accumOrStop)
}
```

> **Note 1:** it is not possible to send a message from one process to another in the same component.

> **Note 2:** at most one process can accept one message in any component. When a message is accepted it is not considered anymore, so some process might not even preceive that a message is received.

> **Note 3:** the order of the processes used when testing the message is implementation dependent.

> **Note 4:** any modifications made to the environment are lost if `acceptFnc` does not accept the message.

> **Example:** we model the teacher that accepts questions and answers them. When it perceives a question, it spawns another process to answer it. It also keeps track of the number of pending questions. 

```go
func listenQuestions(proc *goat.Process){
	for{
		question := proc.Receive(func(attr *goat.Attributes, msg goat.Tuple) bool {
			if msg.IsLong(3) && msg.Get(0) == "question" {
				attr.Set("questions", attr.GetValue("questions").(int) + 1)
				return true
			}
			return false
		})
		questionTxt := question.Get(1).(string)
		asker := question.Get(2).(int)
		proc.Spawn(answerQuestion(questionTxt, asker))
	}
}

func answerQuestion(question string, asker int) func(proc *goat.Process) {
	return func(proc *goat.Process){
		answer := fmt.Sprint("Answer to "+question)
		isAsker := goat.Equals(goat.Receiver("id"), asker)
		proc.SendUpd(goat.NewTuple("answer", answer), isAsker, 
		    func(attr *goat.Attributes){
		        attr.Set("questions", attr.GetValue("questions").(int) - 1)
	        })
	}
}
```


> **Note 5:** this is a _wrong_ solution for the example:
```go
func listenQuestions(proc *goat.Process){
	for{
		question := proc.Receive(func(attr *goat.Attributes, msg goat.Tuple) bool {
			if msg.IsLong(3) && msg.Get(0) == "question" {
				attr.Set("questions", attr.GetValue("questions").(int) + 1)
				return true
			}
			return false
		})
		questionTxt := question.Get(1).(string)
		asker := question.Get(2).(int)
		answer := fmt.Sprint("Answer to "+questionTxt)
		isAsker := goat.Equals(goat.Receiver("id"), asker)
		proc.SendUpd(goat.NewTuple("answer", answer), isAsker, 
		    func(attr *goat.Attributes){
		        attr.Set("questions", attr.GetValue("questions").(int) - 1)
	        })
	}
}
```
> Suppose that the component receives two questions, q1 and q2. The `Receive` accepts q1, then the process builds the answer and then sends it to the asker via `SendUpd`. `SendUpd` (according to the AbC semantics) rejects all pending messages, hence q2 is discarded and will never be answered. The right solution creates a new process that sends the answer and the main one can accept all the pending questions.

##### Spawn a process
`proc.Spawn(procFnc)` creates a new process on the same component of `proc` that behaves as `procFnc`. `procFnc` is of type `func(p *goat.Process)`.

##### Call a process
`proc.Call(procFnc)` makes `proc`behave as `procFnc`. `procFnc` is of type `func(p *goat.Process)`.

##### Wait until
`proc.WaitUntilTrue(cond)` blocks `proc` until `cond` does return `true`. `cond` is of type `func(attr *goat.Attributes) bool` and implements a condition. `proc` will discard all messages received while executing `WaitUntilTrue`.

> **Example:** we model the process that sends lesson parts when there are no questions pending.
```go
func holdLesson(subject string) func (*goat.Process){
    return func (proc *goat.Process){
	    listeningToMe := goat.Equals(goat.Receiver("listening"), goat.Comp("subject"))
	    for i:=0; ; i++{
		    proc.WaitUntilTrue(func(attr *goat.Attributes) bool {
			    return attr.GetValue("questions") == 0
		    })
		    proc.Send(goat.NewTuple("lesson", lessonPart(subject, i)), listeningToMe)
	    }
	}
}
```
> The call to `WaitUntilTrue` completes only when the attribute `questions` is set to 0. This process, in parallel with the one that listens for questions, sends the lesson part only when there are no questions pending. Indeed, the process that listens for questions increases `questions` on arrival. The process that sends the answer decreases `questions` after sending.

##### Select ... case
`proc.Select(cases...)` blocks until one case is completed. This method allows to perform a send or a receive according to the environment. It is analoguous to the `switch ... case` construct.

Each case is expressed with a call to `goat.Case(pred, action, then)`. `pred` is a predicate over the environment. `action` is one of the following calls:
* `goat.ThenSend(message, predicate)` that sends a message (see the `proc.Send` description for more details);
* `goat.ThenSendUpdate(message, predicate, updFnc)` that sends a message and alters the environment (see the `proc.SendUpd` description for more details);
* `goat.ThenReceive(acceptFnc)` that receives a message (see the `proc.Receive` description for more details); if the message is rejected, the whole `Select` is retried;
* `goat.ThenFail()` that fails the call and retries the whole `Select`.

`then` is a function without parameters that is executed if the case has success.

#### Modelling the classroom example
Now we see the classroom example in full. We describe briefly each part. [Here](classroom.go) it is available the full code.

##### Student
A Student is initialised with a call to `createStudent`:

```go
func createStudent(id int, subject string) *Student{
	environment := map[string]interface{}{
		"listening": subject,
		"id": id,
	}
	agent := goat.NewSingleServerAgent("127.0.0.1:17000")
	return &Student{id, goat.NewComponentWithAttributes(agent, environment)}
}
```
`id` is the unique identifier of a student. Each student must have a different `id`. `subject` is the subject that the student will attend at the beginning.

A student is run with a call to `start`:

```go
func (s *Student) start(){
	goat.NewProcess(s.comp).Run(func(proc *goat.Process){
		proc.Spawn(listen)
		proc.Spawn(changeSubject)
		proc.Call(askQuestions)
	})
}
```
Three processes are run in parallel on each student's component: `listen`, `changeSubject` and `askQuestions`.

`listen` listens to a lesson part that is relevant for it.

```go
func listen(proc *goat.Process){
	for{
		lessonPart := proc.Receive(func(attr *goat.Attributes, msg goat.Tuple) bool {
			return msg.IsLong(2) && msg.Get(0) == "lesson"
		})
		fmt.Println("New lesson part: ", lessonPart.Get(1))
	}
}
```

`changeSubject` changes the subject that the student is attending at random time intervals (whose average is 10 seconds).

```go
func changeSubject(proc *goat.Process){
	for {
	    // Time to change subject
	    proc.Sleep(int(rand.ExpFloat64() * 10000))
	    var newSubject string
	    switch(rand.Intn(3)){
	        case 0:
	            newSubject = "chemistry"
                case 1:
                    newSubject = "physics"
                case 2:
                    newSubject = "math"
	    }
	    proc.WaitUntilTrue(func(attr *goat.Attributes) bool{
	        attr.Set("listening", newSubject)
	        return true
	    })
	}
}
```

`askQuestions` generates questions for the lesson. It asks them to the relevant teacher. The `Sleep` role is only to simulate the time to think a question. After asking a question, the student waits for the answer. A student can only ask a question at a time.

```go
func askQuestions(proc *goat.Process){
	for {
	    // Time to generate a question
	    proc.Sleep(int(rand.ExpFloat64() * 5000))
	    question := "question"
		myTeacher := goat.Equals(goat.Receiver("subject"), goat.Comp("listening"))
		proc.Send(goat.NewTuple("question", question, goat.Comp("id")), myTeacher)
		myId := 0
		answer := proc.Receive(func(attr *goat.Attributes, msg goat.Tuple) bool {
		    myId = attr.GetValue("id").(int)
		    return msg.IsLong(2) && msg.Get(0) == "answer"
		})
		fmt.Printf("%d I asked: question and got: %s\n", myId, answer.Get(1).(string))
	}
}
```

##### Teacher
A teacher is initialised with a call to `createTeacher`. We assume that a subject cannot be taught by mote than one teacher.

```go
func createTeacher(subject string) *Teacher{
	environment := map[string]interface{}{
		"subject": subject,
		"questions": 0,
	}
	agent := goat.NewSingleServerAgent("127.0.0.1:17000")
	return &Teacher{goat.NewComponentWithAttributes(agent, environment), subject}
}
```

A teacher is run with a call to `start`. A teacher holds a lesson while listening for questions.

```go
func (t *Teacher) start(){
	goat.NewProcess(t.comp).Run(func(proc *goat.Process){
		proc.Spawn(holdLesson(t.subject))
		proc.Call(listenQuestions)
	})
}
```

The teacher continues its lesson as long as there are no unanswered questions. The lesson parts are directed to the attendees by the predicate. 

```go
func holdLesson(subject string) func (*goat.Process){
    return func (proc *goat.Process){
	    listeningToMe := goat.Equals(goat.Receiver("listening"), goat.Comp("subject"))
	    for i:=0; ; i++{
		    proc.WaitUntilTrue(func(attr *goat.Attributes) bool {
			    return attr.GetValue("questions") == 0
		    })
		    proc.Send(goat.NewTuple("lesson", lessonPart(subject, i)), listeningToMe)
	    }
	}
}
```
Note that a student will receive the relevant lesson parts without communicating which subject is attending. This means also that the set of sudents can freely change over time. Students and teachers do not know each other.

The listening process follows:

```go
func listenQuestions(proc *goat.Process){
	for{
		question := proc.Receive(func(attr *goat.Attributes, msg goat.Tuple) bool {
			if msg.IsLong(3) && msg.Get(0) == "question" {
				attr.Set("questions", attr.GetValue("questions").(int) + 1)
				return true
			}
			return false
		})
		questionTxt := question.Get(1).(string)
		asker := question.Get(2).(int)
		proc.Spawn(answerQuestion(questionTxt, asker))
	}
}

func answerQuestion(question string, asker int) func(proc *goat.Process) {
	return func(proc *goat.Process){
		answer := fmt.Sprint("Answer to "+question)
		isAsker := goat.Equals(goat.Receiver("id"), asker)
		proc.SendUpd(goat.NewTuple("answer", answer), isAsker, 
		    func(attr *goat.Attributes){
		        attr.Set("questions", attr.GetValue("questions").(int) - 1)
	        })
	}
}
```


## How to instantiate an infrastructure
Since the infrastructures presented here are distributed, you need to create one program for each node type. Before running the components, you need to make sure that the infrastructure is up and running.

### Single Server
It can be instantiated with:

    package main

    import (
        "github.com/goat-pakage/goat/goat"
    )

    func main(){
        port := 17000
        var timeoutMsec int64 = 15000
        term := make(chan struct{})
        goat.RunCentralServer(port, term , timeoutMsec)
        <-term
    }
    
Components can connect to the infrastructure by calling `goat.NewSingleServerAgent("<serverAddress>:<port>")` with the address of the central node and the listening port provided (here 17000).

### Cluster infrastructure
This infrastructure has:
* a node that handles the registration procedure;
* a node that handles the message queue;
* a node that provides fresh message ids upon request;
* a set of serving nodes.

The following code is used to instantiate the registration node

	package main

	import (
	    "github.com/goat-pakage/goat/goat"
	)

	func main(){
	    port := 17000
	    nodesAddresses := []string{} // list of all the serving nodes in the cluster
	    chnTimeout := make(chan struct{})
	    go goat.NewClusterAgentRegistration(port, "<messageQueueAddress>:<mqPort>", nodesAddresses).Work(0, chnTimeout)
	    <-chnTimeout
	}

The following code instantiates the message queue

	package main

	import (
	    "github.com/goat-pakage/goat/goat"
	)

	func main(){
	    port := 17001
	    chnTimeout := make(chan struct{})
	    go goat.NewClusterMessageQueue(port).Work(0, chnTimeout)
	    <-chnTimeout
	}
	
The following code instantiates the provider of fresh message ids

	package main

	import (
	    "github.com/goat-pakage/goat/goat"
	)

	func main(){
	    port := 17002
	    chnTimeout := make(chan struct{})
	    go goat.NewClusterCounter(port).Work(0, chnTimeout)
	    <-chnTimeout
	}

The following code instantiates a serving node

	package main

	import (
	    "github.com/goat-pakage/goat/goat"
	)

	func main(){
	    port := 17003
	    chnTimeout := make(chan struct{})
	    messageQueueAddress := "..."
	    freshMidAddress := "..."
	    registrationAddress := "..."
	    go goat.NewClusterNode(port, messageQueueAddress, freshMidAddress, registrationAddress).Work(0, chnTimeout)
	    <-chnTimeout
	}
	
Components can connect to the infrastructure by calling `goat.NewClusterAgent("<messageQueueAddress>:<port>", "<registrationAddress>:<port>")` with the address of the message queue node and the registration node with the ports provided (here 17001 and 17000).
	
### Ring infrastructure
This infrastructure has:
* a node that handles the registration procedure;
* a node that provides fresh message ids upon request;
* a set of serving nodes.

The following code is used to instantiate the registration node

	package main

	import (
	    "github.com/goat-pakage/goat/goat"
	)

	func main(){
	    port := 17000
	    nodesAddresses := []string{} // list of all the serving nodes in the cluster
	    chnTimeout := make(chan struct{})
	    go goat.NewRingAgentRegistration(port, nodesAddresses).Work(0, chnTimeout)
	    <-chnTimeout
	}

The following code instantiates the provider of fresh message ids

	package main

	import (
	    "github.com/goat-pakage/goat/goat"
	)

	func main(){
	    port := 17001
	    chnTimeout := make(chan struct{})
	    go goat.NewClusterCounter(port).Work(0, chnTimeout)
	    <-chnTimeout
	}

The following code instantiates a serving node

	package main

	import (
	    "github.com/goat-pakage/goat/goat"
	)

	func main(){
	    port := 17002
	    chnTimeout := make(chan struct{})
	    freshMidAddress := "..."
	    nextNodeAddress := "..."
	    go goat.NewRingNode(port, freshMidAddress, nextNodeAddress).Work(0, chnTimeout)
	    <-chnTimeout
	}

Components can connect to the infrastructure by calling `goat.NewRingAgent("<registrationAddress>:<port>")` with the address of the registration node and the listening port provided (here 17000).

### Tree infrastructure

The following code is used to instantiate the registration node

	package main

	import (
	    "github.com/goat-pakage/goat/goat"
	)

	func main(){
	    port := 17000
	    nodesAddresses := []string{} // list of all the serving nodes in the cluster
	    chnTimeout := make(chan struct{})
	    go goat.NewTreeAgentRegistration(port, nodesAddresses).Work(0, chnTimeout)
	    <-chnTimeout
	}

The following code instantiates the root node

	package main

	import (
	    "github.com/goat-pakage/goat/goat"
	)

	func main(){
	    port := 17001
	    chnTimeout := make(chan struct{})
	    childNodesAddresses := []string{...}
	    go goat.NewTreeNode(port, "", childNodesAddresses).Work(0, chnTimeout)
	    <-chnTimeout
	}
	

The following code instantiates the a non-root serving node

	package main

	import (
	    "github.com/goat-pakage/goat/goat"
	)

	func main(){
	    port := 17002
	    chnTimeout := make(chan struct{})
	    childNodesAddresses := []string{}
	    parentAddress := ...
	    go goat.NewTreeNode(port, parentAddress, childNodesAddresses).Work(0, chnTimeout)
	    <-chnTimeout
	}

Components can connect to the infrastructure by calling `goat.NewTreeAgent("<registrationAddress>:<port>")` with the address of the registration node and the listening port provided (here 17000).
