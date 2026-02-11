package models

type SonoStim struct {
	direction string
	site      string
}

func NewSonoStim(direction, site string) *SonoStim {
	return &SonoStim{
		direction: direction,
		site:      site,
	}
}

func (s *SonoStim) SetDirection(direction string) {
	s.direction = direction
}

func (s *SonoStim) SetSite(site string) {
	s.site = site
}

func (s *SonoStim) SetReset() {
	s.direction = ""
	s.site = ""
}

func (s SonoStim) GetDirection() string {
	return s.direction
}

func (s SonoStim) GetSite() string {
	return s.site
}

func (s SonoStim) IsEmpty() bool {
	return s.direction == "" && s.site == ""
}

func (s SonoStim) ToString() string {
	return "Direction: " + s.direction + " Site: " + s.site
}
