package main

import (
	"context"
	"fmt"
	"math"
)

func Add(a, b float64) float64 {
	return a + b
}

func Subtract(a, b float64) float64 {
	return a - b
}

func Multiply(a, b float64) float64 {
	return a * b
}

func Divide(a, b float64) (float64, error) {
	if b == 0 {
		return 0, fmt.Errorf("divisor cannot be zero")
	}
	return a / b, nil
}

func Modulus(a, b float64) (float64, error) {
	if b == 0 {
		return 0, fmt.Errorf("divisor cannot be zero")
	}

	return math.Mod(a, b), nil
}

func Exponentiate(a, b float64) float64 {
	return math.Pow(a, b)
}

func SquareRoot(a float64) (float64, error) {
	if a < 0 {
		return 0, fmt.Errorf("cannot compute square root of a negative number")
	}

	return math.Sqrt(a), nil
}

func Factorial(ctx context.Context, n int) (float64, error) {
	if n < 0 {
		return 0, fmt.Errorf("cannot compute factorial of a negative number")
	}

	value := 1.0

	for i := range n {
		if err := ctx.Err(); err != nil {
			return 0, err
		}

		ifloat := float64(i + 1)
		value *= ifloat
	}

	return value, nil
}

func Abs(a float64) float64 {
	return math.Abs(a)
}

func Percentage(a, b float64) (float64, error) {
	if b == 0 {
		return 0, fmt.Errorf("cannot calculate percentage with a divisor of zero")
	}

	return (a / b) * 100, nil
}

func NatLogarithm(a float64) (float64, error) {
	if a <= 0 {
		return 0, fmt.Errorf("cannot calculate natural logarithm of a non-positive number")
	}

	return math.Log(a), nil
}

func Logarithm(a, b float64) (float64, error) {
	if a <= 0 {
		return 0, fmt.Errorf("cannot calculate logarithm with a non-positive base")
	}
	if b <= 0 {
		return 0, fmt.Errorf("cannot calculate logarithm with a non-positive argument")
	}

	return math.Log(a) / math.Log(b), nil
}
