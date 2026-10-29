package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewHandlerGetOperations(t *testing.T) {
	c := require.New(t)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()

	NewHandler().ServeHTTP(response, request)

	c.Equal(http.StatusOK, response.Code)
	c.Equal("application/json", response.Header().Get("Content-Type"))

	var body struct {
		Operations []OperationMetadata `json:"operations"`
	}
	c.NoError(json.NewDecoder(response.Body).Decode(&body))
	c.Len(body.Operations, len(AllowedOperations))
	c.Equal("add", body.Operations[0].Name)
	c.Equal("subtract", body.Operations[len(body.Operations)-1].Name)
	for i := 1; i < len(body.Operations); i++ {
		c.LessOrEqual(body.Operations[i-1].Name, body.Operations[i].Name)
	}
}

func TestHandleRequestPathAndMethodErrors(t *testing.T) {
	c := require.New(t)
	handler := NewHandler()

	notFound := httptest.NewRecorder()
	handler.ServeHTTP(notFound, httptest.NewRequest(http.MethodGet, "/unknown", nil))
	c.Equal(http.StatusNotFound, notFound.Code)

	methodNotAllowed := httptest.NewRecorder()
	handler.ServeHTTP(methodNotAllowed, httptest.NewRequest(http.MethodPut, "/", nil))
	c.Equal(http.StatusMethodNotAllowed, methodNotAllowed.Code)
	c.Equal("GET, POST", methodNotAllowed.Header().Get("Allow"))
}

func TestHandleCalculationSuccess(t *testing.T) {
	c := require.New(t)

	tests := []struct {
		name     string
		body     string
		expected float64
	}{
		{name: "add", body: `{"operation":"add","operand_a":10,"operand_b":6}`, expected: 16},
		{name: "subtract", body: `{"operation":"subtract","operand_a":10,"operand_b":4}`, expected: 6},
		{name: "multiply", body: `{"operation":"multiply","operand_a":3,"operand_b":4}`, expected: 12},
		{name: "divide", body: `{"operation":"divide","operand_a":12,"operand_b":3}`, expected: 4},
		{name: "modulus", body: `{"operation":"modulus","operand_a":11,"operand_b":4}`, expected: 3},
		{name: "exponentiate", body: `{"operation":"exponentiate","operand_a":2,"operand_b":3}`, expected: 8},
		{name: "square root", body: `{"operation":"square_root","operand_a":81}`, expected: 9},
		{name: "factorial", body: `{"operation":"factorial","operand_a":5}`, expected: 120},
	}
	c.Len(tests, len(AllowedOperations))

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := require.New(t)
			response := performRequest(NewHandler(), http.MethodPost, "/", test.body)

			c.Equal(http.StatusOK, response.Code)
			c.Equal("application/json", response.Header().Get("Content-Type"))
			body := decodeCalculationResponse(t, response)
			c.Equal("", body.Error)
			c.Equal(test.expected, body.Result)
		})
	}
}

func TestHandleCalculationMalformedRequests(t *testing.T) {
	c := require.New(t)

	tests := []struct {
		name string
		body string
	}{
		{name: "invalid json", body: `{"operation":`},
		{name: "non-json", body: `not json`},
		{name: "nonnumeric operand", body: `{"operation":"add","operand_a":"abc","operand_b":1}`},
		{name: "missing operand", body: `{"operation":"add","operand_b":1}`},
		{name: "unknown field", body: `{"operation":"add","operand_a":1,"extra":2}`},
		{name: "trailing json value", body: `{"operation":"add","operand_a":1} {}`},
		{name: "null operand", body: `{"operation":"add","operand_a":null}`},
	}
	c.Len(tests, 7)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := require.New(t)
			response := performRequest(NewHandler(), http.MethodPost, "/", test.body)

			c.Equal(http.StatusBadRequest, response.Code)
			body := decodeCalculationResponse(t, response)
			c.Equal("err", body.Result)
			c.NotEmpty(body.Error)
		})
	}
}

func TestHandleCalculationErrors(t *testing.T) {
	c := require.New(t)

	tests := []struct {
		name          string
		body          string
		expectedError string
	}{
		{name: "unknown operation", body: `{"operation":"unknown","operand_a":1}`, expectedError: "Invalid operation"},
		{name: "missing second operand", body: `{"operation":"add","operand_a":1}`, expectedError: "operand_b is required"},
		{name: "unexpected second operand", body: `{"operation":"square_root","operand_a":1,"operand_b":2}`, expectedError: "operand_b is not allowed"},
		{name: "divide by zero", body: `{"operation":"divide","operand_a":1,"operand_b":0}`, expectedError: "divisor cannot be zero"},
		{name: "modulus by zero", body: `{"operation":"modulus","operand_a":1,"operand_b":0}`, expectedError: "divisor cannot be zero"},
		{name: "negative square root", body: `{"operation":"square_root","operand_a":-1}`, expectedError: "cannot compute square root"},
		{name: "negative factorial", body: `{"operation":"factorial","operand_a":-1}`, expectedError: "cannot compute factorial"},
		{name: "fractional factorial", body: `{"operation":"factorial","operand_a":2.5}`, expectedError: "factorial operand must be an integer"},
		{name: "factorial out of range", body: `{"operation":"factorial","operand_a":9223372036854775808}`, expectedError: "factorial operand is out of range"},
		{name: "undefined result", body: `{"operation":"exponentiate","operand_a":-1,"operand_b":0.5}`, expectedError: "Result out of bounds"},
		{name: "non-finite result", body: `{"operation":"exponentiate","operand_a":1e308,"operand_b":2}`, expectedError: "Result out of bounds"},
	}
	c.Len(tests, 11)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := require.New(t)
			response := performRequest(NewHandler(), http.MethodPost, "/", test.body)

			c.Equal(http.StatusOK, response.Code)
			body := decodeCalculationResponse(t, response)
			c.Equal("err", body.Result)
			c.Contains(body.Error, test.expectedError)
		})
	}
}

func TestHandleCalculationTimeout(t *testing.T) {
	c := require.New(t)
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(
		`{"operation":"factorial","operand_a":1000000000}`,
	))
	response := httptest.NewRecorder()

	handleCalculationWithTimeout(response, request, time.Millisecond)

	c.Equal(http.StatusOK, response.Code)
	body := decodeCalculationResponse(t, response)
	c.Equal("err", body.Result)
	c.Equal("calculation timed out", body.Error)
}

func TestHandleCalculationCanceledRequest(t *testing.T) {
	c := require.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(
		`{"operation":"factorial","operand_a":1000000}`,
	)).WithContext(ctx)
	response := httptest.NewRecorder()

	handleCalculation(response, request)

	c.Equal(http.StatusOK, response.Code)
	body := decodeCalculationResponse(t, response)
	c.Equal("err", body.Result)
	c.Equal(context.Canceled.Error(), body.Error)
}

func TestOptionalNumberUnmarshal(t *testing.T) {
	c := require.New(t)
	var number optionalNumber

	c.NoError(json.Unmarshal([]byte(`12.5`), &number))
	c.True(number.Present)
	c.Equal(12.5, number.Value)

	err := json.Unmarshal([]byte(`"invalid"`), &number)
	c.Error(err)
	c.EqualError(err, "operand must be a number")
}

func TestCalculateRejectsUnknownOperation(t *testing.T) {
	c := require.New(t)

	_, err := calculate(context.Background(), calculationRequest{Operation: "invalid"})
	c.ErrorIs(err, errInvalidOperation)
}

func TestCalculateReturnsContextCanceledAfterOperation(t *testing.T) {
	c := require.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const operationName = "test_context_cancel"
	previousOperation, hadPreviousOperation := AllowedOperations[operationName]
	AllowedOperations[operationName] = operationFunc{
		metadata: OperationMetadata{Name: operationName, Arity: "unary"},
		calculate: func(context.Context, float64, *float64) (float64, error) {
			cancel()
			return 1, nil
		},
	}
	defer func() {
		if hadPreviousOperation {
			AllowedOperations[operationName] = previousOperation
		} else {
			delete(AllowedOperations, operationName)
		}
	}()

	_, err := calculate(ctx, calculationRequest{
		Operation: operationName,
		OperandA:  optionalNumber{Present: true, Value: 1},
	})
	c.ErrorIs(err, context.Canceled)
}

func BenchmarkServer(b *testing.B) {
	handler := NewHandler()
	b.Run("catalog", func(b *testing.B) {
		c := require.New(b)
		b.ReportAllocs()
		status := 0
		for i := 0; i < b.N; i++ {
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			status = response.Code
		}
		b.StopTimer()
		c.Equal(http.StatusOK, status)
	})
	b.Run("calculation", func(b *testing.B) {
		c := require.New(b)
		body := `{"operation":"add","operand_a":10,"operand_b":6}`
		b.ReportAllocs()
		status := 0
		for i := 0; i < b.N; i++ {
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			status = response.Code
		}
		b.StopTimer()
		c.Equal(http.StatusOK, status)
	})
}

type testCalculationResponse struct {
	Result any    `json:"result"`
	Error  string `json:"error"`
}

func performRequest(handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func decodeCalculationResponse(t *testing.T, response *httptest.ResponseRecorder) testCalculationResponse {
	t.Helper()
	c := require.New(t)
	var body testCalculationResponse
	c.NoError(json.NewDecoder(response.Body).Decode(&body))
	return body
}
