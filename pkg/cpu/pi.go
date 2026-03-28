package cpu

import "math/big"

// c3Over24 = 640320³ / 24 — a constant in the Chudnovsky series.
var c3Over24 = func() *big.Int {
	c3 := new(big.Int).Exp(big.NewInt(640320), big.NewInt(3), nil)
	return new(big.Int).Div(c3, big.NewInt(24))
}()

// ComputePi computes π to the given number of decimal digits using the
// Chudnovsky algorithm with binary splitting.  The algorithm converges at
// ~14.18 decimal digits per term and runs in O(n log³ n) time, making it
// suitable for scaling the CPU load by varying the digit count.
func ComputePi(digits int) *big.Float {
	// Number of series terms required (~14.18 decimal digits per term).
	terms := int64(float64(digits)/14.18) + 10
	// Working precision in bits (log2(10) ≈ 3.3219 bits per decimal digit).
	prec := uint(float64(digits)*3.3219280948874) + 64

	_, Q, T := binarySplit(0, terms)

	fQ := new(big.Float).SetPrec(prec).SetInt(Q)
	fT := new(big.Float).SetPrec(prec).SetInt(T)

	// π = 426880 · √10005 · Q / T
	sqrt10005 := new(big.Float).SetPrec(prec).SetInt64(10005)
	sqrt10005.Sqrt(sqrt10005)
	num := new(big.Float).SetPrec(prec).SetInt64(426880)
	num.Mul(num, sqrt10005)
	num.Mul(num, fQ)

	return new(big.Float).SetPrec(prec).Quo(num, fT)
}

// binarySplit computes the P, Q, T integer triples for the Chudnovsky series
// over the half-open interval [a, b) by recursive binary splitting.
//
// The recurrence is:
//
//	P(a,a+1) = (6a−5)(2a−1)(6a−1)          [1 for a=0]
//	Q(a,a+1) = a³ · c3Over24                [1 for a=0]
//	T(a,a+1) = P(a,a+1) · (13591409 + 545140134·a) · (−1)^a
//
//	P(a,b) = P(a,m) · P(m,b)
//	Q(a,b) = Q(a,m) · Q(m,b)
//	T(a,b) = T(a,m) · Q(m,b) + P(a,m) · T(m,b)
func binarySplit(a, b int64) (P, Q, T *big.Int) {
	if b-a == 1 {
		if a == 0 {
			P = big.NewInt(1)
			Q = big.NewInt(1)
		} else {
			// P(a) = (6a−5)(2a−1)(6a−1)
			P = new(big.Int).Mul(
				new(big.Int).Mul(big.NewInt(6*a-5), big.NewInt(2*a-1)),
				big.NewInt(6*a-1),
			)
			// Q(a) = a³ · c3Over24
			a3 := new(big.Int).Mul(
				new(big.Int).Mul(big.NewInt(a), big.NewInt(a)),
				big.NewInt(a),
			)
			Q = new(big.Int).Mul(a3, c3Over24)
		}
		// T(a) = P(a) · (13591409 + 545140134·a) · (−1)^a
		coeff := new(big.Int).Add(
			big.NewInt(13591409),
			new(big.Int).Mul(big.NewInt(545140134), big.NewInt(a)),
		)
		T = new(big.Int).Mul(P, coeff)
		if a%2 != 0 {
			T.Neg(T)
		}
		return
	}

	m := (a + b) / 2
	Pl, Ql, Tl := binarySplit(a, m)
	Pr, Qr, Tr := binarySplit(m, b)

	P = new(big.Int).Mul(Pl, Pr)
	Q = new(big.Int).Mul(Ql, Qr)
	T = new(big.Int).Add(
		new(big.Int).Mul(Tl, Qr),
		new(big.Int).Mul(Pl, Tr),
	)
	return
}
