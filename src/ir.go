package main

type innerRing struct {
	keys [][]byte
}

func (x *innerRing) InnerRingKeys() ([][]byte, error) {
	return x.keys, nil
}
