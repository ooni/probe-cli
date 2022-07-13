package testingx

import (
	"math/rand"
	"reflect"
	"sync"
	"time"
)

// FakeFiller fills specific data structures with random data. The only
// exception to this behaviour is time.Time, which is instead filled
// with the current time plus a small random number of seconds.
//
// We use this implementation to initialize data in our model. The code
// has been written with that in mind. It will require some hammering in
// case we extend the model with new field types.
//
// Caveat: this kind of fillter does not support filling interfaces
// and channels and other complex types. The current behavior when this
// kind of data types is encountered is to just ignore them.
//
// This struct is quite limited in scope and we can fill only the
// structures you typically send over as JSONs.
//
// As part of future work, we aim to investigate whether we can
// replace this implementation with https://go.dev/blog/fuzz-beta.
type FakeFiller struct {
	// mu provides mutual exclusion
	mu sync.Mutex

	// Now is OPTIONAL and allows to mock the current time
	Now func() time.Time

	// rnd is the random number generator and is
	// automatically initialized on first use
	rnd *rand.Rand
}

func (ff *FakeFiller) getRandLocked() *rand.Rand {
	if ff.rnd == nil {
		now := time.Now
		if ff.Now != nil {
			now = ff.Now
		}
		ff.rnd = rand.New(rand.NewSource(now().UnixNano()))
	}
	return ff.rnd
}

func (ff *FakeFiller) getRandomString() string {
	defer ff.mu.Unlock()
	ff.mu.Lock()
	rnd := ff.getRandLocked()
	n := rnd.Intn(63) + 1
	// See https://stackoverflow.com/a/31832326
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rnd.Intn(len(letterRunes))]
	}
	return string(b)
}

func (ff *FakeFiller) getRandomInt64() int64 {
	defer ff.mu.Unlock()
	ff.mu.Lock()
	rnd := ff.getRandLocked()
	return rnd.Int63()
}

func (ff *FakeFiller) getRandomBool() bool {
	defer ff.mu.Unlock()
	ff.mu.Lock()
	rnd := ff.getRandLocked()
	return rnd.Float64() >= 0.5
}

func (ff *FakeFiller) getRandomSmallPositiveInt() int {
	defer ff.mu.Unlock()
	ff.mu.Lock()
	rnd := ff.getRandLocked()
	return int(rnd.Int63n(8)) + 1 // safe cast
}

func (ff *FakeFiller) doFill(v reflect.Value) {
	for v.Type().Kind() == reflect.Ptr {
		if v.IsNil() {
			// if the pointer is nil, allocate an element
			v.Set(reflect.New(v.Type().Elem()))
		}
		// switch to the element
		v = v.Elem()
	}
	switch v.Type().Kind() {
	case reflect.String:
		v.SetString(ff.getRandomString())
	case reflect.Int64:
		v.SetInt(ff.getRandomInt64())
	case reflect.Bool:
		v.SetBool(ff.getRandomBool())
	case reflect.Struct:
		if v.Type().String() == "time.Time" {
			// Implementation note: we treat the time specially
			// and we avoid attempting to set its fields.
			v.Set(reflect.ValueOf(time.Now().Add(
				time.Duration(ff.getRandomSmallPositiveInt()) * time.Second)))
			return
		}
		for idx := 0; idx < v.NumField(); idx++ {
			ff.doFill(v.Field(idx)) // visit all fields
		}
	case reflect.Slice:
		kind := v.Type().Elem()
		total := ff.getRandomSmallPositiveInt()
		for idx := 0; idx < total; idx++ {
			value := reflect.New(kind) // make a new element
			ff.doFill(value)
			v.Set(reflect.Append(v, value.Elem())) // append to slice
		}
	case reflect.Map:
		if v.Type().Key().Kind() != reflect.String {
			panic("fakefill: we only support string key types")
		}
		v.Set(reflect.MakeMap(v.Type())) // we need to init the map
		total := ff.getRandomSmallPositiveInt()
		kind := v.Type().Elem()
		for idx := 0; idx < total; idx++ {
			value := reflect.New(kind)
			ff.doFill(value)
			v.SetMapIndex(reflect.ValueOf(ff.getRandomString()), value.Elem())
		}
	}
}

// Fill fills the input structure or pointer with random data.
func (ff *FakeFiller) Fill(in interface{}) {
	ff.doFill(reflect.ValueOf(in))
}
