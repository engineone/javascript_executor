package executor

import (
	"context"
	"reflect"
	"strings"

	"github.com/engineone/types"
	"github.com/engineone/utils"
	validate "github.com/go-playground/validator/v10"
	"github.com/palantir/stacktrace"
	"github.com/robertkrimen/otto"
	"github.com/robertkrimen/otto/parser"
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

type IfExecutor struct {
	validator  *validate.Validate
	inputCache *Input // to prevent converting if already done
}

// NewIfExecutor creates a new IfExecutor
func NewIfExecutor() *IfExecutor {
	return &IfExecutor{
		validator: utils.NewValidator(),
	}
}

func (e *IfExecutor) New() types.Executor {
	return NewIfExecutor()
}

func (e *IfExecutor) ID() string {
	return "if"
}

func (e *IfExecutor) Name() string {
	return "If"
}

func (e *IfExecutor) InputRules() map[string]interface{} {
	return utils.ExtractValidationRules(&Input{})
}

func (e *IfExecutor) OutputRules() map[string]interface{} {
	return utils.ExtractValidationRules(&Output{})
}

func (e *IfExecutor) Description() string {
	return "If executor performs a comparison based on the input and decides the next task to execute."
}

func (e *IfExecutor) convertInput(input interface{}) (*Input, error) {
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

func (e *IfExecutor) Validate(ctx context.Context, task *types.Task, otherTasks []*types.Task) error {

	if task.Input == nil {
		return stacktrace.NewErrorWithCode(types.ErrInvalidTask, "Input is required")
	}

	if reflect.TypeOf(task.Input).Kind() != reflect.Map {
		return stacktrace.NewErrorWithCode(types.ErrInvalidTask, "Input must be an object")
	}

	v := validate.New()
	err := v.RegisterValidation("javascript", func(fl validate.FieldLevel) bool {
		_, err := parser.ParseFile(nil, "input", fl.Field().String(), 0)
		if err != nil {
			logrus.Errorf("Error validating javascript: %v", err)
			return false
		}
		return true
	})
	if err != nil {
		return stacktrace.Propagate(err, "Error registering javascript validation")
	}

	input, err := e.convertInput(task.Input)
	if err != nil {
		return stacktrace.Propagate(err, "Failed to convert input")
	}

	if err := v.Struct(input); err != nil {
		return stacktrace.PropagateWithCode(err, types.ErrInvalidTask, "Input validation failed")
	}

	if t := getTaskById(input.If.Trigger, otherTasks); t == nil {
		return stacktrace.NewErrorWithCode(types.ErrInvalidTask, "If trigger %s does not exist in the workflow", input.If.Trigger)
	}
	if t := getTaskById(input.Else.Trigger, otherTasks); t == nil {
		return stacktrace.NewErrorWithCode(types.ErrInvalidTask, "Else trigger %s does not exist in the workflow", input.Else.Trigger)
	}
	return nil
}

func (e *IfExecutor) Execute(ctx context.Context, task *types.Task, otherTasks []*types.Task) (interface{}, error) {
	logrus.Debugf("Executing task %s in an if executor", task.ID)
	input, err := e.convertInput(task.Input)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Failed to convert input")
	}

	code := "if (" + input.If.Condition + ") { return '" + input.If.Trigger + "' } else { return '" + input.Else.Trigger + "' }"
	// We need to automatically wrap the input in a function so that we can return a value
	code = "function main() { " + code + " }; var output = {'next': main()}; output;"

	// Make sure we render template tags in the code
	// First check if the input contains template tags
	if strings.Contains(code, "{{") {
		var err error
		code, err = utils.RenderInputTemplate(code, task, otherTasks)
		if err != nil {
			return nil, stacktrace.Propagate(err, "Error rendering input template")
		}
	}

	vm := otto.New()
	// Add task output from dependencies of this task to the VM.
	// These must be set as a global variable `deps` which is a map of task IDs to their output.
	if task.Dependencies != nil {
		vm.Set("deps", task.Dependencies)
	}

	if task.GlobalInput != nil {
		vm.Set("input", task.GlobalInput)
	}

	out, err := vm.Run(code)
	if err != nil {
		return nil, stacktrace.NewError("Error executing javascript: %v", err)
	}

	if !out.IsObject() {
		return nil, stacktrace.NewError("Expected output to be an object with a trigger field")
	}

	t, err := out.Export()
	return t, stacktrace.Propagate(err, "Invalid output type")
}
