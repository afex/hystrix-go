package hystrix

type Executor struct {}

var executor_pools = make(map[string]chan *Executor)

func (executor *Executor) Run(command *Command) {
	command.Run(command.ResultChannel)
}