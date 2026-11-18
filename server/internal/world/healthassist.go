package world

const detailHealthAssist = 87 // buff: SP and/or MP take the damage before HP

// healthAssistLevel is CUser +0x1338: the level of the active Health
// Assistant (AddApplySkillBuff case 0x57), 0 without it.
func (s *Session) healthAssistLevel() uint8 {
	for _, b := range s.buffs {
		if b.Effect == typeEffectBuff && b.Detail == detailHealthAssist {
			return b.SkillLevel
		}
	}
	return 0
}

// absorbHealthAssist is the Health Assistant part of CUser::Damage
// (0x0044c259): level 1 spends SP against the hit, level 2 MP, level 3 SP and
// then MP, point for point; what is left is the HP damage. Other levels do
// nothing. The amounts really taken are left in s.assisted (HP, SP, MP) so the
// result packets can show them.
func (s *Session) absorbHealthAssist(hp uint32) uint32 {
	s.assisted = [3]uint32{hp, 0, 0}
	level := s.healthAssistLevel()
	if hp == 0 || level < 1 || level > 3 {
		return hp
	}
	st := s.stats()
	rest, sp, mp := hp, uint32(0), uint32(0)
	if level == 1 || level == 3 {
		sp = min(rest, st.CurrentSP)
		rest -= sp
	}
	if level == 2 || level == 3 {
		mp = min(rest, st.CurrentMP)
		rest -= mp
	}
	if sp > 0 || mp > 0 {
		s.spendDelta(mp, sp)
	}
	s.assisted = [3]uint32{rest, sp, mp}
	return rest
}
