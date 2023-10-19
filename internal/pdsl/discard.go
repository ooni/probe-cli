package pdsl

// Discard returns a [Filter] that converts a [Result] T to a [Result] [Void].
func Discard[T any]() Filter[T, Void] {
	return startFilterService(func(input T) (Void, error) {
		return Void{}, nil
	})
}
