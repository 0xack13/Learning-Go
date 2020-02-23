# Golang Notes

### From A Tour of Go
#### `Basics`
- Go exports names if they begin with a capital letter.
- type comes after var names in golang
- functions can return any number of results
- naked return & named return values
  - named after args. naked return returns named values. only use with short funcs. 
- var statements can be at package or func level
- var declarations (=) can have initializers, in which case type can b eomitted.
- short variable declarations (:=, no var keyword) only available in functions. outside of funcs, every statement must begin with a keyword.
- Go basic types: bool, string, int, int8-64, uint, uint8-64, uintptr, byte (alias for uint8), rune (alias for int32 - unicode code point), float32, float64, complex64, complex128
- vars declared without initial value are given zero value for their type
- type conversions - T(v) convert v to type T
  - In Go (unlike C), assignments between items of diff type requires explicit conversion
- const keyword for constants. cannot use short (:=) declarations with constants.
  - only char, string, bool, or numeric vals
- "For" is the only looping construct
  - no parens around for statement and braces always required.
  - can add empty init and post statements w/ ; (or nothing, then you have a while)
- infinite loop is just an empty "for"
- if also no parens
- if can start with short var declaration that is limited to that scope (+ else)
- switch: go only runs the selected case and not all that follow. break if supplied automatically. can also start with short decl
- can use empty switch

#### `New feature: defer`
-  defer statement defers exec of func until surrounding function returns.
  - args are evaluated immediately, but the func call is not executed until surrounding func returns
  - deferred func calls are pushed onto a stack. when func returns, its deferred calls are executed in LIFO order.
#### `Pointers`
- pointer holds mem address of val
- type *T is pointer to a T value. Its zero value is nil
- & operator generates a pointer to its operand
- The * operator denotes the pointer's underlying value (dereferences/indirects)
  - Unlike C, Go has no pointer arithmetic
#### `Structs`
- struct is a collection of fields
- access via dot
struct pointers & field access: instead of writing (*p).X w/ a pointer, you can just write p.X without the explicit dereference.
- struct literals, you can allocate structs w/ a subset of fields set explicitly.
- p = &StructType{x:1} returns pointer to struct
#### `Arrays and Slices`
- [n]T is array of n values of type T
  - var a [10]int
- Arrays cannot be resized in Go
- Slices []T: slice w/ elements of type T
  - a[low: high]
  - includes first element, excludes last
- slice doesn't store data, it just references section of undelying array.
  - changing slice changes array & its other slices!
- array & slice literals: [3]bool{true, false, true}
[]]bool{true, false, true}
- slices have high & low defaults like python
- slices have length & capacity:
  - length is num elems in slice
  - capacity is num elements in array counting from first elem of slice
  - can get w/ len(s) & cap(s)
  - can extend len of slice by reslicing up to cap
- Make: create slices w/ make to create dynamically sized arrays.
  - make func allocates zeroed array & returns a lsice to that array.
- you can make slices of slices
- you can append to slices w/ build-in func "append"
  - automatically allocates a bigger array & repoints your slice if underlying array is too small 
- for i, v := range pow (iterates w/ index & `copy` of elem)
  - can omit v or use anom var _ for i
#### `Maps (keys to values)`
- zero is nil
- make returns map of given type initalized & ready to use.
  - var m map[string]Vertex
  - m = make(map[string]Vertex)
  - m["Bell Labs"] = Vertex{40.2, -42,5}
- map literals are like struct literals, but keys are necessary
  - var m = map[string]Vertex{
      "test1": Vertex{1,2},
      "test2": Vertex{3,4}
  }
- if top level type is just a type name (like vertex above), you can omit it
- mutating maps
  - m[key] = elem
- elem = m[key]
- delete(m, key)
- test w/ 2-value assignment
  - elem, exists_bool = m[key]
  - returns 0 value w/ false
#### `functions`
- funcs are values and can be passed around as args & return values
  - func compute(fn func(float64, float64) float64)
- Go functions can be `closures`
  - closure if func value that references variables from outside its body. func may access and assign to the referenced vars.
  - this can give a function state between calls via the variables the func is bound to.
#### `Methods`
- No classes in Go, but you can define methods on types.
- a method is a special func w/ receiver argument
- receiver appears in its own arg list between func keyword and method name
- Abs is a method w/ receiver type Vertex named v
  - func (v Vertex) Abs() float64 {
    return math.Sqrt(v.X * v.X + v.Y * v.Y)
}
  - could just be rewritten as a func that takes a Vertex arg
- you can declare methods on non-struct types.
  - type MyFloat float64
  - func (f MyFloat) Abs() float64 {
      ...
  }
- methods can have pointer receivers
  - func (v *Vertex) Scale(f float64) {...}
  - methods w/ pointer receivers can modify the value to which the receiver points
  - without this, you would operate on a copy of the value
- methods and pointer indirection
  - as a convenience, go interprets v.Scale(5) as (&v).Scale(5)
  - same happens in reverse w/ methods w/ value receivers
- in general, all methods on a given type should have either value or pointer receivers, but not a mixture of both (why?)
#### `Interfaces`
- An interface type has a set of method signatures
- the value of an interface type can hold any value that implements those methods.
- interfaces are implemented implicitly
  - type implements an interface by implementing the methods. there is no declaration of intent.
  - desire was to decouple the definition of an interface from its implementation, which can then appear in any package without prearrangement  `???`
- under the hood, interfaces are a tuple of value & concrete type
  - (value, type)
- in Go it's common to write methods that gracefully handle being called with nil receivers `???`
- calling method on nil interface will cause a runtime error
- use empty interface for code that handles values of unknown type, like fmt.Print
- Stringer interface: implements String() string method
#### `Type assertions`
- provides access to an interface value's underlying concrete value
- t :=i.(T)
  - if i does not hold T, statement will trigger panic
  - if interface i holds concrete type T, it assigns the underlying T value to var t
  - can also do t, ok := i.(T)
- type assertion will probably be used inside of function that accepts the empty interface (interface is, itself a type) to find the underlying concret value
- type switches
  - easy way to do multiple type assertions in series
#### `Errors`
- error type is built-in interface
- funcs often return an error value, and calling code should handle errors by testing whether error is nil
- i, err := strconv.Atoi("42")
- if err != nil {...}
#### `Readers`
- io.Reader interface represents read end of a stream of data
  - func (T) Read(b []byte) (n int, err error)
  - populates the byte slice w/ data and returns num populated and error.
  - io.EOF error indicates stream end
#### `Goroutines`
- goroutine is a lightweight thread managed by the Go runtime
  - go f(x, y, z)
  - start by calling f(x, y, z)
  - eval of f, x, y, and z happen in current goroutine and execution of f happens in new
- goroutines run in the same address space, so access to shared memory must be synchronized. sync package can provides useful primitives
#### `Channels`
- a typed conduit through which you can send and receive values w/ the channel operator: <-
- ch <- v
  - send v to channel ch
- v := <-ch
  - receive from ch, and assign value to v
- data flows in direction of arrow
- create before use w/ make, like slices & maps
  - ch := make(chan int)
- by default, sends & receives block until the other side is ready - allows goroutines to synch without explicit locks or cond vars `????`
- channels can be buffered, provide buffer length as second arg to make
  - sends to buffered channel block only when buffer is full.
  - receives block only whne the buffer is empty
- Range and close
  - sender can close a channel to indicate no more values will be sent.
  - receivers can test whether val has been closed by assigning second val to receive expression
    - v, ok := <-ch
  - i := range c recieves vals from channel until closed
    - only sender can close channel, never receiver.
    - closing only necessary when receiver must be told no more vals are coming
- Select statement lets goroutine wait on multiple comm ops
  - select blocks until one of its cases can run, then executes that case
  - chooses one at random if multiple ready
  - can have default case
- sync.Mutex
  - if we don't need communication, but we want to make sur eonly one goroutine can access a var at a time
  - "mutual exclusion"
  - Mutex has Lock and Unlock methods
#### `More`
- A go motto: "Share memory by communicating, don't communicate by sharing memory"
  - What does this mean regarding using things like channels instead of mutexes
- Error/exception handling
  - lots of val, err := func()
    - if err != nil: {...}
- diff between var and type declarations.