package models

import (
	"strings"
)

type FirstMeeting struct {
	sites        []string
	duration     string
	xray         []Xray
	physicalExam string // 물리적 검사
	extraExam    []any  // 추가적인 검사
}

func NewFirstMeeting(site []string, pe string, xray []Xray, extraExam []any) *FirstMeeting {
	return &FirstMeeting{
		sites:        site,
		duration:     "",
		xray:         xray,
		physicalExam: pe,
		extraExam:    extraExam,
	}
}
func (fm *FirstMeeting) SetSite(sites []string) {
	fm.sites = sites
}
func (fm *FirstMeeting) SetAddSite(site []string) {
	fm.sites = append(fm.sites, site...)
}
func (fm *FirstMeeting) SetAddphysicalExam(physicalExam string) {
	if fm.physicalExam == "" {
		fm.physicalExam = physicalExam
	} else {
		fm.physicalExam += "\n" + physicalExam
	}
}
func (fm *FirstMeeting) SetDuration(duration string) {
	fm.duration = duration
}
func (fm *FirstMeeting) SetPhysicalExam(physicalExam string) {
	fm.physicalExam = physicalExam
}
func (fm *FirstMeeting) SetXray(xray []Xray) {
	fm.xray = xray
}
func (fm *FirstMeeting) SetExtraExam(extraExam []any) {
	fm.extraExam = extraExam
}
func (fm *FirstMeeting) SetAddExtraExams(extraExam []any) {
	fm.extraExam = append(fm.extraExam, extraExam...)
}
func (fm *FirstMeeting) SetAddExtraExam(extra *any) {
	if extra != nil {
		fm.extraExam = append(fm.extraExam, *extra)
	}
}
func (fm *FirstMeeting) SetAddXray(xray []Xray) {
	fm.xray = append(fm.xray, xray...)
}
func (fm *FirstMeeting) GetDuration() string {
	return fm.duration
}
func (fm *FirstMeeting) GetPhysicalExam() string {
	return fm.physicalExam
}
func (fm *FirstMeeting) GetExtraExam() []any {
	return fm.extraExam
}
func (fm *FirstMeeting) AddSite(site string) {
	fm.sites = append(fm.sites, site)
}
func (fm *FirstMeeting) GetSites() []string {
	return fm.sites
}
func (fm *FirstMeeting) GetXray() []Xray {
	return fm.xray
}
func (fm *FirstMeeting) GetXrayCode() []string {
	var xrayCode []string
	for _, x := range fm.xray {
		xrayCode = append(xrayCode, x.GetCode())
	}
	return xrayCode
}
func (fm *FirstMeeting) GetAllCodes() []string {
	var allCodes []string
	for _, x := range fm.xray {
		allCodes = append(allCodes, x.GetCode())
	}
	for _, e := range fm.extraExam {
		if sono, ok := e.(*Sono); ok {
			allCodes = append(allCodes, sono.GetCode())
		}
	}
	return allCodes
}
func (fm *FirstMeeting) ToString() string {
	var temXray, tempExtraExam string
	for _, x := range fm.xray {
		temXray += x.ToString() + ", "
	}
	for _, e := range fm.extraExam {
		if sono, ok := e.(*Sono); ok {
			tempExtraExam += sono.ToString() + ", "
		}
	}
	if len(temXray) > 0 {
		temXray = temXray[:len(temXray)-2] // 마지막 쉼표 제거
	}
	return "FirstMeeting, site:" + strings.Join(fm.sites, ", ") +
		", duration: " + fm.duration +
		", physicalExam: " + fm.physicalExam +
		", xray: " + temXray +
		", extraExam: " + tempExtraExam
}
func (fm *FirstMeeting) IsEmpty() bool {
	return len(fm.sites) == 0 && fm.duration == "" && fm.physicalExam == "" && len(fm.xray) == 0 && len(fm.extraExam) == 0
}
func (fm *FirstMeeting) SetReset() {
	fm.sites = []string{}
	fm.duration = ""
	fm.physicalExam = ""
	fm.xray = []Xray{}
	fm.extraExam = []any{}
}
func (fm FirstMeeting) GetChartText() string {
	var temp string
	var xr string
	var dur string
	var extra string

	dayMonYear := "d"

	if fm.duration != "" {
		if strings.Contains(fm.duration, "m") {
			dayMonYear = ""
		}
		if strings.Contains(fm.duration, "w") {
			dayMonYear = ""
		}
		if strings.Contains(fm.duration, "y") {
			dayMonYear = ""
		}
		dur = "for " + fm.duration + dayMonYear
	}

	for i, site := range fm.sites {
		if i == 0 {
			temp = "cc: " + site + " " + dur + "\n"
		} else {
			temp += "     " + site + "\n"
		}
	}
	if fm.physicalExam != "" {
		temp += fm.physicalExam + "\n"
	}
	for _, x := range fm.xray {
		if x.GetText() != "" {
			xr += x.GetText() + "\n"
		}
	}
	if len(fm.extraExam) > 0 {
		for _, e := range fm.extraExam {
			if sono, ok := e.(*Sono); ok {
				if sono.GetText() != "" {
					extra += sono.GetText() + "\n"
				}
			}
		}
	}

	temp += "pmhx: (-)" + "\n"
	temp += " " + "\n" //빈칸이라도 들어가야 \n이 엔터로 들어갑니다, 버그?
	temp += xr
	if extra != "" {
		temp += extra + "\n"
	}
	temp += " " + "\n" //빈칸이라도 들어가야 \n이 엔터로 들어갑니다
	return temp
}
