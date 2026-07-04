package sample

func compute(limit int) int {
	total := 0
	for i := 0; i < limit; i++ {
		total += i
	}

	x, y := 1, 2
	_ = x

	done := make(chan struct{})
	go func() {
		_ = y
		close(done)
	}()
	defer func() {
		<-done
	}()

	switch {
	case total > 100:
		total = 100
	default:
	}

	return total
}
