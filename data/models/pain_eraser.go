package models

type PainEraser struct {
	direction string
	site      string
}

func NewPainEraser(direction, site string) *PainEraser {
	return &PainEraser{
		direction: direction,
		site:      site,
	}
}

func (p *PainEraser) SetDirection(direction string) {
	p.direction = direction
}

func (p *PainEraser) SetSite(site string) {
	p.site = site
}

func (p *PainEraser) SetReset() {
	p.direction = ""
	p.site = ""
}

func (p PainEraser) GetDirection() string {
	return p.direction
}

func (p PainEraser) GetSite() string {
	return p.site
}

func (p PainEraser) IsEmpty() bool {
	return p.direction == "" && p.site == ""
}

func (p PainEraser) ToString() string {
	return "Direction: " + p.direction + " Site: " + p.site
}
