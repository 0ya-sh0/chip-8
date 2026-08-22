package vm

type FrameBuffer [64][32]bool

type KeyboardProvider interface {
	GetPressedKey() (key uint8, pressed bool)
	IsKeyPressed(key uint8) bool
	Close()
}

type DisplayProvider interface {
	Clear()
	Draw(FrameBuffer)
	Close()
}

type SoundProvider interface {
	PlaySound()
	StopSound()
}
