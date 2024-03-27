# If Executor

The **If Executor** is a decision node that returns the next task to be performed based on the condition provided. The condition is written in Javascript and may contain Golang templating tags.

> **Note:** This project was created to be used in Lepsta's EngineOne workflow engine. As such, it is structured to be compatible with that use.

## EngineOne compatibility
To make sure that this project remains compatible with EngineOne, the following must be retained and left unchanged:

### `main.go`
If you have to change this file, please refrain from modifying the line `var Executor = executor.NewIfExecutor()`. Without this line, this will no longer qualify as an EngineOne executor.

### `executor/executor.go`
The `IfExecutor` struct must implement the following interface:

```go
type Executor interface {
	New() Executor
	ID() string
	Name() string
	Description() string
	InputRules() map[string]interface{}
	OutputRules() map[string]interface{}
	Validate(ctx context.Context, task *types.Task, tasks []*types.Task) error
	Execute(ctx context.Context, task *types.Task, tasks []*types.Task) (Output, error)
}
```

Anything else can be changed.

## Setup
After cloning this repository, you need to run:

```
go mod vendor
```
This will download all the dependencies.

## Testing
To run the unit tests, we use Ginkgo, so you will need to install it using:

```
go install github.com/onsi/ginkgo/v2/ginkgo
```
You can then run the tests using:

```
make test
```

## Build
To build this project into Golang modules, simply run:

```
make build
```

On an environment that supports Cgo.