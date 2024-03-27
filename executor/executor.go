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
	If struct {
		Condition string `json:"condition" validate:"required,javascript"`
		Trigger   string `json:"trigger" validate:"required,alphanum"`
	} `json:"if" validate:"required"`
	Else struct {
		Trigger string `json:"trigger" validate:"required,alphanum"`
	} `json:"else" validate:"required"`
}

type Output struct {
	Next string `json:"next" validate:"required,alphanum"`
}

type JavascriptExecutor struct {
	validator  *validate.Validate
	inputCache *Input // to prevent converting if already done
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
	return "if"
}

func (e *JavascriptExecutor) Name() string {
	return "If"
}

func (e *JavascriptExecutor) InputRules() map[string]interface{} {
	return utils.ExtractValidationRules(&Input{})
}

func (e *JavascriptExecutor) OutputRules() map[string]interface{} {
	return utils.ExtractValidationRules(&Output{})
}

func (e *JavascriptExecutor) Description() string {
	return "If executor performs a comparison based on the input and decides the next task to execute."
}

func (e *JavascriptExecutor) convertInput(input interface{}) (*Input, error) {
	if e.inputCache != nil {
		return e.inputCache, nil
	}

	e.inputCache = &Input{}
	if err := utils.ConvertToStruct(input, e.inputCache); err != nil {
		return nil, stacktrace.PropagateWithCode(err, types.ErrInvalidTask, "Error converting input to struct")
	}
	return e.inputCache, nil
}

func getTaskById(id string, tasks []*types.Task) *types.Task {
	for _, t := range tasks {
		if t.ID == id {
			return t
		}
	}
	return nil
}

func (e *JavascriptExecutor) Validate(ctx context.Context, task *types.Task, otherTasks []*types.Task) error {
	if task.Input == nil {
		return stacktrace.NewErrorWithCode(types.ErrInvalidInput, "Input is required")
	}
	// Make sure the input is a string
	if _, ok := task.Input.(string); !ok {
		return stacktrace.NewErrorWithCode(types.ErrInvalidInput, "Input must be a string")
	}
	return nil
}

func (e *JavascriptExecutor) Execute(ctx context.Context, task *types.Task, otherTasks []*types.Task) (interface{}, error) {
	logrus.Debugf("Executing task %s in a javascript executor", task.ID)

	// Ensure task input is a string
	input, ok := task.Input.(string)
	if !ok {
		return nil, stacktrace.NewErrorWithCode(types.ErrInvalidInput, "Expected input to be a string")
	}

	var err error
	//Check if the input contains template tags
	if strings.Contains(input, "{{") {
		input, err = utils.RenderInputTemplate(input, task, otherTasks)
		if err != nil {
			return nil, stacktrace.Propagate(err, "Error rendering input template")
		}
	}

	// We need to automatically wrap the input in a function so that we can return a value
	input = "function main() { return " + input + " }; var output = main(); output;"

	vm := otto.New()
	// Add task output from dependencies of this task to the VM.
	// These must be set as a global variable `deps` which is a map of task IDs to their output.
	if task.Dependencies != nil {
		vm.Set("deps", task.Dependencies)
	}

	if task.GlobalInput != nil {
		vm.Set("input", task.GlobalInput)
	}

	out, err := vm.Run(input)
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
