package types

func ClampF64(Value, Min, Max float64) float64 {

	if Value >= Max {
		return Max
	}

	if Value <= Min {
		return Min
	}

	return Value

}
