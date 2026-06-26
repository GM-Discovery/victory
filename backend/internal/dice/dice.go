package dice

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"
)

const (
	MaxExpressionLength = 64
	MinDiceCount        = 1
	MaxBaseDiceCount    = 100
	MinSides            = 2
	MaxSides            = 1000000
	MaxAbsModifier      = 1000000
	MaxAtomicThrows     = 1000
	RollVersion         = 1
)

var expressionPattern = regexp.MustCompile(`^\s*(\d*)\s*[dD]\s*(\d+)\s*(!?)\s*(?:([+-])\s*(\d+))?\s*$`)

type Spec struct {
	Count        int  `json:"count"`
	Sides        int  `json:"sides"`
	ExplodeOnMax bool `json:"explode_on_max"`
	Modifier     int  `json:"modifier"`
}

type DieResult struct {
	Index    int   `json:"index"`
	Chain    []int `json:"chain"`
	Subtotal int   `json:"subtotal"`
}

type Result struct {
	Expression     string      `json:"expression"`
	Spec           Spec        `json:"spec"`
	Dice           []DieResult `json:"dice"`
	ExplosionCount int         `json:"explosion_count"`
	Modifier       int         `json:"modifier"`
	Total          int         `json:"total"`
	RollVersion    int         `json:"roll_version"`
}

type RandomSource interface {
	Intn(maxExclusive int) (int, error)
}

type CryptoSource struct{}

func (CryptoSource) Intn(maxExclusive int) (int, error) {
	if maxExclusive <= 0 {
		return 0, fmt.Errorf("invalid maxExclusive %d", maxExclusive)
	}

	n, err := rand.Int(rand.Reader, big.NewInt(int64(maxExclusive)))
	if err != nil {
		return 0, err
	}

	return int(n.Int64()), nil
}

type parsedExpression struct {
	Spec Spec
}

func ParseExpression(raw string) (Spec, string, error) {
	expression := strings.TrimSpace(raw)
	if expression == "" {
		return Spec{}, "", errors.New("expression_required")
	}
	if len(expression) > MaxExpressionLength {
		return Spec{}, "", errors.New("expression_too_long")
	}

	match := expressionPattern.FindStringSubmatch(expression)
	if match == nil {
		return Spec{}, "", errors.New("invalid_expression")
	}

	count := 1
	if strings.TrimSpace(match[1]) != "" {
		parsedCount, err := strconv.Atoi(match[1])
		if err != nil {
			return Spec{}, "", errors.New("invalid_expression")
		}
		count = parsedCount
	}
	sides, err := strconv.Atoi(match[2])
	if err != nil {
		return Spec{}, "", errors.New("invalid_expression")
	}

	modifier := 0
	if strings.TrimSpace(match[4]) != "" {
		parsedModifier, err := strconv.Atoi(match[5])
		if err != nil {
			return Spec{}, "", errors.New("invalid_expression")
		}
		if match[4] == "-" {
			parsedModifier = -parsedModifier
		}
		modifier = parsedModifier
	}

	spec := Spec{
		Count:        count,
		Sides:        sides,
		ExplodeOnMax: strings.TrimSpace(match[3]) == "!",
		Modifier:     modifier,
	}
	if err := ValidateSpec(spec); err != nil {
		return Spec{}, "", err
	}

	return spec, NormalizeExpression(spec), nil
}

func ValidateSpec(spec Spec) error {
	if spec.Count < MinDiceCount {
		return errors.New("invalid_dice_count")
	}
	if spec.Count > MaxBaseDiceCount {
		return errors.New("dice_count_too_large")
	}
	if spec.Sides < MinSides {
		return errors.New("invalid_sides")
	}
	if spec.Sides > MaxSides {
		return errors.New("sides_too_large")
	}
	if spec.Modifier < -MaxAbsModifier || spec.Modifier > MaxAbsModifier {
		return errors.New("modifier_too_large")
	}
	return nil
}

func NormalizeExpression(spec Spec) string {
	if spec.Count < 1 {
		return ""
	}

	var b strings.Builder
	if spec.Count != 1 {
		b.WriteString(strconv.Itoa(spec.Count))
	}
	b.WriteByte('d')
	b.WriteString(strconv.Itoa(spec.Sides))
	if spec.ExplodeOnMax {
		b.WriteByte('!')
	}
	if spec.Modifier != 0 {
		if spec.Modifier > 0 {
			b.WriteByte('+')
			b.WriteString(strconv.Itoa(spec.Modifier))
		} else {
			b.WriteByte('-')
			b.WriteString(strconv.Itoa(-spec.Modifier))
		}
	}
	return b.String()
}

func Roll(ctx context.Context, spec Spec, source RandomSource) (Result, error) {
	if err := ValidateSpec(spec); err != nil {
		return Result{}, err
	}
	if source == nil {
		source = CryptoSource{}
	}
	if ctx == nil {
		ctx = context.Background()
	}

	dice := make([]DieResult, 0, spec.Count)
	explosionCount := 0
	totalThrows := 0
	total := 0

	for index := 0; index < spec.Count; index++ {
		chain := make([]int, 0, 2)
		subtotal := 0

		for {
			if err := ctx.Err(); err != nil {
				return Result{}, err
			}
			if totalThrows >= MaxAtomicThrows {
				return Result{}, errors.New("atomic_roll_cap_exceeded")
			}

			raw, err := source.Intn(spec.Sides)
			if err != nil {
				return Result{}, err
			}
			value := raw + 1
			totalThrows++
			chain = append(chain, value)
			subtotal += value

			if !spec.ExplodeOnMax || value != spec.Sides {
				break
			}
			explosionCount++
		}

		dice = append(dice, DieResult{
			Index:    index,
			Chain:    chain,
			Subtotal: subtotal,
		})
		total += subtotal
	}

	total += spec.Modifier

	return Result{
		Expression:     NormalizeExpression(spec),
		Spec:           spec,
		Dice:           dice,
		ExplosionCount: explosionCount,
		Modifier:       spec.Modifier,
		Total:          total,
		RollVersion:    RollVersion,
	}, nil
}

func RollExpression(ctx context.Context, raw string, source RandomSource) (Result, error) {
	spec, expression, err := ParseExpression(raw)
	if err != nil {
		return Result{}, err
	}

	result, err := Roll(ctx, spec, source)
	if err != nil {
		return Result{}, err
	}
	result.Expression = expression
	return result, nil
}
