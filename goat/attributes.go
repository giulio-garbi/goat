package goat

/*
Attributes holds the set of attributes defined for a component. It should not be 
instantiated by the user. It is designed to allow transactions, but the completion
of transactions is demanded to the library (according to the AbC semantics).
*/
type Attributes struct {
	actual map[string]interface{}
	changes map[string]interface{}
}


func (attr *Attributes) init(attrM map[string]interface{}){
	attr.actual = map[string]interface{}{}
	for k, v := range attrM{
		attr.actual[k] = v
	}
}

/*
Get returns the value of the attribute x in the component. If the attribute x
has the value v associated, Get(x) returns v, True; otherwise if the attribute x
has no value associated, it returns "", False. Note that Get takes in account also
the uncommitted attribute modifications.
*/
func (attr *Attributes) Get(x string) (interface{}, bool){
	var out interface{} 
	has := false
	var val interface{}
	if attr.changes != nil{
		if val, has = attr.changes[x]; has {
			out = val
			return out, has
		}
	}
	if attr.actual != nil{
		if val, has = attr.actual[x]; has {
			out = val
		}
	}
	return out, has
}

/*
GetValue behaves like Get called with the same argument, but returns only the first 
return value (hence the value of attribute x or "" if not set).
*/
func (attr *Attributes) GetValue(x string) interface{}{
    val, has := attr.Get(x)
    if !has{
        panic("Attribute "+x+" not set")
    }
    return val
}

/*
Has behaves like Get called with the same argument, but returns only the second 
return value (hence True iff the attribute x has been set, else False).
*/
func (attr *Attributes) Has(x string) bool{
    _, has := attr.Get(x)
    return has
}

/*
Set sets the value of attribute key to val. Note that, since Attributes uses 
transactions, GetValue(key) == val until one of the following happens:
* the last committed value of attribute was val1 (where val1 != val), and rollback() is called;
* a call to Set(key, val2) is performed (where val2 != val).
*/
func (attr *Attributes) Set(key string, val interface{}){
	if attr.changes == nil{
		attr.changes = map[string]interface{}{key: val}
	} else {
		attr.changes[key] = val
	}
}

/*
commit completes the transaction with success. The new values of the attributes
are permanently saved. Returns True whether there was any change to the attribute values.
*/
func (attr *Attributes) commit() bool{
	if attr.actual == nil{
		attr.actual = attr.changes
		return attr.changes != nil && len(attr.changes) > 0
	} else {
		anyChange := len(attr.changes)>0
		for k, v := range attr.changes{
			attr.actual[k] = v
		}
		attr.changes = nil
		return anyChange
	}  
}

/*
rollback completes the transaction without success. The values of the attributes get restored
to the values of the last committed transaction.
*/
func (attr *Attributes) rollback(){
	attr.changes = nil
}

/*
Satisfy returns True iff the attributes satisfy the predicate p.
*/
func (attr *Attributes) Satisfy(p ClosedPredicate) bool{
	return p.Satisfy(attr)
}
