package nativemapset

// this type is for benchmark comparison only
type NativeMapSet[T comparable] map[T]struct{}

func EmptyNativeMapSetWithCapacity[T comparable](size uint32) *NativeMapSet[T] {
	result := make(NativeMapSet[T], size)
	return &result
}

func (thisSet *NativeMapSet[T]) Add(val T) {
	(*thisSet)[val] = struct{}{}
}

func (thisSet *NativeMapSet[T]) Contains(val T) bool {
	_, b := (*thisSet)[val]
	return b
}

func (thisSet *NativeMapSet[T]) Count() uint32 {
	return uint32(len(*thisSet)) //nolint:gosec
}

func (thisSet *NativeMapSet[T]) Clear() {
	for k := range *thisSet {
		delete(*thisSet, k)
	}
}
