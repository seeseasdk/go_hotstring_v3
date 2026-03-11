package models

import "fmt"

type Injection struct {
	direction       string
	site            string
	code            string
	eswtFocus       string
	eswtRadial      string
	sonoStim        string
	isWithCarm      bool
	isP             bool
	isN             bool
	isFromClipboard bool
	etc             any
}

func NewInjection(direction, site, code, eswtFocus, eswtRadial, sonoStim string, isWithCarm, isP, isN bool, etc any) *Injection {
	return &Injection{
		direction:       direction,
		site:            site,
		code:            code,
		eswtFocus:       eswtFocus,
		eswtRadial:      eswtRadial,
		sonoStim:        sonoStim,
		isWithCarm:      isWithCarm,
		isP:             isP,
		isN:             isN,
		isFromClipboard: false,
		etc:             etc,
	}
}

func (i *Injection) GetDirection() string {
	return i.direction
}
func (i *Injection) GetSite() string {
	return i.site
}
func (i *Injection) GetCode() string {
	return i.code
}
func (i *Injection) GetEswtFocus() string {
	return i.eswtFocus
}
func (i *Injection) GetEswtRadial() string {
	return i.eswtRadial
}
func (i *Injection) GetSonoStim() string {
	return i.sonoStim
}
func (i *Injection) GetIsWithCarm() bool {
	return i.isWithCarm
}
func (i *Injection) GetIsP() bool {
	return i.isP
}
func (i *Injection) GetIsN() bool {
	return i.isN
}
func (i *Injection) GetIsFromClipboard() bool {
	return i.isFromClipboard
}
func (i *Injection) GetEtc() any {
	return i.etc
}
func (i *Injection) SetDirection(direction string) {
	i.direction = direction
}
func (i *Injection) SetSite(site string) {
	i.site = site
}
func (i *Injection) SetCode(code string) {
	i.code = code
}
func (i *Injection) SetEswtFocus(eswtFocus string) {
	i.eswtFocus = eswtFocus
}
func (i *Injection) SetEswtRadial(eswtRadial string) {
	i.eswtRadial = eswtRadial
}
func (i *Injection) SetSonoStim(sonoStim string) {
	i.sonoStim = sonoStim
}
func (i *Injection) SetIsWithCarm(isWithCarm bool) {
	i.isWithCarm = isWithCarm
}
func (i *Injection) SetIsP(isP bool) {
	i.isP = isP
}
func (i *Injection) SetIsN(isN bool) {
	i.isN = isN
}
func (i *Injection) SetIsFromClipboard(isFromClipboard bool) {
	i.isFromClipboard = isFromClipboard
}
func (i *Injection) SetEtc(etc any) {
	i.etc = etc
}
func (i *Injection) ToString() string {
	return "Direction: " + i.direction + " Site: " + i.site + " Code: " + i.code +
		" EswtFocus: " + i.eswtFocus + " EswtRadial: " + i.eswtRadial + " SonoStim: " + i.sonoStim +
		" IsWithCarm: " + fmt.Sprint(i.isWithCarm) + " IsP: " + fmt.Sprint(i.isP) + " IsN: " + fmt.Sprint(i.isN) +
		" Etc: " + fmt.Sprint(i.etc)
}
func (i *Injection) IsEmpty() bool {
	if i.direction == "" && i.site == "" && i.code == "" && i.eswtFocus == "" && i.eswtRadial == "" && i.sonoStim == "" &&
		!i.isWithCarm && !i.isP && !i.isN && i.etc == nil {
		return true
	}
	return false
}
