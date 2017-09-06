package goat

import (
    "fmt"
    "strconv"
)

/*
Predicate represents a predicate to be satisfied by the receiver component of
a message sent.
*/
type Predicate interface {
    ImmediateSatisfy() (bool,bool)
    Satisfy(*Attributes) bool
    String() string
}

/*
Equal represents a predicate that is true iff the receiver component has the
attributes Attr1 and Attr2 both set and they evaluate to the same value.
*/
type Equal struct {
    Attr1 string
    Attr2 string
}
func (eq Equal) Satisfy(attr *Attributes) bool {
    a1Val, a1Exists := (*attr).Get(eq.Attr1)
    a2Val, a2Exists := (*attr).Get(eq.Attr2)
    return a1Exists && a2Exists && (a1Val == a2Val)
}
func (eq Equal) ImmediateSatisfy() (bool, bool) {
    return false, false
}
func (eq Equal) String() string {
    return fmt.Sprintf("=(%s,%s)", escape(eq.Attr1), escape(eq.Attr2))
}

/*
EqualImm represents a predicate that is true iff the receiver component has the
attribute Attr set to Val.
*/
type EqualImm struct {
    Attr string
    Val string
}
func (eq EqualImm) Satisfy(attr *Attributes) bool {
    aVal, aExists := (*attr).Get(eq.Attr)
    return aExists && (aVal == eq.Val)
}
func (eq EqualImm) ImmediateSatisfy() (bool, bool) {
    return false, false
}
func (eq EqualImm) String() string {
    return fmt.Sprintf("E(%s,%s)", escape(eq.Attr), escape(eq.Val))
}

type NotEqualImm struct {
    Attr string
    Val string
}
func (eq NotEqualImm) Satisfy(attr *Attributes) bool {
    aVal, aExists := (*attr).Get(eq.Attr)
    return aExists && (aVal != eq.Val)
}
func (eq NotEqualImm) ImmediateSatisfy() (bool, bool) {
    return false, false
}
func (eq NotEqualImm) String() string {
    return fmt.Sprintf("N(%s,%s)", escape(eq.Attr), escape(eq.Val))
}

type LowerImm struct {
    Attr string
    Val string
}
func (eq LowerImm) Satisfy(attr *Attributes) bool {
    aVal, aExists := (*attr).Get(eq.Attr)
    if !aExists {
        return false
    }
    aInt, err := strconv.Atoi(aVal)
    if err != nil {
        return false
    }
    vInt, err := strconv.Atoi(eq.Val)
    if err != nil {
        return false
    }
    return aInt < vInt
}
func (eq LowerImm) ImmediateSatisfy() (bool, bool) {
    return false, false
}
func (eq LowerImm) String() string {
    return fmt.Sprintf("l(%s,%s)", escape(eq.Attr), escape(eq.Val))
}

type LowerEqualImm struct {
    Attr string
    Val string
}
func (eq LowerEqualImm) Satisfy(attr *Attributes) bool {
    aVal, aExists := (*attr).Get(eq.Attr)
    if !aExists {
        return false
    }
    aInt, err := strconv.Atoi(aVal)
    if err != nil {
        return false
    }
    vInt, err := strconv.Atoi(eq.Val)
    if err != nil {
        return false
    }
    return aInt <= vInt
}
func (eq LowerEqualImm) ImmediateSatisfy() (bool, bool) {
    return false, false
}
func (eq LowerEqualImm) String() string {
    return fmt.Sprintf("L(%s,%s)", escape(eq.Attr), escape(eq.Val))
}

type GreaterImm struct {
    Attr string
    Val string
}
func (eq GreaterImm) Satisfy(attr *Attributes) bool {
    aVal, aExists := (*attr).Get(eq.Attr)
    if !aExists {
        return false
    }
    aInt, err := strconv.Atoi(aVal)
    if err != nil {
        return false
    }
    vInt, err := strconv.Atoi(eq.Val)
    if err != nil {
        return false
    }
    return aInt > vInt
}
func (eq GreaterImm) ImmediateSatisfy() (bool, bool) {
    return false, false
}
func (eq GreaterImm) String() string {
    return fmt.Sprintf("g(%s,%s)", escape(eq.Attr), escape(eq.Val))
}

type GreaterEqualImm struct {
    Attr string
    Val string
}
func (eq GreaterEqualImm) Satisfy(attr *Attributes) bool {
    aVal, aExists := (*attr).Get(eq.Attr)
    if !aExists {
        return false
    }
    aInt, err := strconv.Atoi(aVal)
    if err != nil {
        return false
    }
    vInt, err := strconv.Atoi(eq.Val)
    if err != nil {
        return false
    }
    return aInt >= vInt
}
func (eq GreaterEqualImm) ImmediateSatisfy() (bool, bool) {
    return false, false
}
func (eq GreaterEqualImm) String() string {
    return fmt.Sprintf("G(%s,%s)", escape(eq.Attr), escape(eq.Val))
}

/*
And represents a predicate that is true iff both the predicates P1 and P2 are true.
*/
type And struct {
    P1 Predicate
    P2 Predicate
}
func (and And) Satisfy(attr *Attributes) bool {
    return and.P1.Satisfy(attr) && and.P2.Satisfy(attr)
}
func (and And) ImmediateSatisfy() (bool, bool) {
    eval1, can1 := and.P1.ImmediateSatisfy()
    eval2, can2 := and.P2.ImmediateSatisfy()
    if can1 && can2 {
        return eval1 && eval2, true
    } else if (can1 && !eval1) || (can2 && !eval2){
        return false, true
    } else {
        return false, false
    }
}
func (and And) String() string {
    return fmt.Sprintf("&(%s,%s)", and.P1, and.P2)
}

/*
Or represents a predicate that is true iff either P1 or P2 is true (or both).
*/
type Or struct {
    P1 Predicate
    P2 Predicate
}
func (or Or) Satisfy(attr *Attributes) bool {
    return or.P1.Satisfy(attr) || or.P2.Satisfy(attr)
}
func (or Or) ImmediateSatisfy() (bool, bool) {
    eval1, can1 := or.P1.ImmediateSatisfy()
    eval2, can2 := or.P2.ImmediateSatisfy()
    if can1 && can2 {
        return eval1 || eval2, true
    } else if (can1 && eval1) || (can2 && eval2){
        return true, true
    } else {
        return false, false
    }
}
func (or Or) String() string {
    return fmt.Sprintf("|(%s,%s)", or.P1, or.P2)
}

/*
Not represents a predicate that is true iff the predicate P is false.
*/
type Not struct {
    P Predicate
}
func (not Not) Satisfy(attr *Attributes) bool {
    return !not.P.Satisfy(attr)
}
func (not Not) ImmediateSatisfy() (bool, bool) {
    eval, can := not.P.ImmediateSatisfy()
    if can {
        return !eval, true
    } else {
        return false, false
    }
}
func (not Not) String() string {
    return fmt.Sprintf("!(%s)", not.P)
}

/*
True represents a predicate that is always true.
*/
type True struct {}
func (t True) Satisfy(*Attributes) bool {
    return true
}
func (t True) ImmediateSatisfy() (bool, bool) {
    return true, true
}
func (t True) String() string {
    return "TT"
}

/*
And represents a predicate that is always false.
*/
type False struct {}
func (f False) Satisfy(*Attributes) bool {
    return false
}
func (f False) ImmediateSatisfy() (bool, bool) {
    return false, true
}
func (f False) String() string {
    return "FF"
}

func ToPredicate(s string) (Predicate, error){
    p, _, err := toPredicateInt(s, 0)
    return p, err
}

func toPredicateInt(s string, from int) (Predicate, int, error) {
    escapedS := (s)
    switch s[from: from+2] {
        case "=(":
            attr1, commaPos := unescape(escapedS, from+2)
            attr2, bracketPos := unescape(escapedS, commaPos+1)
            return Equal{attr1, attr2}, bracketPos+1, nil
        case "E(":
            attr, commaPos := unescape(escapedS, from+2)
            val, bracketPos := unescape(escapedS, commaPos+1)
            return EqualImm{attr, val}, bracketPos+1, nil
        case "N(":
            attr, commaPos := unescape(escapedS, from+2)
            val, bracketPos := unescape(escapedS, commaPos+1)
            return NotEqualImm{attr, val}, bracketPos+1, nil
        case "l(":
            attr, commaPos := unescape(escapedS, from+2)
            val, bracketPos := unescape(escapedS, commaPos+1)
            return LowerImm{attr, val}, bracketPos+1, nil
        case "L(":
            attr, commaPos := unescape(escapedS, from+2)
            val, bracketPos := unescape(escapedS, commaPos+1)
            return LowerEqualImm{attr, val}, bracketPos+1, nil
        case "g(":
            attr, commaPos := unescape(escapedS, from+2)
            val, bracketPos := unescape(escapedS, commaPos+1)
            return GreaterImm{attr, val}, bracketPos+1, nil
        case "G(":
            attr, commaPos := unescape(escapedS, from+2)
            val, bracketPos := unescape(escapedS, commaPos+1)
            return GreaterEqualImm{attr, val}, bracketPos+1, nil
        case "&(":
            p1, commaPos, _ := toPredicateInt(s, from+2)
            p2, bracketPos, _ := toPredicateInt(s, commaPos+1)
            return And{p1, p2}, bracketPos+1, nil
        case "|(":
            p1, commaPos, _ := toPredicateInt(s, from+2)
            p2, bracketPos, _ := toPredicateInt(s, commaPos+1)
            return Or{p1, p2}, bracketPos+1, nil
        case "!(":
            p, bracketPos, _ := toPredicateInt(s, from+2)
            return Not{p}, bracketPos+1, nil
        case "TT":
            return True{}, from+2, nil
        case "FF":
            return False{}, from+2, nil
        default:
            //TODO Error!
            return nil, from + 1, nil
    }
}
