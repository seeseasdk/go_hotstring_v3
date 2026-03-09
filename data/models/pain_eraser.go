package models

type PainEraser struct {
	direction string
	site      string
	code      string
}

func NewPainEraser(direction, site, code string) *PainEraser {
	return &PainEraser{
		direction: direction,
		site:      site,
		code:      code,
	}
}

func (p *PainEraser) SetDirection(direction string) {
	p.direction = direction
}

func (p *PainEraser) SetSite(site string) {
	p.site = site
}

func (p *PainEraser) SetCode(code string) {
	p.code = code
}

func (p *PainEraser) SetReset() {
	p.direction = ""
	p.site = ""
	p.code = ""
}

func (p PainEraser) GetDirection() string {
	return p.direction
}

func (p PainEraser) GetSite() string {
	return p.site
}

func (p PainEraser) GetCode() string {
	return p.code
}

func (p PainEraser) IsEmpty() bool {
	return p.direction == "" && p.site == "" && p.code == ""
}

func (p PainEraser) ToString() string {
	return "Direction: " + p.direction + " Site: " + p.site + " Code: " + p.code
}
