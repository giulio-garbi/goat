package goat

import(
    "fmt"
)

func dprintln(args ...interface{}) (int, error){
    if isDebug(){
        return fmt.Println(args...)
    } else {
        return 0, nil
    }
}

func dprint(args ...interface{}) (int, error){
    if isDebug(){
        return fmt.Print(args...)
    } else {
        return 0, nil
    }
}

func dprintf(format string, a ...interface{}) (int, error){
    if isDebug(){
        return fmt.Printf(format, a...)
    } else {
        return 0, nil
    }
}
