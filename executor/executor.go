package executor

import (
	"context"
	"strings"

	"github.com/engineone/types"
	"github.com/engineone/utils"
	validate "github.com/go-playground/validator/v10"
	"github.com/palantir/stacktrace"
	"github.com/robertkrimen/otto"
	"github.com/sirupsen/logrus"
)

type Input struct {
	Source string `json:"source" validate:"required,javascript"`
}

type Output struct {
	Return interface{} `json:"return"`
}

type JavascriptExecutor struct {
	validator   *validate.Validate
	cachedInput *Input
}

// NewJavascriptExecutor creates a new JavascriptExecutor
func NewJavascriptExecutor() *JavascriptExecutor {
	return &JavascriptExecutor{
		validator: utils.NewValidator(),
	}
}

func (e *JavascriptExecutor) New() types.Executor {
	return NewJavascriptExecutor()
}

func (e *JavascriptExecutor) ID() string {
	return "javascript"
}

func (e *JavascriptExecutor) Name() string {
	return "Javascript"
}

func (e *JavascriptExecutor) InputRules() map[string]interface{} {
	return utils.ExtractValidationRules(&Input{})
}

func (e *JavascriptExecutor) OutputRules() map[string]interface{} {
	return utils.ExtractValidationRules(&Output{})
}

func (e *JavascriptExecutor) Description() string {
	return "Javascript executor that can execute javascript code. The input must be a string containing valid javascript code. The output will be the result of the javascript code."
}

func (e *JavascriptExecutor) converInput(input interface{}) (*Input, error) {
	if e.cachedInput != nil {
		return e.cachedInput, nil
	}

	e.cachedInput = &Input{}
	if err := utils.ConvertToStruct(input, e.cachedInput); err != nil {
		return nil, stacktrace.Propagate(err, "Error converting input to struct")
	}
	return e.cachedInput, nil
}

func (e *JavascriptExecutor) Validate(ctx context.Context, task *types.Task, otherTasks []*types.Task) error {
	if task.Input == nil {
		return stacktrace.NewErrorWithCode(types.ErrInvalidInput, "Input is required")
	}

	if _, err := e.converInput(task.Input); err != nil {
		return stacktrace.PropagateWithCode(err, types.ErrInvalidInput, "Error converting input to struct")
	}
	return nil
}

func (e *JavascriptExecutor) Execute(ctx context.Context, task *types.Task, otherTasks []*types.Task) (interface{}, error) {
	logrus.Debugf("Executing task %s in a javascript executor", task.ID)

	input, err := e.converInput(task.Input)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Error converting input to struct")
	}

	// Ensure task input is not an empty string
	if input.Source == "" {
		return nil, stacktrace.NewErrorWithCode(types.ErrInvalidInput, "Expected input to be a string")
	}

	//Check if the input contains template tags
	source := input.Source
	if strings.Contains(input.Source, "{{") {
		source, err = utils.RenderInputTemplate(input.Source, task, otherTasks)
		if err != nil {
			return nil, stacktrace.Propagate(err, "Error rendering input template")
		}
	}

	// We need to automatically wrap the input in a function so that we can return a value
	source = "function main() { return " + input.Source + " }; var output = main(); output;"

	vm := otto.New()
	// Add task output from dependencies of this task to the VM.
	// These must be set as a global variable `deps` which is a map of task IDs to their output.
	if task.Dependencies != nil {
		vm.Set("deps", task.Dependencies)
	}

	if task.GlobalInput != nil {
		vm.Set("input", task.GlobalInput)
	}

	out, err := vm.Run(source)
	if err != nil {
		return nil, stacktrace.NewErrorWithCode(types.ErrExecutorFailed, "Error executing javascript: %v", err)
	}

	if out.IsString() {
		return out.String(), nil
	} else if out.IsNumber() {
		t, err := out.ToInteger()
		return t, stacktrace.Propagate(err, "Invalid output type is not an integer")
	} else if out.IsBoolean() {
		t, err := out.ToBoolean()
		return t, stacktrace.Propagate(err, "Invalid output type is not a boolean")
	} else if out.IsUndefined() || out.IsNull() {
		return nil, nil
	} else if out.IsFunction() {
		return nil, stacktrace.NewErrorWithCode(types.ErrExecutorFailed, "Invalid output type is a function")
	}

	t, err := out.Export()
	return t, stacktrace.Propagate(err, "Invalid output type")
}
