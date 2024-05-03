package httpclientx

import "testing"

func TestConfigMaxResponseBodySize(t *testing.T) {
	t.Run("the default returned value corresponds to the constant default", func(t *testing.T) {
		config := &Config{}
		if value := config.maxResponseBodySize(); value != DefaultMaxResponseBodySize {
			t.Fatal("unexpected maxResponseBodySize()", value)
		}
	})

	t.Run("we can override the default", func(t *testing.T) {
		config := &Config{}
		const expectedValue = DefaultMaxResponseBodySize / 2
		config.MaxResponseBodySize = expectedValue
		if value := config.maxResponseBodySize(); value != expectedValue {
			t.Fatal("unexpected maxResponseBodySize()", value)
		}
	})
}
