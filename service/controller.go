package main

import (
	"context"
	"fmt"
	"math"
	"strconv"
)

type Operation interface {
	Metadata() OperationMetadata
	Calculate(ctx context.Context, operandA float64, operandB *float64) (float64, error)
}

type OperationMetadata struct {
	Name         string  `json:"name"`
	Arity        string  `json:"arity"`
	Abbreviation string  `json:"abbreviation"`
	Shortcut     *string `json:"shortcut,omitempty"`
}

type operationFunc struct {
	metadata  OperationMetadata
	calculate func(ctx context.Context, operandA float64, operandB *float64) (float64, error)
}

func (operation operationFunc) Metadata() OperationMetadata {
	return operation.metadata
}

func (operation operationFunc) Calculate(ctx context.Context, operandA float64, operandB *float64) (float64, error) {
	return operation.calculate(ctx, operandA, operandB)
}

var (
	ErrDivisorZero  = fmt.Errorf("divisor cannot be zero")
	ErrNegativeSqrt = fmt.Errorf("cannot compute square root of a negative number")
	ErrNegativeFact = fmt.Errorf("cannot compute factorial of a negative number")

	errMissingOperandB    = fmt.Errorf("operand_b is required")
	errUnexpectedOperandB = fmt.Errorf("operand_b is not allowed for this operation")
	errFactorialInteger   = fmt.Errorf("factorial operand must be an integer")
	errFactorialRange     = fmt.Errorf("factorial operand is out of range")
	errInvalidOperation   = fmt.Errorf("Invalid operation")

	AllowedOperations = map[string]Operation{
		"add":          simpleBinaryOperation("add", "+", "+", Add),
		"subtract":     simpleBinaryOperation("subtract", "-", "-", Subtract),
		"multiply":     simpleBinaryOperation("multiply", "*", "*", Multiply),
		"divide":       binaryOperation("divide", "/", "/", Divide),
		"modulus":      binaryOperation("modulus", "%", "%", Modulus),
		"exponentiate": simpleBinaryOperation("exponentiate", "^", "^", Exponentiate),
		"square_root":  unaryOperation("square_root", "sqrt", "", SquareRoot),
		"factorial":    contextUnaryOperation("factorial", "!", "!", factorialResult),
	}
)

func simpleBinaryOperation(name, abbreviation, shortcut string, calculate func(float64, float64) float64) Operation {
	return binaryOperation(name, abbreviation, shortcut, func(operandA, operandB float64) (float64, error) {
		return calculate(operandA, operandB), nil
	})
}

func binaryOperation(name, abbreviation, shortcut string, calculate func(float64, float64) (float64, error)) Operation {
	return contextBinaryOperation(name, abbreviation, shortcut, func(_ context.Context, operandA, operandB float64) (float64, error) {
		return calculate(operandA, operandB)
	})
}

func contextBinaryOperation(name, abbreviation, shortcut string, calculate func(context.Context, float64, float64) (float64, error)) Operation {
	calculateWithOperands := requireBinaryOperands(calculate)
	return operationFunc{
		metadata: operationMetadata(name, "binary", abbreviation, shortcut),
		calculate: func(ctx context.Context, operandA float64, operandB *float64) (float64, error) {
			if err := ctx.Err(); err != nil {
				return 0, err
			}

			result, err := calculateWithOperands(ctx, operandA, operandB)
			if contextErr := ctx.Err(); contextErr != nil {
				return 0, contextErr
			}

			return result, err
		},
	}
}

func unaryOperation(name, abbreviation, shortcut string, calculate func(float64) (float64, error)) Operation {
	return contextUnaryOperation(name, abbreviation, shortcut, func(_ context.Context, operand float64) (float64, error) {
		return calculate(operand)
	})
}

func contextUnaryOperation(name, abbreviation, shortcut string, calculate func(context.Context, float64) (float64, error)) Operation {
	return operationFunc{
		metadata: operationMetadata(name, "unary", abbreviation, shortcut),
		calculate: func(ctx context.Context, operandA float64, operandB *float64) (float64, error) {
			if err := ctx.Err(); err != nil {
				return 0, err
			}

			if operandB != nil {
				return 0, errUnexpectedOperandB
			}

			result, err := calculate(ctx, operandA)
			if contextErr := ctx.Err(); contextErr != nil {
				return 0, contextErr
			}

			return result, err
		},
	}
}

func requireBinaryOperands(calculate func(context.Context, float64, float64) (float64, error)) func(context.Context, float64, *float64) (float64, error) {
	return func(ctx context.Context, operandA float64, operandB *float64) (float64, error) {
		if operandB == nil {
			return 0, errMissingOperandB
		}

		return calculate(ctx, operandA, *operandB)
	}
}

func operationMetadata(name, arity, abbreviation, shortcut string) OperationMetadata {
	metadata := OperationMetadata{
		Name:         name,
		Arity:        arity,
		Abbreviation: abbreviation,
	}

	if shortcut != "" {
		metadata.Shortcut = &shortcut
	}

	return metadata
}

func factorialResult(ctx context.Context, operand float64) (float64, error) {
	if math.Trunc(operand) != operand {
		return 0, errFactorialInteger
	}

	if operand < -math.Exp2(float64(strconv.IntSize-1)) || operand >= math.Exp2(float64(strconv.IntSize-1)) {
		return 0, errFactorialRange
	}

	result, err := Factorial(ctx, int(operand))

	return result, err
}
