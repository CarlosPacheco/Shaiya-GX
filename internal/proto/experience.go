package proto

// ExperienceGain is the native 0x0207 base amount and separate Perfect Party
// bonus. Both contribute to the character's experience total.
type ExperienceGain struct{ Base, Perfect uint32 }

func DecodeExperienceGain(p []byte) (ExperienceGain, error) {
	r := NewReader(p)
	gain := ExperienceGain{Base: r.U32(), Perfect: r.U32()}
	return gain, r.Err()
}
