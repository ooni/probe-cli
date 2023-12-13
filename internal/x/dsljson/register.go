package dsljson

import "fmt"

func registerMakeOutput[T any](lx *loader, name string) (chan T, error) {
	if _, found := lx.registers[name]; found || lx.gone[name] {
		return nil, fmt.Errorf("register already exists: %s", name)
	}
	c := make(chan T)
	lx.registers[name] = c
	return c, nil
}

func registerPopInputRaw(lx *loader, name string) (any, error) {
	rawch, found := lx.registers[name]
	if !found {
		if lx.gone[name] {
			return nil, fmt.Errorf("register has already been used: %s", name)
		}
		return nil, fmt.Errorf("register does not exist: %s", name)
	}
	lx.gone[name] = true
	delete(lx.registers, name)
	return rawch, nil
}

func registerPopInput[T any](lx *loader, name string) (chan T, error) {
	rawch, err := registerPopInputRaw(lx, name)
	if err != nil {
		return nil, err
	}
	ch, okay := rawch.(chan T)
	if !okay {
		return nil, fmt.Errorf("invalid type for register: %s", name)
	}
	return ch, nil
}
