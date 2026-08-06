package vm

type KeyboardProvider interface {
	GetPressedKey() (key uint8, pressed bool)
	IsKeyPressed(key uint8) bool
	Close()
}

type DisplayProvider interface {
	Clear()
	Draw([64][32]bool)
	Close()
}

type SoundProvider interface {
	PlaySound()
	StopSound()
}
