package controllers

import (
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"

	"github.com/seeseasdk/go_hotstring_v3/data/hotstrings"
	"github.com/seeseasdk/go_hotstring_v3/data/models"
)

type HotstringController struct {
	cc         *ChannelController
	buffer     string
	treatments *models.Treatments
}

func NewHotstringController(cc *ChannelController) *HotstringController {
	return &HotstringController{
		cc:         cc,
		buffer:     "",
		treatments: models.NewTreatments(),
	}
}

func (c *HotstringController) Start() {
	slog.Info("HotstringController started")

	go func() {
		for {
			select {
			case stuff := <-c.cc.InputChan:
				if stuff.Do == "FlushTreatments" {
					fmt.Printf("🚀 [FLUSH] Processing buffer: '%s'\n", c.buffer)
					os.Stdout.Sync()

					// Trigger processing of the accumulated buffer
					output := c.processBuffer()

					// Then output
					c.cc.OutputChan <- models.NewChannelStuff("HotstringController", "OutputController", "UpdateOutput", true, output)

					// Clear buffer after processing
					c.buffer = ""
					continue
				}

				// AddChar message processing
				if stuff.Do == "AddChar" {
					if charMap, ok := stuff.Object.(map[string]interface{}); ok {
						if char, exists := charMap["char"]; exists {
							if charRune, ok := char.(rune); ok {
								c.buffer += string(charRune)
								fmt.Printf("📝 [BUFFER] '%s'\n", c.buffer)
								os.Stdout.Sync()
							}
						}
					}
					continue
				}

				// Backspace message processing
				if stuff.Do == "Backspace" {
					if len(c.buffer) > 0 {
						// Remove the last character (considering unicode/runes properly by converting to runes first)
						runes := []rune(c.buffer)
						if len(runes) > 0 {
							c.buffer = string(runes[:len(runes)-1])
						}
						fmt.Printf("🔙 [BUFFER] '%s'\n", c.buffer)
						os.Stdout.Sync()
					}
					continue
				}

			case <-c.cc.ResetChan:
				// Reset treatments when output is complete
				c.treatments = models.NewTreatments()
				slog.Debug("HotstringController: Treatments reset")
			}
		}
	}()
}

// processBuffer scans the buffer and extracts matches sequentially by position
func (c *HotstringController) processBuffer() *models.OutputStuff {
	// 모든 가능한 매치를 찾아서 위치별로 정렬
	type Match struct {
		position int
		length   int
		category string
		key      string
		baseKey  string
		value    interface{}
	}

	var matches []Match

	// 0. K_Drugs 매치 찾기 (d, du, dp + 숫자. 예: d3, du7)
	drugRegex := regexp.MustCompile(`(du|dp|d)(\d+)`)
	drugMatches := drugRegex.FindAllStringSubmatchIndex(c.buffer, -1)
	for _, dm := range drugMatches {
		fullMatchStart := dm[0]
		fullMatchEnd := dm[1]
		prefixStart := dm[2]
		prefixEnd := dm[3]
		numStart := dm[4]
		numEnd := dm[5]

		fullStr := c.buffer[fullMatchStart:fullMatchEnd]
		prefix := c.buffer[prefixStart:prefixEnd]
		numStr := c.buffer[numStart:numEnd]

		matches = append(matches, Match{
			position: fullMatchStart,
			length:   len(fullStr),
			category: "Drugs",
			key:      fullStr,
			baseKey:  prefix,
			value:    numStr,
		})
	}

	// 1. K_Simples 매치 찾기 (최우선: 모든 모드보다 먼저 검사)
	for k, v := range hotstrings.K_Simples {
		if idx := strings.Index(c.buffer, k); idx != -1 {
			matches = append(matches, Match{idx, len(k), "Simples", k, k, v})
		}
	}

	// 2. K_SIMPLE_CODE 매치 찾기 (최우선: 모든 모드보다 먼저 검사)
	for k, v := range hotstrings.K_SIMPLE_CODE {
		if idx := strings.Index(c.buffer, k); idx != -1 {
			matches = append(matches, Match{idx, len(k), "SimpleCode", k, k, v})
		}
	}

	isFirstMeetingMode := strings.HasPrefix(c.buffer, "z")
	isXrayMode := strings.HasPrefix(c.buffer, "x")
	isSonoMode := strings.HasPrefix(c.buffer, "s")

	if isFirstMeetingMode {
		// z로 시작하면 나머지 문자열에서는 K_FirstMeeting만 연속으로 찾는다 (예: zcvbshb -> cvb, shb)
		for k, v := range hotstrings.K_FirstMeeting {
			idx := 1 // 'z' 이후부터 검색
			for {
				if idx >= len(c.buffer) {
					break
				}
				foundIdx := strings.Index(c.buffer[idx:], k)
				if foundIdx == -1 {
					break
				}
				actualPos := idx + foundIdx
				matches = append(matches, Match{actualPos, len(k), "FirstMeeting", k, k, v})
				idx = actualPos + len(k)
			}
		}
	} else if isXrayMode {
		for k, v := range hotstrings.K_Xrays {
			idx := 1 // 'x' 이후부터 검색
			for {
				if idx >= len(c.buffer) {
					break
				}
				foundIdx := strings.Index(c.buffer[idx:], k)
				if foundIdx == -1 {
					break
				}
				actualPos := idx + foundIdx
				matches = append(matches, Match{actualPos, len(k), "Xrays", k, k, v})
				idx = actualPos + len(k)
			}
		}
	} else if isSonoMode {
		for k, v := range hotstrings.K_Sonos {
			idx := 1 // 's' 이후부터 검색
			for {
				if idx >= len(c.buffer) {
					break
				}
				foundIdx := strings.Index(c.buffer[idx:], k)
				if foundIdx == -1 {
					break
				}
				actualPos := idx + foundIdx
				matches = append(matches, Match{actualPos, len(k), "Sonos", k, k, v})
				idx = actualPos + len(k)
			}
		}
	} else {
		// 3. K_ESWT_ONLY (Prefix 'e') 매치 찾기
		for k, v := range hotstrings.K_ESWT_ONLY {
			target := "e" + k
			if idx := strings.Index(c.buffer, target); idx != -1 {
				matches = append(matches, Match{idx, len(target), "ESWT", target, k, v})
			}
		}

		// 4. K_FirstMeeting (Prefix 'z') 매치 찾기 (혹시 z가 중간에 있는 경우 대비)
		for k, v := range hotstrings.K_FirstMeeting {
			target := "z" + k
			if idx := strings.Index(c.buffer, target); idx != -1 {
				matches = append(matches, Match{idx, len(target), "FirstMeeting", target, k, v})
			}
		}

		// 5. K_Xrays (Prefix 'x') 매치 찾기
		for k, v := range hotstrings.K_Xrays {
			target := "x" + k
			if idx := strings.Index(c.buffer, target); idx != -1 {
				matches = append(matches, Match{idx, len(target), "Xrays", target, k, v})
			}
		}

		// 6. K_Sonos (Prefix 's') 매치 찾기
		for k, v := range hotstrings.K_Sonos {
			target := "s" + k
			if idx := strings.Index(c.buffer, target); idx != -1 {
				matches = append(matches, Match{idx, len(target), "Sonos", target, k, v})
			}
		}

		// 7. K_Blocks 직접 키 매치 찾기
		for k, v := range hotstrings.K_Blocks {
			if idx := strings.Index(c.buffer, k); idx != -1 {
				matches = append(matches, Match{idx, len(k), "Blocks-Direct", k, k, v})
			}
		}

		// 8. K_Blocks 프리픽스 매치 찾기 ('c', 'p')
		for k, v := range hotstrings.K_Blocks {
			// 관절 부위(sh, kn, ak, eb, wr)나 caudal 자체는 C-arm(c) prefix를 거의 쓰지 않으므로, c를 단독으로(caudal) 식별할 수 있게 prefix 조합에서 제외
			isJoint := strings.HasPrefix(k, "sh") || strings.HasPrefix(k, "kn") || strings.HasPrefix(k, "ak") || strings.HasPrefix(k, "eb") || strings.HasPrefix(k, "wr") || k == "caudal"
			if !isJoint {
				target_c := "c" + k
				if idx := strings.Index(c.buffer, target_c); idx != -1 {
					matches = append(matches, Match{idx, len(target_c), "Blocks-C", target_c, k, v})
				}
			}

			target_p := "p" + k
			if idx := strings.Index(c.buffer, target_p); idx != -1 {
				matches = append(matches, Match{idx, len(target_p), "Blocks-P", target_p, k, v})
			}
		}
	}

	// 위치 순으로 정렬 (앞에서부터), 같은 위치면 긴 것이 우선 (Longest match first)
	for i := 0; i < len(matches)-1; i++ {
		for j := i + 1; j < len(matches); j++ {
			if matches[i].position > matches[j].position {
				matches[i], matches[j] = matches[j], matches[i]
			} else if matches[i].position == matches[j].position {
				if matches[i].length < matches[j].length {
					matches[i], matches[j] = matches[j], matches[i]
				}
			}
		}
	}

	// 겹치지 않는 매치들을 순서대로 처리
	processed := make(map[int]bool)
	output := models.NewOutputStuff(len(c.buffer), "", "", "", []string{}, "", "", "")

	var combinedFirstMeeting *models.FirstMeeting

	for _, match := range matches {
		// 이미 처리된 부분과 겹치는지 확인
		overlap := false
		for pos := match.position; pos < match.position+match.length; pos++ {
			if processed[pos] {
				overlap = true
				break
			}
		}

		if !overlap {
			// 처리된 위치 표시 - 매치 길이만큼 일단 다 표시
			for pos := match.position; pos < match.position+match.length; pos++ {
				processed[pos] = true
			}

			// 접미사 'e', 's', 'pe'가 있는지 확인 (각각 ESWT, SonoStim, PainEraser 처리를 추가하기 위함)
			// 여러 접미사가 연달아 올 수 처리 (예: ...pes, ...espe)
			if match.category != "Sonos" && match.category != "Xrays" && match.category != "FirstMeeting" {
				remainderPos := match.position + match.length
				for remainderPos < len(c.buffer) && !processed[remainderPos] {
					remainder := c.buffer[remainderPos:]
					matchedSuffix := false

					if strings.HasPrefix(remainder, "pe") {
						if peVal, exists := hotstrings.K_PainEraser[match.baseKey]; exists {
							slog.Debug("Hotstring Triggered (PainEraser Suffix)", "trigger", match.key+"pe", "baseKey", match.baseKey)
							c.treatments.SetAddExtraTreatments(*peVal)
							processed[remainderPos] = true
							processed[remainderPos+1] = true
							remainderPos += 2
							matchedSuffix = true
						}
					} else if strings.HasPrefix(remainder, "e") {
						if eswtVal, exists := hotstrings.K_ESWT_ONLY[match.baseKey]; exists {
							slog.Debug("Hotstring Triggered (ESWT Suffix)", "trigger", match.key+"e", "baseKey", match.baseKey)
							c.treatments.SetESWT(*eswtVal)
							processed[remainderPos] = true
							remainderPos += 1
							matchedSuffix = true
						}
					} else if strings.HasPrefix(remainder, "s") {
						if sntVal, exists := hotstrings.K_SonoStim[match.baseKey]; exists {
							slog.Debug("Hotstring Triggered (SonoStim Suffix)", "trigger", match.key+"s", "baseKey", match.baseKey)
							c.treatments.SetAddExtraTreatments(*sntVal)
							processed[remainderPos] = true
							remainderPos += 1
							matchedSuffix = true
						}
					} else if remainder[0] == 'p' || remainder[0] == 'c' {
						// 'p'(isP 속성)나 'c'(caudal) 문자가 중간에 끼어 있어도 뒤의 leftover 루프에서 처리할 수 있도록, 무시하고 다음 접미사 탐색을 계속함
						remainderPos += 1
						matchedSuffix = true
					}

					if !matchedSuffix {
						break
					}
				}
			}

			// 매치 처리
			switch match.category {
			case "Drugs":
				slog.Debug("Hotstring Triggered (Drugs)", "trigger", match.key, "drugDays", match.value)
				days, ok := match.value.(string)
				if ok {
					output.SetDrug(days)
					if oc, exists := hotstrings.K_Drugs[match.baseKey]; exists {
						output.AddOrderCode(oc)
					}
				}
			case "ESWT":
				slog.Debug("Hotstring Triggered (ESWT)", "trigger", match.key)
				c.treatments.SetESWT(*match.value.(*models.ESWT))
			case "FirstMeeting":
				firstMeeting := match.value.(*models.FirstMeeting)
				slog.Debug("Hotstring Triggered (FirstMeeting)", "trigger", match.key, "sites", firstMeeting.GetSites())

				if combinedFirstMeeting == nil {
					// 새로운 객체를 만들어서 복사 (참조를 막기 위함)
					combinedFirstMeeting = models.NewFirstMeeting(
						append([]string{}, firstMeeting.GetSites()...),
						firstMeeting.GetPhysicalExam(),
						append([]models.Xray{}, firstMeeting.GetXray()...),
						append([]any{}, firstMeeting.GetExtraExam()...),
					)
					combinedFirstMeeting.SetDuration(firstMeeting.GetDuration())
				} else {
					combinedFirstMeeting.Merge(firstMeeting)
				}
			case "Xrays":
				slog.Debug("Hotstring Triggered (Xrays)", "trigger", match.key)
				xray, ok := match.value.(*models.Xray)
				if ok {
					curChart := output.GetChartText()
					if curChart != "" {
						output.SetChartText(curChart + "\n" + xray.GetText())
					} else {
						output.SetChartText(xray.GetText())
					}
					output.AddOrderCode(xray.GetCode())
				}
			case "Sonos":
				slog.Debug("Hotstring Triggered (Sonos)", "trigger", match.key)
				sono, ok := match.value.(*models.Sono)
				if ok {
					curChart := output.GetChartText()
					if curChart != "" {
						output.SetChartText(curChart + "\n" + sono.GetText())
					} else {
						output.SetChartText(sono.GetText())
					}
					output.AddOrderCode(sono.GetCode())
				}
			case "Blocks-Direct", "Blocks-C", "Blocks-P":
				injection := match.value.(*models.Injection)
				slog.Debug("Hotstring Triggered (Blocks)", "trigger", match.key, "site", injection.GetSite())
				c.treatments.SetAddInjection(*injection)
			case "Simples":
				slog.Debug("Hotstring Triggered (Simples)", "trigger", match.key)
				simple, ok := match.value.(*models.SimpleInput)
				if ok {
					curSimple := output.GetSimpleText()
					if curSimple != "" {
						output.SetSimpleText(curSimple + "\n" + simple.GetText())
					} else {
						output.SetSimpleText(simple.GetText())
					}
					if simple.GetExtraDo() != "" {
						output.SetExtraDo(simple.GetExtraDo())
					}
				}
				case "SimpleCode":
					slog.Debug("Hotstring Triggered (SimpleCode)", "trigger", match.key)
					strCode, ok := match.value.(string)
					if ok {
						output.AddOrderCode(strCode)
					}				}
			}
		}
	// 모든 매치가 끝난 후 처리되지 않은 문자 중 'c'가 있으면 'caudal'로 처리, 'p'가 남으면 모든 injection을 isP = true로 변경
	leftoverP := false
	for i := 0; i < len(c.buffer); i++ {
		if !processed[i] {
			if c.buffer[i] == 'c' {
				if caudalVal, exists := hotstrings.K_Blocks["caudal"]; exists {
					slog.Debug("Hotstring Triggered (Leftover 'c' -> caudal)", "trigger", "caudal", "site", caudalVal.GetSite())
					c.treatments.SetAddInjection(*caudalVal)
					processed[i] = true
				}
			} else if c.buffer[i] == 'p' {
				slog.Debug("Hotstring Triggered (Leftover 'p' -> isP=true)")
				leftoverP = true
				processed[i] = true
			} else if c.buffer[i] == '7' {
				slog.Debug("Hotstring Triggered (Leftover '7' -> hasSeven=true)")
				c.treatments.SetHasSeven(true)
				processed[i] = true
			}
		}
	}

	if leftoverP {
		c.treatments.SetAllInjectionsIsP(true)
	}

	if isFirstMeetingMode && combinedFirstMeeting != nil {
		re := regexp.MustCompile(`f(\d+[dmyw]?)`)
		durationMatches := re.FindStringSubmatch(c.buffer)
		if len(durationMatches) > 1 {
			combinedFirstMeeting.SetDuration(durationMatches[1])
		}
	}

	if combinedFirstMeeting != nil {
		newChart := combinedFirstMeeting.GetChartText()
		curChart := output.GetChartText()
		if curChart != "" && newChart != "" {
			// FirstMeeting의 내용이 다른 항목들보다 먼저 나오는 것이 자연스러우므로 앞에 배치합니다.
			output.SetChartText(newChart + "\n" + curChart)
		} else if newChart != "" {
			output.SetChartText(newChart)
		}
		output.AddOrderCodeList(combinedFirstMeeting.GetAllCodes())
	}

	if !c.treatments.IsEmpty() {
		// 우선순위에 맞게 정렬 (caudal, mbb 등 순서 보장)
		c.treatments.SortInjections()

		chartText := output.GetChartText()
		newChartText := c.treatments.GetTextForChart()
		if newChartText != "" {
			if chartText != "" {
				output.SetChartText(chartText + "\n" + newChartText)
			} else {
				output.SetChartText(newChartText)
			}
		}

		spec := output.GetSpecificText()
		newSpec := c.treatments.GetTextForSpecific()
		if newSpec != "" {
			if spec != "" {
				output.SetSpecificText(spec + "\n" + newSpec)
			} else {
				output.SetSpecificText(newSpec)
			}
		}

		mx := output.GetMx999Text()
		newMx := c.treatments.GetTextForMx999()
		if newMx != "" {
			if mx != "" {
				output.SetMx999Text(mx + "\n" + newMx)
			} else {
				output.SetMx999Text(newMx)
			}
		}

		codes, _ := c.treatments.GetOrderCode()
		output.AddOrderCodeList(codes)

		drug := c.treatments.GetDrug()
		if drug != "" {
			output.SetDrug(drug)
		}
	}

	return output
}
