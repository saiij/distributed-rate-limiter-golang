package limiter

type Algorithm int

const (
	FixedWindowAlgorithm Algorithm = iota
	SlidingWind
)
