package executor_test

import (
	"context"

	if_executor "github.com/engineone/if_executor/executor"
	"github.com/engineone/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/palantir/stacktrace"
)

var _ = Describe("If Executor", func() {
	var (
		executor *if_executor.IfExecutor
		task     *types.Task
		tasks    []*types.Task
	)

	BeforeEach(func() {
		executor = if_executor.NewIfExecutor()
		task = &types.Task{
			ID:       "task1",
			Executor: "if",
			Input: map[string]interface{}{
				"condition": true,
			},
		}

		tasks = []*types.Task{
			{
				ID:       "task2",
				Executor: "sleep",
				Input:    10,
			},
			{
				ID:       "sample",
				Executor: "sleep",
				Input:    10,
			},
			task,
		}
	})

	Describe("Validate", func() {
		Context("When the input is valid", func() {
			It("should return nil", func() {
				task.Input = map[string]interface{}{
					"if": map[string]interface{}{
						"condition": "task1.succeeded === true",
						"trigger":   "sample",
					},
					"else": map[string]interface{}{
						"trigger": "sample",
					},
				}
				err := executor.Validate(context.TODO(), task, tasks)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("When the input is nil", func() {
			It("should return an error with code ErrInvalidInput", func() {
				task.Input = nil
				err := executor.Validate(context.TODO(), task, tasks)
				Expect(err).To(HaveOccurred())
				Expect(stacktrace.GetCode(err)).To(Equal(types.ErrInvalidTask))
				Expect(err.Error()).To(ContainSubstring("Input is required"))
			})
		})

		Context("When the input is not a map", func() {
			It("should return an error with code ErrInvalidInput", func() {
				task.Input = "invalid input"
				err := executor.Validate(context.TODO(), task, tasks)
				Expect(err).To(HaveOccurred())
				Expect(stacktrace.GetCode(err)).To(Equal(types.ErrInvalidTask))
				Expect(err.Error()).To(ContainSubstring("Input must be an object"))
			})
		})

		Context("When the input does not pass validation rules", func() {
			It("should return an error with code ErrInvalidInput", func() {
				task.Input = map[string]interface{}{
					"condition": "invalid condition",
				}
				err := executor.Validate(context.TODO(), task, tasks)
				Expect(err).To(HaveOccurred())
				Expect(stacktrace.GetCode(err)).To(Equal(types.ErrInvalidTask))
			})
		})

		Context("When input conditions are not a javascript", func() {
			BeforeEach(func() {
				task.Input = map[string]interface{}{
					"if": map[string]interface{}{
						"trigger": "task1",
					},
					"else": map[string]interface{}{
						"trigger": "task2",
					},
				}
			})

			It("should return an error if the condition is not valid javascript", func() {
				task.Input.(map[string]interface{})["if"].(map[string]interface{})["condition"] = "do it"
				err := executor.Validate(context.TODO(), task, tasks)
				Expect(err).To(HaveOccurred())
				Expect(stacktrace.GetCode(err)).To(Equal(types.ErrInvalidTask))
				Expect(err.Error()).To(ContainSubstring("Input validation failed"))
			})

			It("should pass when the condition is valid javascript", func() {
				task.Input.(map[string]interface{})["if"].(map[string]interface{})["condition"] = "task1.succeeded === true"
				err := executor.Validate(context.TODO(), task, tasks)
				Expect(err).NotTo(HaveOccurred())
			})

			It("must return task1 as the next trigger", func() {
				task.Input.(map[string]interface{})["if"].(map[string]interface{})["condition"] = "true"
				err := executor.Validate(context.TODO(), task, tasks)
				Expect(err).NotTo(HaveOccurred())
				out, err := executor.Execute(context.Background(), task, tasks)
				Expect(err).NotTo(HaveOccurred())
				Expect(out.(map[string]interface{})["next"]).To(Equal("task1"))
			})
		})
	})
})
