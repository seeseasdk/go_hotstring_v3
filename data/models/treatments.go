package models

import (
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/seeseasdk/go_hotstring_v3/data/constants"
)

type Treatments struct {
	day             time.Time
	injections      []Injection
	extraTreatments []any
	isP             bool
	hasSeven        bool
	drug            string
	followUp        string
	clipboardMemos  []string
	eSWT            ESWT
}

func NewTreatments() *Treatments {
	return &Treatments{
		day:             time.Now(),
		injections:      []Injection{},
		extraTreatments: []any{},
		isP:             false,
		drug:            "",
		followUp:        "",
		clipboardMemos:  []string{},
		eSWT:            *NewESWT("", "", "", "", false),
	}
}
func (i *Treatments) SetDay(day time.Time) {
	i.day = day
}
func (i *Treatments) SetAddInjection(injection Injection) {
	i.injections = append(i.injections, injection)
}
func (i *Treatments) SetAddExtraTreatments(extraTreatment any) {
	i.extraTreatments = append(i.extraTreatments, extraTreatment)
}
func (i *Treatments) SetIsP(isP bool) {
	i.isP = isP
}
func (i *Treatments) SetAllInjectionsIsP(isP bool) {
	for idx := range i.injections {
		i.injections[idx].SetIsP(isP) // assuming Injection has SetIsP, else i.injections[idx].isP = isP
	}
}
func (i *Treatments) SetDrug(drugDays string) {
	i.drug = drugDays
}
func (i *Treatments) SetFollowUp(followUp string) {
	i.followUp = followUp
}
func (i *Treatments) AddClipboardMemo(memo string) {
	i.clipboardMemos = append(i.clipboardMemos, memo)
}
func (i *Treatments) SetESWT(eswt ESWT) {
	i.eSWT = eswt
}
func (i *Treatments) SortInjections() {
	sort.SliceStable(i.injections, func(a, b int) bool {
		priorityA := getPriority(i.injections[a].site)
		priorityB := getPriority(i.injections[b].site)
		return priorityA > priorityB
	})
}

func getPriority(site string) int {
	// mbb 등 메인 척추 시술이 첫 번째(c) 기호 표시)가 되어야 하므로 가장 높은 우선순위(3)를 줍니다.
	if strings.Contains(site, "mbb") || strings.Contains(site, "fjb") ||
		strings.Contains(site, "snrb") || strings.Contains(site, "drgb") ||
		strings.Contains(site, "cpb") || strings.Contains(site, "pcb") ||
		strings.Contains(site, "pdnb") || strings.Contains(site, "intercostal") ||
		strings.Contains(site, "quadratus") || strings.Contains(site, "trapezius") {
		return 3
	}
	// caudal은 두 번째에 와야 하므로 중간 우선순위(2)를 줍니다.
	if strings.Contains(site, "caudal") {
		return 2
	}
	// 그 외(어깨 IA 등 관절 주사 등)는 세 번째(1)로 갑니다.
	return 1
}
func (i *Treatments) SetReset() {
	i.day = time.Now()
	i.injections = []Injection{}
	i.extraTreatments = []any{}
	i.isP = false
	i.drug = ""
	i.followUp = ""
	i.clipboardMemos = []string{}
	i.eSWT.SetReset()
	log.Debug("models/treatment.go", "treatment", "treatment reset")
}
func (i Treatments) GetDay() time.Time {
	return i.day
}
func (i Treatments) GetTreatments() []Injection {
	return i.injections
}
func (i Treatments) GetExtraTreatments() []any {
	return i.extraTreatments
}
func (i Treatments) GetIsP() bool {
	return i.isP
}
func (i Treatments) GetDrug() string {
	return i.drug
}
func (i Treatments) GetFollowUp() string {
	return i.followUp
}
func (i Treatments) getFollowUpDate(count int) string {
	daysToAdd := count
	if daysToAdd <= 0 {
		switch i.day.Weekday() {
		case time.Monday, time.Tuesday, time.Wednesday:
			daysToAdd = 3
		default: // 목, 금, 토, 일
			daysToAdd = 4
		}
	}
	futureDate := i.day.AddDate(0, 0, daysToAdd)
	return futureDate.Format("2006-01-02")
}
func (i Treatments) getFollowUpDateString(countStr string) string {

	// --- 1. 요청하신 특별 케이스: countStr가 "6m"이면 6개월 추가 ---
	if countStr == "6m" {
		monthsToAdd := 6

		year, month, day := i.day.Date()
		targetMonth := int(month) + monthsToAdd
		targetYear := year

		if targetMonth > 12 {
			targetYear += (targetMonth - 1) / 12
			targetMonth = (targetMonth-1)%12 + 1
		}

		lastDayOfTargetMonth := time.Date(targetYear, time.Month(targetMonth)+1, 1, 0, 0, 0, -1, i.day.Location()).Day()

		var futureDate time.Time
		if day > lastDayOfTargetMonth {
			// "날짜가 없는 날이면 다음날로"
			lastDate := time.Date(targetYear, time.Month(targetMonth), lastDayOfTargetMonth, i.day.Hour(), i.day.Minute(), i.day.Second(), i.day.Nanosecond(), i.day.Location())
			futureDate = lastDate.AddDate(0, 0, 1) // 마지막 날의 다음 날 (즉, 다음 달 1일)
		} else {
			// 날짜가 존재하면 그대로 사용
			futureDate = time.Date(targetYear, time.Month(targetMonth), day, i.day.Hour(), i.day.Minute(), i.day.Second(), i.day.Nanosecond(), i.day.Location())
		}
		// 6개월 로직이 끝나면 바로 반환
		return futureDate.Format("2006-01-02")
	}

	// --- 2. 기존 로직: "6m"이 아닌 모든 경우 (일(Day) 더하기) ---

	// 2-1. 문자열 'countStr'를 정수 'daysToAdd'로 변환합니다.
	//      변환 실패(err != nil) 시, 0으로 간주하여 요일 로직을 타도록 합니다.
	daysToAdd, err := strconv.Atoi(countStr)
	if err != nil {
		daysToAdd = 0 // "abc" 같은 잘못된 입력이 오면 0일로 처리
	}

	// 2-2. count가 0 이하일 때만 요일 기반 로직을 실행하여
	//      daysToAdd 값을 덮어씁니다.
	if daysToAdd <= 0 {
		// 2-3. switch 문을 사용하면 코드가 명확해집니다.
		switch i.day.Weekday() {
		case time.Monday, time.Tuesday, time.Wednesday:
			daysToAdd = 3
		default: // 목, 금, 토, 일
			daysToAdd = 4
		}
	}

	// 2-4. 결정된 daysToAdd 값을 기준으로 AddDate를 한 번만 호출합니다.
	futureDate := i.day.AddDate(0, 0, daysToAdd)
	return futureDate.Format("2006-01-02")
}
func (i Treatments) GetESWT() ESWT {
	return i.eSWT
}
func (i Treatments) GetTextForChart() string {
	var text string
	p := ""

	carmTreat := ""
	periTreat := ""

	for _, inject := range i.injections {
		if inject.isFromClipboard {
			continue
		}

		switch inject.isP {
		case true:
			p = "p"
		default:
			p = ""
		}

		// isN 처리 추가
		if inject.isN {
			p = "n"
		}
		switch inject.isWithCarm {
		case true:
			if carmTreat == "" {
				carmTreat += "c) " + inject.direction + " " + inject.site + " " + p + "\n"
			} else {
				if strings.Contains(inject.site, "caudal") {
					carmTreat += "    " + inject.site + " " + p + "\n"
				} else {
					carmTreat += "    " + inject.direction + " " + inject.site + " " + p + "\n"
				}
			}
		default:
			if periTreat == "" {
				periTreat += "s) " + inject.direction + " " + inject.site + " " + p + "\n"
			} else {
				periTreat += "    " + inject.direction + " " + inject.site + " " + p + "\n"
			}
		}
	}
	text += carmTreat
	text += periTreat

	if !i.eSWT.IsEmpty() {
		switch i.eSWT.GetFeeType() {
		case constants.K_NORMAL:
			text += "e) focus on " + i.eSWT.GetDirection() + " " + i.eSWT.GetFocus() + "\n"
			text += "   radial on " + i.eSWT.GetDirection() + " " + i.eSWT.GetRadial() + "\n"
		case constants.K_FREE:
			text += "ef) focus on " + i.eSWT.GetDirection() + " " + i.eSWT.GetFocus() + "\n"
			text += "    radial on " + i.eSWT.GetDirection() + " " + i.eSWT.GetRadial() + "\n"
		case constants.K_FREE_RADIAL_ONLY:
			text += "ef) radial on " + i.eSWT.GetDirection() + " " + i.eSWT.GetRadial() + "\n"
		}
	}
	if len(i.extraTreatments) > 0 {
		for _, extra := range i.extraTreatments {
			switch extra := extra.(type) {
			case string:
				if strings.Contains(text, constants.K_PT) {
					continue
				}
				if !strings.Contains(text, "pt) ") {
					text += "pt) "
				} else {
					text += "     "
				}
				text += fmt.Sprintf("%v", extra) + "\n"
			case SonoStim:
				text += "snt) " + extra.GetDirection() + " " + extra.GetSite() + "\n"
			case PainEraser:
				text += "pe) " + extra.GetDirection() + " " + extra.GetSite() + "\n"
			}
		}
	}

	if text == "" {
		return ""
	}

	if i.followUp == "" {
		if strings.Contains(periTreat, "hyaluron") {
			return text + "f/u) " + i.getFollowUpDate(7) + "\n"
		}
		if strings.Contains(periTreat, "시노비안") {
			return text + "f/u) " + i.getFollowUpDateString("6m") + "\n"
		}
		if strings.Contains(periTreat, "아라간") {
			return text + "f/u) " + i.getFollowUpDate(7) + "\n"
		}
		return text + "f/u) " + i.getFollowUpDate(0) + "\n"
	}

	day, _ := strconv.Atoi(i.followUp)
	return text + "f/u) " + i.getFollowUpDate(day) + "\n"
}
func (i Treatments) GetTextForSpecific() string {
	var text string
	p := ""

	carmTreat := ""
	periTreat := ""

	formattedDate := i.day.Format("2006-01-02")

	for _, inject := range i.injections {
		switch inject.isP {
		case true:
			p = "p"
		default:
			p = ""
		}
		// isN 처리 추가
		if inject.isN {
			p = "n"
		}

		switch inject.isWithCarm {
		case true:
			if carmTreat == "" {
				carmTreat += formattedDate + " " + "c) " + inject.direction + " " + inject.site + " " + p + "\n"
			} else {
				if strings.Contains(inject.site, "caudal") {
					carmTreat += "                    " + inject.site + " " + p + "\n"
				} else {
					carmTreat += "                    " + inject.direction + " " + inject.site + " " + p + "\n"
				}
			}
		default:
			if periTreat == "" && carmTreat == "" {
				periTreat += formattedDate + " " + "s) " + inject.direction + " " + inject.site + " " + p + "\n"
			} else if periTreat == "" && carmTreat != "" {
				periTreat += "                " + "s) " + inject.direction + " " + inject.site + " " + p + "\n"
			} else {
				periTreat += "                   " + inject.direction + " " + inject.site + " " + p + "\n"
			}
		}
	}

	text += carmTreat
	text += periTreat

	if !i.eSWT.IsEmpty() {
		switch i.eSWT.GetIsOnlyEswt() {
		case true:
			text += formattedDate + " "
		default:
			text += "                "
		}

		switch i.eSWT.GetFeeType() {
		case constants.K_NORMAL:
			text += "e) focus on " + i.eSWT.GetDirection() + " " + i.eSWT.GetFocus() + "\n"
			text += "                    " + "radial on " + i.eSWT.GetDirection() + " " + i.eSWT.GetRadial() + "\n"
		case constants.K_ESWT_FREE:
			text += "ef) focus on " + i.eSWT.GetDirection() + " " + i.eSWT.GetFocus() + "\n"
			text += "                    " + "radial on " + i.eSWT.GetDirection() + " " + i.eSWT.GetRadial() + "\n"
		case constants.K_FREE_RADIAL_ONLY:
			text += "ef) radial on " + i.eSWT.GetDirection() + " " + i.eSWT.GetRadial() + "\n"
		}
	}
	if len(i.extraTreatments) > 0 {
		for _, extra := range i.extraTreatments {
			switch extra := extra.(type) {
			case SonoStim:
				text += "                snt) " + extra.GetDirection() + " " + extra.GetSite() + "\n"
			case PainEraser:
				text += "                pe) " + extra.GetDirection() + " " + extra.GetSite() + "\n"
			}
		}
	}

	// 클립보드 매모(복사한 텍스트)를 specificText에 추가 반영
	for _, memo := range i.clipboardMemos {
		if text == "" {
			text += memo + "\n"
		} else {
			text += "                " + memo + "\n"
		}
	}

	return text
}
func (i Treatments) GetTextForMx999() string {
	var text string
	scIsWithKnee := false

	for _, inject := range i.injections {
		if inject.code == "" {
			continue
		}
		inject.SetSite(strings.TrimSpace(inject.site))
		if len(inject.site) > 4 {
			if inject.site[len(inject.site)-2:len(inject.site)-1] == " n" ||
				inject.site[len(inject.site)-2:len(inject.site)-1] == " p" {
				inject.SetSite(inject.site[:len(inject.site)-2])
			}
		}
		// log.Error(inject.site[:len(inject.site)-2])
		// log.Error(inject.ToString())

		switch inject.code {
		case constants.K_SHB, constants.K_SH, constants.K_PSHB, constants.K_PSH:
			text += inject.direction + " " + "견갑신경차단술" + "\n"
		case constants.K_ANB, constants.K_AN:
			text += inject.direction + " " + "액와신경차단술" + "\n"
		case constants.K_SANB, constants.K_SAN, constants.K_PSANB, constants.K_PSAN:
			text += inject.direction + " " + "액와하부신경차단술" + "\n"
		case constants.K_SCB, constants.K_SC:
			if inject.site == "caudal" {
				if inject.direction != "" {
					text += inject.direction + "좌골신경차단술" + "\n"
				} else {
					text += "좌골신경차단술" + "\n"
				}
			} else {
				switch scIsWithKnee {
				case true:
					text += inject.direction + " " + "좌골신경차단술 (대퇴)" + "\n"
				default:
					text += inject.direction + " " + "좌골신경차단술" + "\n"
				}
			}
		case constants.K_F1, constants.K_PF1:
			text += inject.direction + " " + "대퇴신경차단술 (무릎)" + "\n"
			scIsWithKnee = true
		case constants.K_F05, constants.K_F025, constants.K_PF05, constants.K_PF025:
			if strings.Contains(inject.site, "ankle") ||
				strings.Contains(inject.site, "atfl") ||
				strings.Contains(inject.site, "deltoid") {
				text += inject.direction + " " + "대퇴신경차단술 (발목)" + "\n"
			} else if strings.Contains(inject.site, "foot") {
				text += inject.direction + " " + "대퇴신경차단술 (발목 아래)" + "\n"
			} else {
				text += inject.direction + " " + "대퇴신경차단술 (무릎)" + "\n"
				scIsWithKnee = true
			}
		case constants.K_KHB, constants.K_KH, constants.K_PKHB, constants.K_PKH, constants.K_PKHSB, constants.K_PKSH:
			text += "KL grade G" + "\n"
			text += inject.direction + " 차수: " + "\n"
		default:
			text += inject.direction + " " + inject.site + "\n"
		}
	}
	return text
}
func (i *Treatments) SetHasSeven(hasSeven bool) {
	i.hasSeven = hasSeven
}

func (i Treatments) GetOrderCode() ([]string, error) {
	var firstCode string
	var secondCode string
	var injections []Injection
	for _, inj := range i.injections {
		if !inj.isFromClipboard {
			injections = append(injections, inj)
		}
	}
	// var thirdCode string
	var injectCode string
	var result []string
	for _, inject := range injections { //main c-arm block
		if slices.Contains(constants.K_FIRST_BLOCKS, inject.code) {
			firstCode = inject.code
			injections = slices.DeleteFunc(injections, func(inj Injection) bool {
				return inj.code == inject.code
			})
			break
		}
	}
	if firstCode == "" {
		for _, inject := range injections { //first block peri carm not axial
			if slices.Contains(constants.K_FIRST_BLOCKS_NOT_AXIAL, inject.code) {
				switch inject.code {
				case constants.K_SHB:
					firstCode = constants.K_CSHB
				case constants.K_SH:
					firstCode = constants.K_CSH
				case constants.K_PSHB:
					firstCode = constants.K_PSHB
				case constants.K_PSH:
					firstCode = constants.K_PSH
				case constants.K_SANB:
					firstCode = constants.K_CSANB
				case constants.K_SAN:
					firstCode = constants.K_CSAN
				case constants.K_PSANB:
					firstCode = constants.K_PSANB
				case constants.K_PSAN:
					firstCode = constants.K_PSAN
				case constants.K_PKHB:
					firstCode = constants.K_PKHB
				case constants.K_PKH:
					firstCode = constants.K_PKH
				case constants.K_PKHSB:
					firstCode = constants.K_PKHSB
				case constants.K_PKSH:
					firstCode = constants.K_PKSH
				case constants.K_F1:
					firstCode = constants.K_CF1
				case constants.K_F05:
					firstCode = constants.K_CF05
				case constants.K_F025:
					firstCode = constants.K_CF025
				case constants.K_PF1:
					firstCode = constants.K_PF1
				case constants.K_PF05:
					firstCode = constants.K_PF05
				case constants.K_PF025:
					firstCode = constants.K_PF025
				}
				injections = slices.DeleteFunc(injections, func(inj Injection) bool {
					return inj.code == inject.code
				})
				break
			}
		}
	}
	if firstCode != "" { //add pcb, cpb, pdnb, snrb, caudal
		for _, inject := range injections {
			if slices.Contains(constants.K_SECOND_BLOCKS, inject.code) {
				switch inject.code {
				case constants.K_PDB, constants.K_PD:
					secondCode = constants.K_PD
				case constants.K_CPB, constants.K_CP:
					secondCode = constants.K_CP
				case constants.K_PCB, constants.K_PC:
					secondCode = constants.K_PC
				}
				// switch inject.code {
				// case constants.K_MBB25, constants.K_FJB25, constants.K_SNR25, constants.K_DRG25,
				// 	constants.K_MBB20, constants.K_FJB20, constants.K_SNR20, constants.K_DRG20,
				// 	constants.K_MBB15, constants.K_FJB15, constants.K_SNR15, constants.K_DRG15:
				// 	if inject.code == constants.K_PDB || inject.code == constants.K_PD {
				// 		secondCode = constants.K_PD
				// 	} else if inject.code == constants.K_CPB || inject.code == constants.K_CP {
				// 		secondCode = constants.K_CP
				// 	} else if inject.code == constants.K_PCB || inject.code == constants.K_PC {
				// 		secondCode = constants.K_PC
				// 	}
				// default:
				// 	// secondCode = inject.code
				// 	if inject.code == constants.K_PDB || inject.code == constants.K_PD {
				// 		secondCode = constants.K_PD
				// 	} else if inject.code == constants.K_CPB || inject.code == constants.K_CP {
				// 		secondCode = constants.K_CP
				// 	} else if inject.code == constants.K_PCB || inject.code == constants.K_PC {
				// 		secondCode = constants.K_PC
				// 	}
				// }
				injections = slices.DeleteFunc(injections, func(inj Injection) bool {
					return inj.code == inject.code
				})
				break
			}
		}
	}
	for _, inject := range injections { //add peripheral block
		if slices.Contains(constants.K_THIRD_BLOCKS, inject.code) {
			if firstCode == "" {
				firstCode = inject.code
			} else if firstCode != "" && secondCode == "" {
				switch inject.code {
				case constants.K_SHB, constants.K_SH:
					secondCode = constants.K_SH
				case constants.K_SANB, constants.K_SAN:
					secondCode = constants.K_SAN
				case constants.K_SCB, constants.K_SC:
					secondCode = constants.K_SC
				case constants.K_ICB, constants.K_IC:
					secondCode = constants.K_IC
				case constants.K_F1:
					switch firstCode {
					case constants.K_MBB25, constants.K_FJB25, constants.K_SNR25, constants.K_DRG25:
						secondCode = constants.K_F05
					case constants.K_MBB20, constants.K_FJB20, constants.K_SNR20, constants.K_DRG20, constants.K_MBB15, constants.K_FJB15, constants.K_SNR15, constants.K_DRG15:
						secondCode = constants.K_F1
					default: //first가 peripheral block 일때
						secondCode = constants.K_F1
					}
				case constants.K_F05:
					secondCode = constants.K_F05
				case constants.K_F025:
					secondCode = constants.K_F025
				}
			} else if firstCode != "" && secondCode != "" {
				// thirdCode = inject.code
			}
			// injections = slices.DeleteFunc(injections, func(inj Injection) bool {
			// 	return inj.code == inject.code
			// })
			break
		}
	}
	// if firstCode == "" {
	// 	return nil, fmt.Errorf("no first code found")
	// }
	if firstCode != "" || secondCode != "" {
		injectCode = firstCode + secondCode // + thirdCode
		if i.hasSeven {
			injectCode += "7"
		}
		result = append(result, injectCode)
	}

	// c타입 (isWithCarm=true) 치료가 있으면 .+999_pt0 추가
	for _, inject := range injections {
		if inject.isWithCarm {
			result = append(result, ".+999_pt0")
			break
		}
	}

	for _, extra := range i.extraTreatments {
		switch extra := extra.(type) {
		case string:
			switch extra {
			case constants.K_PT:
				result = append(result, constants.K_NORMAL_PT)
			default:
				result = append(result, extra)
			}
		case SonoStim:
			if !extra.IsEmpty() {
				code := extra.GetCode()
				if code != "" {
					result = append(result, code)
				} else if extra.GetDirection() == "both" {
					result = append(result, constants.K_BOTH_SNT)
				} else {
					result = append(result, constants.K_SINGLE_SNT)
				}
			}
		case PainEraser:
			if !extra.IsEmpty() {
				code := extra.GetCode()
				if code != "" {
					result = append(result, code)
				} else if extra.GetDirection() == "both" {
					result = append(result, constants.K_BOTH_PAIN_ERASER)
				} else {
					result = append(result, constants.K_SINGLE_PAIN_ERASER)
				}
			}
		}
	}
	if !i.eSWT.IsEmpty() {
		switch i.eSWT.GetFeeType() {
		case constants.K_NORMAL:
			if i.eSWT.GetDirection() == constants.K_BOTH {
				if i.eSWT.GetFocus() == "TPZ" || i.eSWT.GetFocus() == "lower back" {
					result = append(result, constants.K_SINGLE_ESWT)
				} else {
					result = append(result, constants.K_BOTH_ESWT)
				}
			} else {
				result = append(result, constants.K_SINGLE_ESWT)
			}
		case constants.K_FREE:
			result = append(result, constants.K_ESWT_FREE)
		case constants.K_FREE_RADIAL_ONLY:
			result = append(result, constants.K_ESWT_FREE_RADIAL)
		default:
			return nil, fmt.Errorf("unknown ESWT fee type: %s", i.eSWT.GetFeeType())
		}
	}
	return result, nil
}
func (i Treatments) IsEmpty() bool {
	return len(i.injections) == 0 &&
		i.drug == "" &&
		i.followUp == "" &&
		i.eSWT.IsEmpty() && len(i.extraTreatments) == 0 && !i.isP
}
func (i Treatments) ToString() string {
	var treatTemp string
	var extraTemp string
	for _, inject := range i.injections {
		treatTemp += inject.direction + " " + inject.site + " " + fmt.Sprintf("%v", inject.isWithCarm) + "\n"
	}
	for _, extra := range i.extraTreatments {
		switch extra := extra.(type) {
		case string:
			extraTemp += fmt.Sprintf("%v", extra) + " "
		case SonoStim:
			extraTemp += extra.ToString()
		case PainEraser:
			extraTemp += extra.ToString()
		}
	}

	return "day: " + i.day.Format("2006-01-02") +
		" treatments: " + treatTemp +
		" extraTreatments: " + extraTemp +
		" isP: " + fmt.Sprintf("%v", i.isP) +
		" drug: " + i.drug +
		" followUp: " + i.followUp +
		" addESWT: " + i.eSWT.ToString()
}
