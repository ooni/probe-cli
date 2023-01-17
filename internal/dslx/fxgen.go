package dslx

//
// Functional extensions (auto-generated code)
//

// Compose3 composes N=3 functions.
func Compose3[
	T0 any,
	T1 any,
	T2 any,
	T3 any,
](
	f0 Func[T0, *Maybe[T1]],
	f1 Func[T1, *Maybe[T2]],
	f2 Func[T2, *Maybe[T3]],
) Func[T0, *Maybe[T3]] {
	return Compose2(f0, Compose2(f1, f2))
}

// Compose4 composes N=4 functions.
func Compose4[
	T0 any,
	T1 any,
	T2 any,
	T3 any,
	T4 any,
](
	f0 Func[T0, *Maybe[T1]],
	f1 Func[T1, *Maybe[T2]],
	f2 Func[T2, *Maybe[T3]],
	f3 Func[T3, *Maybe[T4]],
) Func[T0, *Maybe[T4]] {
	return Compose2(f0, Compose3(f1, f2, f3))
}

// Compose5 composes N=5 functions.
func Compose5[
	T0 any,
	T1 any,
	T2 any,
	T3 any,
	T4 any,
	T5 any,
](
	f0 Func[T0, *Maybe[T1]],
	f1 Func[T1, *Maybe[T2]],
	f2 Func[T2, *Maybe[T3]],
	f3 Func[T3, *Maybe[T4]],
	f4 Func[T4, *Maybe[T5]],
) Func[T0, *Maybe[T5]] {
	return Compose2(f0, Compose4(f1, f2, f3, f4))
}

// Compose6 composes N=6 functions.
func Compose6[
	T0 any,
	T1 any,
	T2 any,
	T3 any,
	T4 any,
	T5 any,
	T6 any,
](
	f0 Func[T0, *Maybe[T1]],
	f1 Func[T1, *Maybe[T2]],
	f2 Func[T2, *Maybe[T3]],
	f3 Func[T3, *Maybe[T4]],
	f4 Func[T4, *Maybe[T5]],
	f5 Func[T5, *Maybe[T6]],
) Func[T0, *Maybe[T6]] {
	return Compose2(f0, Compose5(f1, f2, f3, f4, f5))
}

// Compose7 composes N=7 functions.
func Compose7[
	T0 any,
	T1 any,
	T2 any,
	T3 any,
	T4 any,
	T5 any,
	T6 any,
	T7 any,
](
	f0 Func[T0, *Maybe[T1]],
	f1 Func[T1, *Maybe[T2]],
	f2 Func[T2, *Maybe[T3]],
	f3 Func[T3, *Maybe[T4]],
	f4 Func[T4, *Maybe[T5]],
	f5 Func[T5, *Maybe[T6]],
	f6 Func[T6, *Maybe[T7]],
) Func[T0, *Maybe[T7]] {
	return Compose2(f0, Compose6(f1, f2, f3, f4, f5, f6))
}

// Compose8 composes N=8 functions.
func Compose8[
	T0 any,
	T1 any,
	T2 any,
	T3 any,
	T4 any,
	T5 any,
	T6 any,
	T7 any,
	T8 any,
](
	f0 Func[T0, *Maybe[T1]],
	f1 Func[T1, *Maybe[T2]],
	f2 Func[T2, *Maybe[T3]],
	f3 Func[T3, *Maybe[T4]],
	f4 Func[T4, *Maybe[T5]],
	f5 Func[T5, *Maybe[T6]],
	f6 Func[T6, *Maybe[T7]],
	f7 Func[T7, *Maybe[T8]],
) Func[T0, *Maybe[T8]] {
	return Compose2(f0, Compose7(f1, f2, f3, f4, f5, f6, f7))
}

// Compose9 composes N=9 functions.
func Compose9[
	T0 any,
	T1 any,
	T2 any,
	T3 any,
	T4 any,
	T5 any,
	T6 any,
	T7 any,
	T8 any,
	T9 any,
](
	f0 Func[T0, *Maybe[T1]],
	f1 Func[T1, *Maybe[T2]],
	f2 Func[T2, *Maybe[T3]],
	f3 Func[T3, *Maybe[T4]],
	f4 Func[T4, *Maybe[T5]],
	f5 Func[T5, *Maybe[T6]],
	f6 Func[T6, *Maybe[T7]],
	f7 Func[T7, *Maybe[T8]],
	f8 Func[T8, *Maybe[T9]],
) Func[T0, *Maybe[T9]] {
	return Compose2(f0, Compose8(f1, f2, f3, f4, f5, f6, f7, f8))
}

// Compose10 composes N=10 functions.
func Compose10[
	T0 any,
	T1 any,
	T2 any,
	T3 any,
	T4 any,
	T5 any,
	T6 any,
	T7 any,
	T8 any,
	T9 any,
	T10 any,
](
	f0 Func[T0, *Maybe[T1]],
	f1 Func[T1, *Maybe[T2]],
	f2 Func[T2, *Maybe[T3]],
	f3 Func[T3, *Maybe[T4]],
	f4 Func[T4, *Maybe[T5]],
	f5 Func[T5, *Maybe[T6]],
	f6 Func[T6, *Maybe[T7]],
	f7 Func[T7, *Maybe[T8]],
	f8 Func[T8, *Maybe[T9]],
	f9 Func[T9, *Maybe[T10]],
) Func[T0, *Maybe[T10]] {
	return Compose2(f0, Compose9(f1, f2, f3, f4, f5, f6, f7, f8, f9))
}

// Compose11 composes N=11 functions.
func Compose11[
	T0 any,
	T1 any,
	T2 any,
	T3 any,
	T4 any,
	T5 any,
	T6 any,
	T7 any,
	T8 any,
	T9 any,
	T10 any,
	T11 any,
](
	f0 Func[T0, *Maybe[T1]],
	f1 Func[T1, *Maybe[T2]],
	f2 Func[T2, *Maybe[T3]],
	f3 Func[T3, *Maybe[T4]],
	f4 Func[T4, *Maybe[T5]],
	f5 Func[T5, *Maybe[T6]],
	f6 Func[T6, *Maybe[T7]],
	f7 Func[T7, *Maybe[T8]],
	f8 Func[T8, *Maybe[T9]],
	f9 Func[T9, *Maybe[T10]],
	f10 Func[T10, *Maybe[T11]],
) Func[T0, *Maybe[T11]] {
	return Compose2(f0, Compose10(f1, f2, f3, f4, f5, f6, f7, f8, f9, f10))
}

// Compose12 composes N=12 functions.
func Compose12[
	T0 any,
	T1 any,
	T2 any,
	T3 any,
	T4 any,
	T5 any,
	T6 any,
	T7 any,
	T8 any,
	T9 any,
	T10 any,
	T11 any,
	T12 any,
](
	f0 Func[T0, *Maybe[T1]],
	f1 Func[T1, *Maybe[T2]],
	f2 Func[T2, *Maybe[T3]],
	f3 Func[T3, *Maybe[T4]],
	f4 Func[T4, *Maybe[T5]],
	f5 Func[T5, *Maybe[T6]],
	f6 Func[T6, *Maybe[T7]],
	f7 Func[T7, *Maybe[T8]],
	f8 Func[T8, *Maybe[T9]],
	f9 Func[T9, *Maybe[T10]],
	f10 Func[T10, *Maybe[T11]],
	f11 Func[T11, *Maybe[T12]],
) Func[T0, *Maybe[T12]] {
	return Compose2(f0, Compose11(f1, f2, f3, f4, f5, f6, f7, f8, f9, f10, f11))
}

// Compose13 composes N=13 functions.
func Compose13[
	T0 any,
	T1 any,
	T2 any,
	T3 any,
	T4 any,
	T5 any,
	T6 any,
	T7 any,
	T8 any,
	T9 any,
	T10 any,
	T11 any,
	T12 any,
	T13 any,
](
	f0 Func[T0, *Maybe[T1]],
	f1 Func[T1, *Maybe[T2]],
	f2 Func[T2, *Maybe[T3]],
	f3 Func[T3, *Maybe[T4]],
	f4 Func[T4, *Maybe[T5]],
	f5 Func[T5, *Maybe[T6]],
	f6 Func[T6, *Maybe[T7]],
	f7 Func[T7, *Maybe[T8]],
	f8 Func[T8, *Maybe[T9]],
	f9 Func[T9, *Maybe[T10]],
	f10 Func[T10, *Maybe[T11]],
	f11 Func[T11, *Maybe[T12]],
	f12 Func[T12, *Maybe[T13]],
) Func[T0, *Maybe[T13]] {
	return Compose2(f0, Compose12(f1, f2, f3, f4, f5, f6, f7, f8, f9, f10, f11, f12))
}

// Compose14 composes N=14 functions.
func Compose14[
	T0 any,
	T1 any,
	T2 any,
	T3 any,
	T4 any,
	T5 any,
	T6 any,
	T7 any,
	T8 any,
	T9 any,
	T10 any,
	T11 any,
	T12 any,
	T13 any,
	T14 any,
](
	f0 Func[T0, *Maybe[T1]],
	f1 Func[T1, *Maybe[T2]],
	f2 Func[T2, *Maybe[T3]],
	f3 Func[T3, *Maybe[T4]],
	f4 Func[T4, *Maybe[T5]],
	f5 Func[T5, *Maybe[T6]],
	f6 Func[T6, *Maybe[T7]],
	f7 Func[T7, *Maybe[T8]],
	f8 Func[T8, *Maybe[T9]],
	f9 Func[T9, *Maybe[T10]],
	f10 Func[T10, *Maybe[T11]],
	f11 Func[T11, *Maybe[T12]],
	f12 Func[T12, *Maybe[T13]],
	f13 Func[T13, *Maybe[T14]],
) Func[T0, *Maybe[T14]] {
	return Compose2(f0, Compose13(f1, f2, f3, f4, f5, f6, f7, f8, f9, f10, f11, f12, f13))
}
