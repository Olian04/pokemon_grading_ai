package imageproc

import (
	"image"
	"image/color"
	"math"
)

// homographyFrom4Points computes a 3x3 homography H (h33=1) mapping homogeneous src to dst:
// λ * dst = H * src. Returns ok=false if the system is singular.
func homographyFrom4Points(src, dst []Pt) (H [3][3]float64, ok bool) {
	if len(src) != 4 || len(dst) != 4 {
		return H, false
	}
	// Unknowns x = [h11 h12 h13 h21 h22 h23 h31 h32]^T with h33=1
	// xd = (h11*xs + h12*ys + h13) / (h31*xs + h32*ys + 1)
	// (h31*xs + h32*ys + 1)*xd = h11*xs + h12*ys + h13
	// h11*xs + h12*ys + h13 - h31*xs*xd - h32*ys*xd = xd
	var A [8][8]float64
	var b [8]float64
	for i := 0; i < 4; i++ {
		xs, ys := src[i].X, src[i].Y
		xd, yd := dst[i].X, dst[i].Y
		A[2*i][0] = xs
		A[2*i][1] = ys
		A[2*i][2] = 1
		A[2*i][3] = 0
		A[2*i][4] = 0
		A[2*i][5] = 0
		A[2*i][6] = -xs * xd
		A[2*i][7] = -ys * xd
		b[2*i] = xd

		A[2*i+1][0] = 0
		A[2*i+1][1] = 0
		A[2*i+1][2] = 0
		A[2*i+1][3] = xs
		A[2*i+1][4] = ys
		A[2*i+1][5] = 1
		A[2*i+1][6] = -xs * yd
		A[2*i+1][7] = -ys * yd
		b[2*i+1] = yd
	}
	x, ok := solve8x8(A, b)
	if !ok {
		return H, false
	}
	H[0][0], H[0][1], H[0][2] = x[0], x[1], x[2]
	H[1][0], H[1][1], H[1][2] = x[3], x[4], x[5]
	H[2][0], H[2][1], H[2][2] = x[6], x[7], 1
	return H, true
}

func solve8x8(A [8][8]float64, b [8]float64) (x [8]float64, ok bool) {
	// Gaussian elimination with partial pivoting
	var M [8][9]float64
	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			M[i][j] = A[i][j]
		}
		M[i][8] = b[i]
	}
	for col := 0; col < 8; col++ {
		// pivot
		piv := col
		maxv := math.Abs(M[col][col])
		for r := col + 1; r < 8; r++ {
			if v := math.Abs(M[r][col]); v > maxv {
				maxv = v
				piv = r
			}
		}
		if maxv < 1e-12 {
			return x, false
		}
		if piv != col {
			M[col], M[piv] = M[piv], M[col]
		}
		// normalize row
		div := M[col][col]
		for j := col; j < 9; j++ {
			M[col][j] /= div
		}
		for r := 0; r < 8; r++ {
			if r == col {
				continue
			}
			f := M[r][col]
			if f == 0 {
				continue
			}
			for j := col; j < 9; j++ {
				M[r][j] -= f * M[col][j]
			}
		}
	}
	for i := 0; i < 8; i++ {
		x[i] = M[i][8]
	}
	return x, true
}

func invert3x3(m [3][3]float64) ([3][3]float64, bool) {
	var aug [3][6]float64
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			aug[i][j] = m[i][j]
		}
		aug[i][3+i] = 1
	}
	for col := 0; col < 3; col++ {
		piv := col
		maxv := math.Abs(aug[col][col])
		for r := col + 1; r < 3; r++ {
			if v := math.Abs(aug[r][col]); v > maxv {
				maxv = v
				piv = r
			}
		}
		if maxv < 1e-12 {
			return [3][3]float64{}, false
		}
		if piv != col {
			aug[col], aug[piv] = aug[piv], aug[col]
		}
		div := aug[col][col]
		for j := 0; j < 6; j++ {
			aug[col][j] /= div
		}
		for r := 0; r < 3; r++ {
			if r == col {
				continue
			}
			f := aug[r][col]
			if f == 0 {
				continue
			}
			for j := 0; j < 6; j++ {
				aug[r][j] -= f * aug[col][j]
			}
		}
	}
	var inv [3][3]float64
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			inv[i][j] = aug[i][3+j]
		}
	}
	return inv, true
}

// warpPerspectiveInverse maps each destination pixel through invH to sample src (bilinear).
func warpPerspectiveInverse(src *image.RGBA, invH [3][3]float64, outW, outH int) *image.RGBA {
	b := src.Bounds()
	out := image.NewRGBA(image.Rect(0, 0, outW, outH))
	for y := 0; y < outH; y++ {
		for x := 0; x < outW; x++ {
			// homogeneous dst
			X := float64(x)
			Y := float64(y)
			Z := 1.0
			sx := invH[0][0]*X + invH[0][1]*Y + invH[0][2]*Z
			sy := invH[1][0]*X + invH[1][1]*Y + invH[1][2]*Z
			sw := invH[2][0]*X + invH[2][1]*Y + invH[2][2]*Z
			if math.Abs(sw) < 1e-12 {
				continue
			}
			px := sx / sw
			py := sy / sw
			c := bilinearRGBA(src, b, px, py)
			out.SetRGBA(x, y, c)
		}
	}
	return out
}

func bilinearRGBA(src *image.RGBA, b image.Rectangle, px, py float64) color.RGBA {
	if px < float64(b.Min.X) || py < float64(b.Min.Y) || px >= float64(b.Max.X)-1 || py >= float64(b.Max.Y)-1 {
		return color.RGBA{}
	}
	x0 := int(math.Floor(px - float64(b.Min.X)))
	y0 := int(math.Floor(py - float64(b.Min.Y)))
	tx := px - float64(b.Min.X) - float64(x0)
	ty := py - float64(b.Min.Y) - float64(y0)
	x0 += b.Min.X
	y0 += b.Min.Y
	c00 := src.RGBAAt(x0, y0)
	c10 := src.RGBAAt(x0+1, y0)
	c01 := src.RGBAAt(x0, y0+1)
	c11 := src.RGBAAt(x0+1, y0+1)
	r := (1-tx)*(1-ty)*float64(c00.R) + tx*(1-ty)*float64(c10.R) + (1-tx)*ty*float64(c01.R) + tx*ty*float64(c11.R)
	g := (1-tx)*(1-ty)*float64(c00.G) + tx*(1-ty)*float64(c10.G) + (1-tx)*ty*float64(c01.G) + tx*ty*float64(c11.G)
	bb := (1-tx)*(1-ty)*float64(c00.B) + tx*(1-ty)*float64(c10.B) + (1-tx)*ty*float64(c01.B) + tx*ty*float64(c11.B)
	a := (1-tx)*(1-ty)*float64(c00.A) + tx*(1-ty)*float64(c10.A) + (1-tx)*ty*float64(c01.A) + tx*ty*float64(c11.A)
	return color.RGBA{
		R: uint8(clampInt(int(math.Round(r)), 0, 255)),
		G: uint8(clampInt(int(math.Round(g)), 0, 255)),
		B: uint8(clampInt(int(math.Round(bb)), 0, 255)),
		A: uint8(clampInt(int(math.Round(a)), 0, 255)),
	}
}
