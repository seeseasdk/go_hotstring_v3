package models

type SimpleInput struct {
	text    string
	extraDo string
}

func NewSimpleInput(text string, extraDo string) *SimpleInput {
	return &SimpleInput{
		text:    text,
		extraDo: extraDo,
	}
}
func (s *SimpleInput) GetText() string {
	return s.text
}
func (s *SimpleInput) GetExtraDo() string {
	return s.extraDo
}
func (s *SimpleInput) SetText(text string) {
	s.text = text
}
func (s *SimpleInput) SetExtraDo(extraDo string) {
	s.extraDo = extraDo
}
func (s *SimpleInput) IsEmpty() bool {
	if s.text == "" && s.extraDo == "" {
		return true
	}
	return false
}
func (s *SimpleInput) SetReset() {
	s.text = ""
	s.extraDo = ""
}
func (s *SimpleInput) ToString() string {
	return "text: " + s.text + "\n" +
		"extraDo: " + s.extraDo + "\n"
}
