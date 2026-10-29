package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"sort"
	"strconv"
	"time"
)

const calculationTimeout = 10 * time.Second

type calculationRequest struct {
	Operation string         `json:"operation"`
	OperandA  optionalNumber `json:"operand_a"`
	OperandB  optionalNumber `json:"operand_b"`
}

type optionalNumber struct {
	Present bool
	Value   float64
}

func (n *optionalNumber) UnmarshalJSON(data []byte) error {
	n.Present = true

	value, err := strconv.ParseFloat(string(data), 64)
	if err != nil {
		return errors.New("operand must be a number")
	}

	n.Value = value

	return nil
}

type calculationResponse struct {
	Result any    `json:"result"`
	Error  string `json:"error"`
}

func NewHandler() http.Handler {
	return http.HandlerFunc(handleRequest)
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		operations := make([]OperationMetadata, 0, len(AllowedOperations))

		for _, operation := range AllowedOperations {
			operations = append(operations, operation.Metadata())
		}

		sort.Slice(operations, func(i, j int) bool {
			return operations[i].Name < operations[j].Name
		})
		writeJSON(w, http.StatusOK, map[string][]OperationMetadata{"operations": operations})
	case http.MethodPost:
		handleCalculation(w, r)
	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleCalculation(w http.ResponseWriter, r *http.Request) {
	handleCalculationWithTimeout(w, r, calculationTimeout)
}

func handleCalculationWithTimeout(w http.ResponseWriter, r *http.Request, timeout time.Duration) {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var request calculationRequest
	if err := decoder.Decode(&request); err != nil {
		writeRequestError(w, "Invalid request body")
		return
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeRequestError(w, "Invalid request body")
		return
	}

	if !request.OperandA.Present {
		writeRequestError(w, "operand_a is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	result, err := calculate(ctx, request)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			writeCalculationError(w, "calculation timed out")
			return
		}
		writeCalculationError(w, err.Error())
		return
	}

	if math.IsNaN(result) || math.IsInf(result, 0) {
		writeCalculationError(w, "Result out of bounds")
		return
	}

	writeJSON(w, http.StatusOK, calculationResponse{Result: result, Error: ""})
}

func calculate(ctx context.Context, request calculationRequest) (float64, error) {
	operation, ok := AllowedOperations[request.Operation]
	if !ok {
		return 0, errInvalidOperation
	}

	var operandB *float64
	if request.OperandB.Present {
		operandB = &request.OperandB.Value
	}

	result, err := operation.Calculate(ctx, request.OperandA.Value, operandB)
	if err != nil {
		return 0, err
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	return result, nil
}

func writeRequestError(w http.ResponseWriter, message string) {
	writeJSON(w, http.StatusBadRequest, calculationResponse{Result: "err", Error: message})
}

func writeCalculationError(w http.ResponseWriter, message string) {
	writeJSON(w, http.StatusOK, calculationResponse{Result: "err", Error: message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
