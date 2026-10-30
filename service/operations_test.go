package main

import (
	"context"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestArithmeticOperations(t *testing.T) {
	c := require.New(t)

	c.Equal(16.0, Add(10, 6))
	c.Equal(4.0, Subtract(10, 6))
	c.Equal(60.0, Multiply(10, 6))
	c.Equal(8.0, Exponentiate(2, 3))
	c.Equal(3.0, Abs(-3))
}

func TestDivide(t *testing.T) {
	c := require.New(t)

	result, err := Divide(12, 3)
	c.NoError(err)
	c.Equal(4.0, result)

	result, err = Divide(1, 0)
	c.EqualError(err, ErrDivisorZero.Error())
	c.Zero(result)
}

func TestModulus(t *testing.T) {
	c := require.New(t)

	result, err := Modulus(11, 4)
	c.NoError(err)
	c.Equal(3.0, result)

	result, err = Modulus(11, 0)
	c.EqualError(err, ErrDivisorZero.Error())
	c.Zero(result)
}

func TestSquareRoot(t *testing.T) {
	c := require.New(t)

	result, err := SquareRoot(81)
	c.NoError(err)
	c.Equal(9.0, result)

	result, err = SquareRoot(-1)
	c.EqualError(err, ErrNegativeSqrt.Error())
	c.Zero(result)
}

func TestFactorial(t *testing.T) {
	c := require.New(t)

	tests := []struct {
		name     string
		input    int
		expected float64
		wantErr  error
	}{
		{name: "positive", input: 5, expected: 120},
		{name: "zero", input: 0, expected: 1},
		{name: "negative", input: -1, wantErr: ErrNegativeFact},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := require.New(t)
			result, err := Factorial(context.Background(), test.input)
			if test.wantErr != nil {
				c.EqualError(err, test.wantErr.Error())
				c.Zero(result)
				return
			}
			c.NoError(err)
			c.Equal(test.expected, result)
		})
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result, err := Factorial(ctx, 1000000)
	c.ErrorIs(err, context.Canceled)
	c.Zero(result)
}

func TestPercentage(t *testing.T) {
	c := require.New(t)

	result, err := Percentage(1, 4)
	c.NoError(err)
	c.Equal(25.0, result)

	result, err = Percentage(1, 0)
	c.EqualError(err, "cannot calculate percentage with a divisor of zero")
	c.Zero(result)
}

func TestNatLogarithm(t *testing.T) {
	c := require.New(t)

	result, err := NatLogarithm(1)
	c.NoError(err)
	c.Equal(0.0, result)

	for _, input := range []float64{0, -1} {
		result, err = NatLogarithm(input)
		c.EqualError(err, "cannot calculate natural logarithm of a non-positive number")
		c.Zero(result)
	}
}

func TestLogarithm(t *testing.T) {
	c := require.New(t)

	result, err := Logarithm(8, 2)
	c.NoError(err)
	c.InDelta(3, result, 1e-12)

	result, err = Logarithm(0, 8)
	c.EqualError(err, "cannot calculate logarithm with a non-positive base")
	c.Zero(result)

	result, err = Logarithm(2, 0)
	c.EqualError(err, "cannot calculate logarithm with a non-positive argument")
	c.Zero(result)
}

func TestExponentiateSpecialResults(t *testing.T) {
	c := require.New(t)

	c.True(math.IsNaN(Exponentiate(-1, 0.5)))
	c.True(math.IsInf(Exponentiate(1e308, 2), 1))
}
