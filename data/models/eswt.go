package models

import "fmt"

type ESWT struct {
	direction  string
	focus      string
	radial     string
	feeType    string
	isOnlyEswt bool
}

func NewESWT(direction, focus, radial, feeType string, isOnlyEswt bool) *ESWT {
	return &ESWT{
		direction:  direction,
		focus:      focus,
		radial:     radial,
		feeType:    feeType,
		isOnlyEswt: isOnlyEswt,
	}
}
func (e *ESWT) GetDirection() string {
	return e.direction
}
func (e *ESWT) GetFocus() string {
	return e.focus
}
func (e *ESWT) GetRadial() string {
	return e.radial
}
func (e *ESWT) GetFeeType() string {
	return e.feeType
}
func (e *ESWT) GetIsOnlyEswt() bool {
	return e.isOnlyEswt
}
func (e *ESWT) SetDirection(direction string) {
	e.direction = direction
}
func (e *ESWT) SetFocus(focus string) {
	e.focus = focus
}
func (e *ESWT) SetRadial(radial string) {
	e.radial = radial
}
func (e *ESWT) SetFeeType(feeType string) {
	e.feeType = feeType
}
func (e *ESWT) SetIsOnlyEswt(isOnlyEswt bool) {
	e.isOnlyEswt = isOnlyEswt
}
func (e *ESWT) SetReset() {
	e.direction = ""
	e.focus = ""
	e.radial = ""
	e.feeType = ""
	e.isOnlyEswt = false
}
func (e *ESWT) IsEmpty() bool {
	return e.direction == "" && e.focus == "" && e.radial == "" && e.feeType == "" && !e.isOnlyEswt
}
func (e *ESWT) ToString() string {
	return "Direction: " + e.direction + " Focus: " + e.focus + " Radial: " +
		e.radial + " FeeType: " + e.feeType + " IsOnlyEswt: " + fmt.Sprint(e.isOnlyEswt)
}
