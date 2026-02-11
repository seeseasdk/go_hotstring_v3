package models

type Xray struct {
	text string
	code string
}

func NewXray(text string, code string) *Xray {
	return &Xray{
		text: text,
		code: code,
	}
}

func (x *Xray) SetText(text string) {
	x.text = text
}
func (x *Xray) SetCode(code string) {
	x.code = code
}
func (x *Xray) GetText() string {
	return x.text
}
func (x *Xray) GetCode() string {
	return x.code
}
func (x *Xray) SetReset() {
	x.text = ""
	x.code = ""
}
func (x *Xray) ToString() string {
	return "Xray: " + x.text + ", code: " + x.code
}
func (x *Xray) IsEmpty() bool {
	return x.text == "" && x.code == ""
}
