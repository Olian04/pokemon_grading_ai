package imageproc

// largestComponentMask returns a new mask with only the largest 4-connected foreground (1) component.
// Components that touch all four image borders and cover more than maxBorderTouchRatio of pixels are ignored (full frame).
func largestComponentMask(m [][]uint8, maxBorderTouchRatio float64) [][]uint8 {
	h, w := len(m), len(m[0])
	labels := make([][]int, h)
	for y := 0; y < h; y++ {
		labels[y] = make([]int, w)
	}
	nextLabel := 1
	sizes := make(map[int]int)
	touches := make(map[int][]bool) // L,R,T,B per label

	var flood func(sy, sx, id int)
	flood = func(sy, sx, id int) {
		stack := [][2]int{{sy, sx}}
		for len(stack) > 0 {
			y, x := stack[len(stack)-1][0], stack[len(stack)-1][1]
			stack = stack[:len(stack)-1]
			if y < 0 || y >= h || x < 0 || x >= w {
				continue
			}
			if m[y][x] == 0 || labels[y][x] != 0 {
				continue
			}
			labels[y][x] = id
			sizes[id]++
			if touches[id] == nil {
				touches[id] = make([]bool, 4)
			}
			t := touches[id]
			if x == 0 {
				t[0] = true
			}
			if x == w-1 {
				t[1] = true
			}
			if y == 0 {
				t[2] = true
			}
			if y == h-1 {
				t[3] = true
			}
			touches[id] = t
			stack = append(stack, [][2]int{{y + 1, x}, {y - 1, x}, {y, x + 1}, {y, x - 1}}...)
		}
	}

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if m[y][x] == 1 && labels[y][x] == 0 {
				flood(y, x, nextLabel)
				nextLabel++
			}
		}
	}
	bestID := -1
	bestSize := 0
	total := w * h
	for id := 1; id < nextLabel; id++ {
		sz := sizes[id]
		if sz == 0 {
			continue
		}
		tb := touches[id]
		borderTouch := len(tb) == 4 && tb[0] && tb[1] && tb[2] && tb[3]
		ratio := float64(sz) / float64(total)
		if borderTouch && ratio >= maxBorderTouchRatio {
			continue
		}
		if sz > bestSize {
			bestSize = sz
			bestID = id
		}
	}
	out := make([][]uint8, h)
	for y := 0; y < h; y++ {
		out[y] = make([]uint8, w)
	}
	if bestID < 0 {
		return out
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if labels[y][x] == bestID {
				out[y][x] = 1
			}
		}
	}
	return out
}
