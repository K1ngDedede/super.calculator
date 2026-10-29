package main

import (
	"context"
	"errors"
	"math"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

type testContextKey string

func TestOperationFuncAndMetadata(t *testing.T) {
	c := require.New(t)
	shortcut := "+"
	op := operationFunc{
		metadata: OperationMetadata{
			Name:         "add",
			Arity:        "binary",
			Abbreviation: "+",
			Shortcut:     &shortcut,
		},
		calculate: func(_ context.Context, operandA float64, operandB *float64) (float64, error) {
			return operandA + *operandB, nil
		},
	}
	operandB := 2.0

	c.Equal("add", op.Metadata().Name)
	c.Equal("+", *op.Metadata().Shortcut)
	result, err := op.Calculate(context.Background(), 3, &operandB)
	c.NoError(err)
	c.Equal(5.0, result)

	metadata := operationMetadata("square_root", "unary", "sqrt", "")
	c.Equal(OperationMetadata{
		Name:         "square_root",
		Arity:        "unary",
		Abbreviation: "sqrt",
	}, metadata)
	c.Nil(metadata.Shortcut)
}

func TestSimpleBinaryOperation(t *testing.T) {
	c := require.New(t)
	op := simpleBinaryOperation("add", "+", "+", Add)
	operandB := 6.0

	result, err := op.Calculate(context.Background(), 10, &operandB)
	c.NoError(err)
	c.Equal(16.0, result)
	c.Equal("binary", op.Metadata().Arity)
	c.Equal("+", *op.Metadata().Shortcut)

	result, err = op.Calculate(context.Background(), 10, nil)
	c.ErrorIs(err, errMissingOperandB)
	c.Zero(result)
}

func TestBinaryOperationPropagatesCalculationError(t *testing.T) {
	c := require.New(t)
	expectedErr := errors.New("operation failed")
	op := binaryOperation("test", "?", "", func(float64, float64) (float64, error) {
		return 0, expectedErr
	})
	operandB := 2.0

	result, err := op.Calculate(context.Background(), 1, &operandB)
	c.ErrorIs(err, expectedErr)
	c.Zero(result)
	c.Nil(op.Metadata().Shortcut)
}

func TestContextBinaryOperation(t *testing.T) {
	c := require.New(t)
	seenContext := false
	op := contextBinaryOperation("test", "?", "", func(ctx context.Context, operandA, operandB float64) (float64, error) {
		seenContext = ctx.Value(testContextKey("marker")) == "present"
		return operandA + operandB, nil
	})
	ctx := context.WithValue(context.Background(), testContextKey("marker"), "present")
	operandB := 2.0

	result, err := op.Calculate(ctx, 3, &operandB)
	c.NoError(err)
	c.Equal(5.0, result)
	c.True(seenContext)

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	result, err = op.Calculate(canceled, 3, &operandB)
	c.ErrorIs(err, context.Canceled)
	c.Zero(result)

	expired, expire := context.WithCancel(context.Background())
	result, err = contextBinaryOperation("late", "?", "", func(context.Context, float64, float64) (float64, error) {
		expire()
		return 1, nil
	}).Calculate(expired, 1, &operandB)
	c.ErrorIs(err, context.Canceled)
	c.Zero(result)
}

func TestRequireBinaryOperands(t *testing.T) {
	c := require.New(t)
	seenContext := false
	calculate := requireBinaryOperands(func(ctx context.Context, operandA, operandB float64) (float64, error) {
		seenContext = ctx.Value(testContextKey("marker")) == true
		return operandA - operandB, nil
	})
	ctx := context.WithValue(context.Background(), testContextKey("marker"), true)

	result, err := calculate(ctx, 9, nil)
	c.ErrorIs(err, errMissingOperandB)
	c.Zero(result)
	c.False(seenContext)

	operandB := 4.0
	result, err = calculate(ctx, 9, &operandB)
	c.NoError(err)
	c.Equal(5.0, result)
	c.True(seenContext)
}

func TestUnaryOperation(t *testing.T) {
	c := require.New(t)
	expectedErr := errors.New("unary failed")
	op := unaryOperation("test", "u", "", func(operand float64) (float64, error) {
		if operand < 0 {
			return 0, expectedErr
		}
		return operand * 2, nil
	})

	result, err := op.Calculate(context.Background(), 4, nil)
	c.NoError(err)
	c.Equal(8.0, result)
	c.Nil(op.Metadata().Shortcut)

	unexpectedOperand := 1.0
	result, err = op.Calculate(context.Background(), 4, &unexpectedOperand)
	c.ErrorIs(err, errUnexpectedOperandB)
	c.Zero(result)

	result, err = op.Calculate(context.Background(), -1, nil)
	c.ErrorIs(err, expectedErr)
	c.Zero(result)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result, err = op.Calculate(ctx, 4, nil)
	c.ErrorIs(err, context.Canceled)
	c.Zero(result)
}

func TestContextUnaryOperation(t *testing.T) {
	c := require.New(t)
	seenContext := false
	op := contextUnaryOperation("test", "u", "", func(ctx context.Context, operand float64) (float64, error) {
		seenContext = ctx.Value(testContextKey("marker")) == "present"
		return operand + 1, nil
	})
	ctx := context.WithValue(context.Background(), testContextKey("marker"), "present")

	result, err := op.Calculate(ctx, 3, nil)
	c.NoError(err)
	c.Equal(4.0, result)
	c.True(seenContext)

	operandB := 1.0
	result, err = op.Calculate(ctx, 3, &operandB)
	c.ErrorIs(err, errUnexpectedOperandB)
	c.Zero(result)

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	result, err = op.Calculate(canceled, 3, nil)
	c.ErrorIs(err, context.Canceled)
	c.Zero(result)

	expired, expire := context.WithCancel(context.Background())
	result, err = contextUnaryOperation("late", "u", "", func(context.Context, float64) (float64, error) {
		expire()
		return 1, nil
	}).Calculate(expired, 1, nil)
	c.ErrorIs(err, context.Canceled)
	c.Zero(result)
}

func TestOperationMetadata(t *testing.T) {
	c := require.New(t)
	metadata := operationMetadata("add", "binary", "+", "+")

	c.Equal("add", metadata.Name)
	c.Equal("binary", metadata.Arity)
	c.Equal("+", metadata.Abbreviation)
	c.NotNil(metadata.Shortcut)
	c.Equal("+", *metadata.Shortcut)
}

func TestFactorialResultValidation(t *testing.T) {
	c := require.New(t)

	result, err := factorialResult(context.Background(), 5)
	c.NoError(err)
	c.Equal(120.0, result)

	result, err = factorialResult(context.Background(), 2.5)
	c.ErrorIs(err, errFactorialInteger)
	c.Zero(result)

	tooLarge := math.Exp2(float64(strconv.IntSize - 1))
	result, err = factorialResult(context.Background(), tooLarge)
	c.ErrorIs(err, errFactorialRange)
	c.Zero(result)

	result, err = factorialResult(context.Background(), -1)
	c.EqualError(err, ErrNegativeFact.Error())
	c.Zero(result)
}

func TestAllowedOperations(t *testing.T) {
	c := require.New(t)
	expectedArities := map[string]string{
		"add":          "binary",
		"subtract":     "binary",
		"multiply":     "binary",
		"divide":       "binary",
		"modulus":      "binary",
		"exponentiate": "binary",
		"square_root":  "unary",
		"factorial":    "unary",
	}

	c.Len(AllowedOperations, len(expectedArities))
	for name, arity := range expectedArities {
		operation, exists := AllowedOperations[name]
		c.True(exists, "operation %q should be registered", name)
		c.Equal(name, operation.Metadata().Name)
		c.Equal(arity, operation.Metadata().Arity)
	}
}
