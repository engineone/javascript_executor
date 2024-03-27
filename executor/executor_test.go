package executor_test

import (
	"context"

	"github.com/engineone/javascript_executor/executor"
	"github.com/engineone/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("JavascriptExecutor", func() {
	var (
		jsExecutor *executor.JavascriptExecutor
		ctx        context.Context
		task       *types.Task
	)

	BeforeEach(func() {
		jsExecutor = executor.NewJavascriptExecutor()
		ctx = context.Background()
		task = &types.Task{
			ID: "testTask",
			Input: &executor.Input{
				Source: "1+1;",
			},
		}
	})

	Describe("New", func() {
		It("should return a new JavascriptExecutor", func() {
			Expect(jsExecutor.New()).To(BeAssignableToTypeOf(&executor.JavascriptExecutor{}))
		})
	})

	Describe("ID", func() {
		It("should return the ID", func() {
			Expect(jsExecutor.ID()).To(Equal("javascript"))
		})
	})

	Describe("Name", func() {
		It("should return the name", func() {
			Expect(jsExecutor.Name()).To(Equal("Javascript"))
		})
	})

	Describe("Validate", func() {
		Context("with valid input", func() {
			It("should not return an error", func() {
				Expect(jsExecutor.Validate(ctx, task, nil)).To(BeNil())
			})
		})

		Context("with invalid input", func() {
			It("should return an error", func() {
				task.Input = nil
				Expect(jsExecutor.Validate(ctx, task, nil)).To(HaveOccurred())
			})
		})
	})

	Describe("Execute", func() {
		Context("with valid input", func() {
			It("should not return an error and return correct result", func() {
				result, err := jsExecutor.Execute(ctx, task, nil)
				Expect(err).To(BeNil())
				Expect(result).To(Equal(int64(2)))
			})
		})

		Context("with invalid input", func() {
			It("should return an error", func() {
				task.Input = map[string]interface{}{
					"source": "",
				}
				_, err := jsExecutor.Execute(ctx, task, nil)
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
