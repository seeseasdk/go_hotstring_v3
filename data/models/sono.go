package models

type Sono struct {
	text string
	code string
}

func NewSono(text string, code string) *Sono {
	return &Sono{
		text: text,
		code: code,
	}
}
func (s *Sono) SetText(text string) {
	s.text = text
}
func (s *Sono) SetCode(code string) {
	s.code = code
}
func (s *Sono) GetText() string {
	return s.text
}
func (s *Sono) GetCode() string {
	return s.code
}
func (s *Sono) SetReset() {
	s.text = ""
	s.code = ""
}
func (s *Sono) ToString() string {
	return "Sono: " + s.text + ", code: " + s.code
}
func (s *Sono) IsEmpty() bool {
	return s.text == "" && s.code == ""
}
func (s *Sono) GetChartText() string {
	if s.text != "" {
		return s.text + " (" + s.code + ")"
	}
	return ""
}
