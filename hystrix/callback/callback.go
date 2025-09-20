package callback

var circuitCallback map[string]stateFunc

//State is a type to hold Circuit-state this will be used while calling stateFunc on State change
type State string

const (
	//Open is a state to indicate that Circuit state is Open
	Open = "Open"
	//Close is a state to indicate that Circuit state is Close
	Close = "Close"
	//AllowSingle is a state to indicate that Circuit state is AllowSingle or trying to Open Circuit
	AllowSingle = "Allow Single"
)

type stateFunc func(name string, state State)

func init() {
	circuitCallback = make(map[string]stateFunc)
}

//Register adds callback for a circuit
func Register(name string, callbackFunc stateFunc) {
	circuitCallback[name] = callbackFunc
}

//Invoke is a function to invoke Callback function in a goroutine on State change
func Invoke(name string, state State) {
	callbackFunc, _ := circuitCallback[name]
	if callbackFunc != nil {
		go callbackFunc(name, state)
	}
}
