package vm

type KeyboardProvider interface {
	GetKey() uint8
	IsKeyPressed(u uint8) bool
}

type DisplayProvider interface {
	Clear()
	Draw([64][32]bool)
}
