package models

type SonoStim struct {
	direction string
	site      string
	code      string
}

func NewSonoStim(direction, site, code string) *SonoStim {
	return &SonoStim{
		direction: direction,
		site:      site,
		code:      code,
	}
}

func (s *SonoStim) SetDirection(direction string) {
	s.direction = direction
}

func (s *SonoStim) SetSite(site string) {
	s.site = site
}

func (s *SonoStim) SetCode(code string) {
	s.code = code
}

func (s *SonoStim) SetReset() {
	s.direction = ""
	s.site = ""
	s.code = ""
}

func (s SonoStim) GetDirection() string {
	return s.direction
}

func (s SonoStim) GetSite() string {
	return s.site
}

func (s SonoStim) GetCode() string {
	return s.code
}

func (s SonoStim) IsEmpty() bool {
	return s.direction == "" && s.site == "" && s.code == ""
}

func (s SonoStim) ToString() string {
	return "Direction: " + s.direction + " Site: " + s.site + " Code: " + s.code
}
