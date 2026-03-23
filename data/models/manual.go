package models

import "fmt"

type Manual struct {
	text    string
	code    string
	extraDo any
}

func NewManual(text, code string, extraDo any) *Manual {
	return &Manual{
		text:    text,
		code:    code,
		extraDo: extraDo,
	}
}
func (s *Manual) GetText() string {
	return s.text
}
func (s *Manual) GetCode() string {
	return s.code
}
func (s *Manual) GetExtraDo() any {
	return s.extraDo
}
func (s *Manual) SetText(text string) {
	s.text = text
}
func (s *Manual) SetCode(code string) {
	s.code = code
}
func (s *Manual) SetExtraDo(extraDo any) {
	s.extraDo = extraDo
}
func (s *Manual) IsEmpty() bool {
	if s.text == "" && s.code == "" && s.extraDo == nil {
		return true
	}
	return false
}
func (s *Manual) SetReset() {
	s.text = ""
	s.code = ""
	s.extraDo = nil
}
func (s *Manual) ToString() string {
	return "text: " + s.text + "\n" +
		"code: " + s.code + "\n" +
		"extraDo: " + fmt.Sprintf("%v", s.extraDo) + "\n"
}
