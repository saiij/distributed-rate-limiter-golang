package ratelimiter

type Algorithm int

const (
	FixedWindow Algorithm = iota
	SlidingWindow
	TokenBucket
)
