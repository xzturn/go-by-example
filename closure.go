// A closure is a function value that references variables from outside its body.
// The function may access and assign to the referenced variables; in this sense 
// the function is "bound" to the variables.
//
package main

import "fmt"

// fibonacci is a function that returns a function that returns an int.
func fibonacci() func() int {
    x, y := 0, 1
    return func() int {
        z := x + y
        x, y = y, z
        return z
    }
}

func main() {
    f := fibonacci()
    for i := 0; i < 10; i++ {
        fmt.Println(f())
    }
}
