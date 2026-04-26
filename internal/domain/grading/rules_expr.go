package grading

import (
	"fmt"
	"strconv"
	"strings"
)

type expression struct {
	Operator string
	Value    float64
}

func parseExpression(raw string) (expression, error) {
	parts := strings.Fields(strings.TrimSpace(raw))
	if len(parts) != 2 {
		return expression{}, fmt.Errorf("invalid expression %q", raw)
	}
	if !isSupportedOperator(parts[0]) {
		return expression{}, fmt.Errorf("unsupported operator %q", parts[0])
	}
	value, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return expression{}, fmt.Errorf("invalid numeric value %q", parts[1])
	}
	return expression{
		Operator: parts[0],
		Value:    value,
	}, nil
}

func ValidateExpression(raw string) error {
	_, err := parseExpression(raw)
	return err
}

func evaluateExpression(raw string, left float64) (bool, error) {
	expr, err := parseExpression(raw)
	if err != nil {
		return false, err
	}
	switch expr.Operator {
	case "<":
		return left < expr.Value, nil
	case "<=":
		return left <= expr.Value, nil
	case ">":
		return left > expr.Value, nil
	case ">=":
		return left >= expr.Value, nil
	default:
		return false, fmt.Errorf("unsupported operator %q", expr.Operator)
	}
}

func isSupportedOperator(op string) bool {
	switch op {
	case "<", "<=", ">", ">=":
		return true
	default:
		return false
	}
}
