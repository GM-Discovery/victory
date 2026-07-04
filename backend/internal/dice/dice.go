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
	MaxDiceGroups       = 10
	MinSides            = 2
	MaxSides            = 1000000
	MaxAbsModifier      = 1000000
	MaxAtomicThrows     = 1000
	RollVersion         = 1
)

var dieTermPattern = regexp.MustCompile(`^(\d*)[dD](\d+)(!?)$`)

// DieGroup is one NdS[!] term in an expression. Compound pools like
// "2d20+d12!" carry several groups; each group explodes independently on its
// own max face.
type DieGroup struct {
	Count        int  `json:"count"`
	Sides        int  `json:"sides"`
	ExplodeOnMax bool `json:"explode_on_max"`
}

type Spec struct {
	Groups   []DieGroup `json:"groups"`
	Modifier int        `json:"modifier"`
}

type DieResult struct {
	Index    int   `json:"index"`
	Sides    int   `json:"sides"`
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

// SingleGroup builds a one-group Spec, the common case for tray-driven rolls.
func SingleGroup(count, sides int, explodeOnMax bool, modifier int) Spec {
	return Spec{
		Groups:   []DieGroup{{Count: count, Sides: sides, ExplodeOnMax: explodeOnMax}},
		Modifier: modifier,
	}
}

// ParseExpression accepts compound pools: one or more NdS[!] terms joined by
// "+", plus optional signed integer constants that sum into the modifier.
// Examples: "d20", "2d6+3", "d10!", "2d20+d12", "d10+d4!-1".
func ParseExpression(raw string) (Spec, string, error) {
	expression := strings.TrimSpace(raw)
	if expression == "" {
		return Spec{}, "", errors.New("expression_required")
	}
	if len(expression) > MaxExpressionLength {
		return Spec{}, "", errors.New("expression_too_long")
	}

	compact := strings.Map(func(r rune) rune {
		if r == ' ' || r == '\t' {
			return -1
		}
		return r
	}, expression)
	if compact == "" {
		return Spec{}, "", errors.New("expression_required")
	}

	type term struct {
		sign int
		body string
	}
	terms := []term{}
	var current strings.Builder
	sign := 1
	for i, r := range compact {
		if r == '+' || r == '-' {
			if i == 0 || current.Len() == 0 {
				return Spec{}, "", errors.New("invalid_expression")
			}
			terms = append(terms, term{sign: sign, body: current.String()})
			current.Reset()
			sign = 1
			if r == '-' {
				sign = -1
			}
			continue
		}
		current.WriteRune(r)
	}
	if current.Len() == 0 {
		return Spec{}, "", errors.New("invalid_expression")
	}
	terms = append(terms, term{sign: sign, body: current.String()})

	spec := Spec{}
	for _, t := range terms {
		if match := dieTermPattern.FindStringSubmatch(t.body); match != nil {
			if t.sign < 0 {
				// Subtracting a die group is not a supported operation.
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
			spec.Groups = append(spec.Groups, DieGroup{
				Count:        count,
				Sides:        sides,
				ExplodeOnMax: match[3] == "!",
			})
			continue
		}

		value, err := strconv.Atoi(t.body)
		if err != nil {
			return Spec{}, "", errors.New("invalid_expression")
		}
		if value > MaxAbsModifier {
			return Spec{}, "", errors.New("modifier_too_large")
		}
		spec.Modifier += t.sign * value
	}

	if err := ValidateSpec(spec); err != nil {
		return Spec{}, "", err
	}

	return spec, NormalizeExpression(spec), nil
}

func ValidateSpec(spec Spec) error {
	if len(spec.Groups) == 0 {
		return errors.New("invalid_expression")
	}
	if len(spec.Groups) > MaxDiceGroups {
		return errors.New("dice_count_too_large")
	}
	totalDice := 0
	for _, group := range spec.Groups {
		if group.Count < MinDiceCount {
			return errors.New("invalid_dice_count")
		}
		if group.Sides < MinSides {
			return errors.New("invalid_sides")
		}
		if group.Sides > MaxSides {
			return errors.New("sides_too_large")
		}
		totalDice += group.Count
	}
	if totalDice > MaxBaseDiceCount {
		return errors.New("dice_count_too_large")
	}
	if spec.Modifier < -MaxAbsModifier || spec.Modifier > MaxAbsModifier {
		return errors.New("modifier_too_large")
	}
	return nil
}

func NormalizeExpression(spec Spec) string {
	if len(spec.Groups) == 0 {
		return ""
	}

	var b strings.Builder
	for i, group := range spec.Groups {
		if i > 0 {
			b.WriteByte('+')
		}
		if group.Count != 1 {
			b.WriteString(strconv.Itoa(group.Count))
		}
		b.WriteByte('d')
		b.WriteString(strconv.Itoa(group.Sides))
		if group.ExplodeOnMax {
			b.WriteByte('!')
		}
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

	dice := []DieResult{}
	explosionCount := 0
	totalThrows := 0
	total := 0
	dieIndex := 0

	for _, group := range spec.Groups {
		for i := 0; i < group.Count; i++ {
			chain := make([]int, 0, 2)
			subtotal := 0

			for {
				if err := ctx.Err(); err != nil {
					return Result{}, err
				}
				if totalThrows >= MaxAtomicThrows {
					return Result{}, errors.New("atomic_roll_cap_exceeded")
				}

				raw, err := source.Intn(group.Sides)
				if err != nil {
					return Result{}, err
				}
				value := raw + 1
				totalThrows++
				chain = append(chain, value)
				subtotal += value

				if !group.ExplodeOnMax || value != group.Sides {
					break
				}
				explosionCount++
			}

			dice = append(dice, DieResult{
				Index:    dieIndex,
				Sides:    group.Sides,
				Chain:    chain,
				Subtotal: subtotal,
			})
			total += subtotal
			dieIndex++
		}
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
