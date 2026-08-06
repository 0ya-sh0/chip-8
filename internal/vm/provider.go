package vm

type KeyboardProvider interface {
	GetKey() uint8
}

type DisplayProvider interface {
	Clear()
	Draw([64][32]bool)
}
