package controllers

import (
	"log/slog"
	"regexp"
	"strings"

	"github.com/seeseasdk/go_hotstring_v3/data/constants"
	"github.com/seeseasdk/go_hotstring_v3/data/hotstrings"
	"github.com/seeseasdk/go_hotstring_v3/data/models"
)

type HotstringController struct {
	cc         *ChannelController
	buffer     string
	treatments *models.Treatments
	isMuting   bool // TypeStr 출력 중 버퍼 추가 차단
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
					slog.Info("[FLUSH] Processing buffer", "buffer", c.buffer)

					// Ctrl+* 트리거는 클립보드 기반이므로 deleteHostring = 0
					isClipboard := stuff.Object == nil

					// 트리거 시점의 버퍼 내용을 디버그 로그에 기록
					slog.Debug("HotstringController: Triggered", "buffer", c.buffer)

					// Trigger processing of the accumulated buffer
					isCtrlEnter := !isClipboard
					output := c.processBuffer(isClipboard, isCtrlEnter)

					// Then output
					c.cc.OutputChan <- models.NewChannelStuff("HotstringController", "OutputController", "UpdateOutput", true, output)

					// Clear buffer after processing
					c.buffer = ""
					continue
				}

				// AddChar message processing
				if stuff.Do == "AddChar" {
					// 출력 중 뮤팅 상태이면 버퍼에 추가하지 않음
					if c.isMuting {
						continue
					}
					if charMap, ok := stuff.Object.(map[string]interface{}); ok {
						if char, exists := charMap["char"]; exists {
							if charRune, ok := char.(rune); ok {
								c.buffer += string(charRune)
								slog.Debug("[BUFFER] updated", "buffer", c.buffer)
							}
						}
					}
					continue
				}

				// AddClipboardMemo 처리 추가
				if stuff.Do == "AddClipboardMemo" {
					if text, ok := stuff.Object.(string); ok {
						// 클립보드 텍스트를 줄 단위로 분리하여 Injection 객체로 만듦
						lines := strings.Split(text, "\n")
						lastIsWithCarm := false
						lastIsPeri := false
						lastWasErFocus := false
						for _, line := range lines {
							line = strings.TrimSpace(line)
							if line == "" {
								continue
							}

							// 날짜 정규식 "2006-01-02 " 앞부분 자르기
							dateRe := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}\s+`)
							line = dateRe.ReplaceAllString(line, "")

							isWithCarm := false
							isPeri := false
							if strings.HasPrefix(line, "c) ") {
								isWithCarm = true
								lastIsWithCarm = true
								lastIsPeri = false
								line = strings.TrimPrefix(line, "c) ")
							} else if strings.HasPrefix(line, "s) ") {
								isPeri = true
								lastIsPeri = true
								lastIsWithCarm = false
								line = strings.TrimPrefix(line, "s) ")
							} else {
								// c) 나 s) 가 없으면 이전 상태 따라감
								if lastIsWithCarm {
									isWithCarm = true
								} else if lastIsPeri {
									isPeri = true
								}
							}

							// e) 나 ef) 혹은 snt) 나 pe) 로 시작하는 건 주사가 아니므로 제외
							if strings.HasPrefix(line, "e) ") || strings.HasPrefix(line, "ef) ") || strings.HasPrefix(line, "snt) ") || strings.HasPrefix(line, "pe) ") {
								isWithCarm = false
								isPeri = false
								lastIsWithCarm = false
								lastIsPeri = false
							}

							// m8, m13 라인은 그 전체를 specificText(clipboardMemos)에 바로 추가
							if line == "m8" || line == "m13" || strings.HasPrefix(line, "m8 ") || strings.HasPrefix(line, "m13 ") {
								c.treatments.AddClipboardMemo(line)
								continue
							}

							if isWithCarm || isPeri {
								lastWasErFocus = false
								// 끝에 " p" 나 " n" 이 있는지 확인
								isP := false
								if strings.HasSuffix(line, " p") {
									isP = true
									line = strings.TrimSuffix(line, " p")
								}
								isN := false
								if strings.HasSuffix(line, " n") {
									isN = true
									line = strings.TrimSuffix(line, " n")
								}

								// direction 추출
								direction := ""
								lowerLine := strings.ToLower(line)
								if strings.HasPrefix(lowerLine, "both ") {
									direction = "Both"
									line = line[5:]
								} else if strings.HasPrefix(lowerLine, "rt. ") {
									direction = "Rt."
									line = line[4:]
								} else if strings.HasPrefix(lowerLine, "rt ") {
									direction = "Rt."
									line = line[3:]
								} else if strings.HasPrefix(lowerLine, "lt. ") {
									direction = "Lt."
									line = line[4:]
								} else if strings.HasPrefix(lowerLine, "lt ") {
									direction = "Lt."
									line = line[3:]
								}
								line = strings.TrimSpace(line)

								site := line // 나머지는 site

								// 매칭되는 코드를 K_Blocks에서 검색
								code := ""
								mx999 := site
								for _, v := range hotstrings.K_Blocks {
									if v.GetSite() == site {
										code = v.GetCode()
										mx999 = v.GetMx999()
										break
									}
								}

								// Injection 생성
								inj := models.NewInjection(direction, site, code, "", "", "", isWithCarm, isP, isN, nil)
								inj.SetMx999(mx999)
								inj.SetIsFromClipboard(true)
								c.treatments.SetAddInjection(*inj)
							} else {
								// Injection 형태가 아닌 다른 부분(eswt 등)이라면 일단 기존처럼 메모로 추가
								if strings.HasPrefix(line, "er) focus") {
									lastWasErFocus = true
									c.treatments.AddClipboardMemo(line)
								} else if lastWasErFocus && strings.HasPrefix(line, "radial") {
									if strings.HasPrefix(line, "radial on") {
										c.treatments.AddClipboardMemo("       " + line) // 4+3=7칸
									} else {
										c.treatments.AddClipboardMemo("   " + line) // 3칸
									}
								} else {
									lastWasErFocus = false
									if strings.HasPrefix(line, "radial on") {
										c.treatments.AddClipboardMemo("    " + line)
									} else {
										c.treatments.AddClipboardMemo(line)
									}
								}
							}
						}

						slog.Info("[CLIPBOARD] Parsed to Treatments", "text", text)
					}
					continue
				}

				// MuteStart message processing - TypeStr 출력 중 버퍼 추가 차단 시작
				if stuff.Do == "MuteStart" {
					c.isMuting = true
					slog.Debug("[BUFFER] muting started")
					continue
				}

				// ClearBuffer message processing - 버퍼 초기화 및 뮤팅 해제
				if stuff.Do == "ClearBuffer" {
					c.buffer = ""
					c.isMuting = false
					slog.Debug("[BUFFER] cleared and unmuted")
					continue
				}

				// Backspace message processing
				if stuff.Do == "Backspace" {
					if len(c.buffer) > 0 {
						runes := []rune(c.buffer)
						if len(runes) > 0 {
							c.buffer = string(runes[:len(runes)-1])
						}
						slog.Debug("[BUFFER] backspace", "buffer", c.buffer)
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

// isLumbarSpineInj reports whether the injection site is a lumbar spine injection.
func isLumbarSpineInj(site string) bool {
	return strings.Contains(site, "lmbb") ||
		strings.Contains(site, "lfjb") ||
		strings.Contains(site, "lsnrb") ||
		strings.Contains(site, "ldrgb")
}

// isCervicalSpineInj reports whether the injection site is a cervical spine injection.
func isCervicalSpineInj(site string) bool {
	return strings.Contains(site, "cmbb") ||
		strings.Contains(site, "cfjb") ||
		strings.Contains(site, "csnrb") ||
		strings.Contains(site, "cdrgb")
}

// isUpperLimbInj reports whether the injection site is an upper limb injection.
// (cp, sh, eb, wr, fi, ic 계열)
func isUpperLimbInj(site string) bool {
	return site == "cpb" ||
		strings.Contains(site, "shoulder") ||
		site == "ssnb" ||
		site == "anb" ||
		strings.Contains(site, "elbow") ||
		strings.Contains(site, "cft") ||
		strings.Contains(site, "cet") ||
		strings.Contains(site, "wrist") ||
		site == "apl" ||
		site == "mnb" ||
		site == "tfcc" ||
		site == "hand" ||
		site == "finger" ||
		strings.Contains(site, "intercostal")
}

// isLowerLimbInj reports whether the injection site is a lower limb injection.
// (ql, pc, kn, sc, sr, ft, st, ak 계열)
func isLowerLimbInj(site string) bool {
	return strings.Contains(site, "quadratus") ||
		site == "pcb" ||
		site == "hip" ||
		site == "piriformis" ||
		strings.Contains(site, "knee") ||
		strings.Contains(site, "아라간") ||
		strings.Contains(site, "시노비안") ||
		site == "snrb" ||
		strings.Contains(site, "subtalar") ||
		strings.Contains(site, "ankle") ||
		strings.Contains(site, "atfl") ||
		strings.Contains(site, "achilles") ||
		site == "foot"
}

// processBuffer scans the buffer and extracts matches sequentially by position
func (c *HotstringController) processBuffer(isClipboard bool, isCtrlEnter bool) *models.OutputStuff {
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

	isFirstMeetingMode := strings.HasPrefix(c.buffer, "z")
	isXrayMode := strings.HasPrefix(c.buffer, "x")
	// s로 시작하더라도 K_SIMPLE_CODE 키가 버퍼 앞에서부터 매칭되면 소노 모드가 아닌 일반 모드로 처리
	isSonoMode := strings.HasPrefix(c.buffer, "s")
	if isSonoMode {
		for k := range hotstrings.K_SIMPLE_CODE {
			if strings.HasPrefix(c.buffer, k) {
				isSonoMode = false
				break
			}
		}
	}

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
		// 일반 모드: K_Simples, K_SIMPLE_CODE, 모든 다른 맵 검색
		// 1. K_Simples 매치 찾기
		for k, v := range hotstrings.K_Simples {
			if idx := strings.Index(c.buffer, k); idx != -1 {
				matches = append(matches, Match{idx, len(k), "Simples", k, k, v})
			}
		}

		// 2. K_SIMPLE_CODE 매치 찾기
		for k, v := range hotstrings.K_SIMPLE_CODE {
			if idx := strings.Index(c.buffer, k); idx != -1 {
				matches = append(matches, Match{idx, len(k), "SimpleCode", k, k, v})
			}
		}

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

		// 9. FollowUp (f + 숫자 / ff / f6m / fc / fa)
		followUpRegex := regexp.MustCompile(`f(6m|\d+|f|c|a|o)`)
		fuMatches := followUpRegex.FindAllStringSubmatchIndex(c.buffer, -1)
		for _, fm := range fuMatches {
			fullStr := c.buffer[fm[0]:fm[1]]
			valStr := c.buffer[fm[2]:fm[3]]
			matches = append(matches, Match{
				position: fm[0],
				length:   len(fullStr),
				category: "FollowUp",
				key:      fullStr,
				baseKey:  "f",
				value:    valStr,
			})
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
	deleteCount := 0
	output := models.NewOutputStuff(deleteCount, "", "", "", []string{}, "", "", "")

	// 중복 추가 방지용 추적 맵
	addedInjectionKeys := make(map[string]bool)
	addedSimpleTexts := make(map[string]bool)

	var combinedFirstMeeting *models.FirstMeeting
	hasBlocksMatch := false
	hasCPrefix := false
	hasPPrefix := false
	var etcOrderCodes []string
	var lastBlockBaseKey string
	var lastBlockInjection *models.Injection

	// 다음 매치의 시작 위치 집합: suffix 루프에서 다음 매치를 잘못 소비하지 않도록
	matchStartPositions := make(map[int]bool)
	for _, m := range matches {
		matchStartPositions[m.position] = true
	}

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
					// 현재 위치가 다른 매치의 시작 위치라면 이 match의 suffix 처리를 중단
					if matchStartPositions[remainderPos] {
						break
					}
					remainder := c.buffer[remainderPos:]
					matchedSuffix := false

					if strings.HasPrefix(remainder, "ef") {
						if eswtVal, exists := hotstrings.K_ESWT_ONLY[match.baseKey]; exists {
							slog.Debug("Hotstring Triggered (ESWT-ef Suffix)", "trigger", match.key+"ef", "baseKey", match.baseKey)
							newEswt := *eswtVal
							newEswt.SetFeeType(constants.K_FREE)
							c.treatments.SetESWT(newEswt)
							processed[remainderPos] = true
							processed[remainderPos+1] = true
							remainderPos += 2
							matchedSuffix = true
						} else if inj, ok := match.value.(*models.Injection); ok && inj.GetEswtFocus() != "" {
							eswt := models.NewESWT(inj.GetDirection(), inj.GetEswtFocus(), inj.GetEswtRadial(), constants.K_FREE, false)
							c.treatments.SetESWT(*eswt)
							processed[remainderPos] = true
							processed[remainderPos+1] = true
							remainderPos += 2
							matchedSuffix = true
						}
					} else if strings.HasPrefix(remainder, "er") {
						if eswtVal, exists := hotstrings.K_ESWT_ONLY[match.baseKey]; exists {
							slog.Debug("Hotstring Triggered (ESWT-er Suffix)", "trigger", match.key+"er", "baseKey", match.baseKey)
							newEswt := *eswtVal
							newEswt.SetFeeType(constants.K_FREE_RADIAL_ONLY)
							c.treatments.SetESWT(newEswt)
							processed[remainderPos] = true
							processed[remainderPos+1] = true
							remainderPos += 2
							matchedSuffix = true
						} else if inj, ok := match.value.(*models.Injection); ok && inj.GetEswtFocus() != "" {
							eswt := models.NewESWT(inj.GetDirection(), inj.GetEswtFocus(), inj.GetEswtRadial(), constants.K_FREE_RADIAL_ONLY, false)
							c.treatments.SetESWT(*eswt)
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
						} else if inj, ok := match.value.(*models.Injection); ok && inj.GetEswtFocus() != "" {
							eswt := models.NewESWT(inj.GetDirection(), inj.GetEswtFocus(), inj.GetEswtRadial(), "normal", false)
							c.treatments.SetESWT(*eswt)
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
						} else if inj, ok := match.value.(*models.Injection); ok && inj.GetSonoStim() != "" {
							sntCode := ".+999_s"
							if inj.GetDirection() == "Both" {
								sntCode = ".+999_sb"
							}
							snt := models.NewSonoStim(inj.GetDirection(), inj.GetSonoStim(), sntCode)
							c.treatments.SetAddExtraTreatments(*snt)
							processed[remainderPos] = true
							remainderPos += 1
							matchedSuffix = true
						}
					} else if remainder[0] == 'p' || remainder[0] == 'c' {
						// 'p'(isP 속성)나 'c'(caudal) 문자가 중간에 끼어 있어도 뒤의 leftover 루프에서 처리할 수 있도록, 무시하고 다음 접미사 탐색을 계속함
						remainderPos += 1
						matchedSuffix = true
					} else if strings.HasPrefix(remainder, "pe") {
						// pe(PainEraser)는 e/ef/er/s/p 검색이 모두 끝난 후 마지막으로 검색
						if peVal, exists := hotstrings.K_PainEraser[match.baseKey]; exists {
							slog.Debug("Hotstring Triggered (PainEraser Suffix)", "trigger", match.key+"pe", "baseKey", match.baseKey)
							c.treatments.SetAddExtraTreatments(*peVal)
							processed[remainderPos] = true
							processed[remainderPos+1] = true
							remainderPos += 2
							matchedSuffix = true
						} else if inj, ok := match.value.(*models.Injection); ok && inj.GetEswtFocus() != "" {
							pe := models.NewPainEraser(inj.GetDirection(), inj.GetEswtFocus(), "pe0")
							c.treatments.SetAddExtraTreatments(*pe)
							processed[remainderPos] = true
							processed[remainderPos+1] = true
							remainderPos += 2
							matchedSuffix = true
						}
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
						output.SetDrugCode(oc)
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
				injKey := injection.GetDirection() + "|" + injection.GetSite()
				if addedInjectionKeys[injKey] {
					slog.Error("[DUPLICATE] injection already added", "key", injKey, "buffer", c.buffer)
					output.SetErrorMsg("중복된 항목: " + injKey)
				} else {
					addedInjectionKeys[injKey] = true
					c.treatments.SetAddInjection(*injection)
					// etc 필드 처리: *Injection 이면 추가 주사, string 이면 오더코드
					switch etcVal := injection.GetEtc().(type) {
					case *models.Injection:
						etcKey := etcVal.GetDirection() + "|" + etcVal.GetSite()
						if addedInjectionKeys[etcKey] {
							slog.Error("[DUPLICATE] etc injection already added", "key", etcKey, "buffer", c.buffer)
							output.SetErrorMsg("중복된 항목: " + etcKey)
						} else {
							slog.Debug("Hotstring Triggered (Blocks etc)", "trigger", match.key, "site", etcVal.GetSite())
							addedInjectionKeys[etcKey] = true
							c.treatments.SetAddInjection(*etcVal)
							// 중첩 injection의 etc 코드도 확인
							if nestedCode, ok := etcVal.GetEtc().(string); ok {
								etcOrderCodes = append(etcOrderCodes, nestedCode)
								switch nestedCode {
								case ".+999_pf_0":
									hasCPrefix = true
								case ".+999_pm_0":
									hasPPrefix = true
								}
							}
						}
					case string:
						etcOrderCodes = append(etcOrderCodes, etcVal)
						switch etcVal {
						case ".+999_pf_0":
							hasCPrefix = true
						case ".+999_pm_0":
							hasPPrefix = true
						}
					}
				}
				hasBlocksMatch = true
				lastBlockBaseKey = match.baseKey
				lastBlockInjection = injection
			case "Simples":
				slog.Debug("Hotstring Triggered (Simples)", "trigger", match.key)
				simple, ok := match.value.(*models.SimpleInput)
				if ok {
					if addedSimpleTexts[simple.GetText()] {
						slog.Error("[DUPLICATE] simple text already added", "text", simple.GetText(), "buffer", c.buffer)
						output.SetErrorMsg("중복된 항목: " + simple.GetText())
					} else {
						addedSimpleTexts[simple.GetText()] = true
						curSimple := output.GetSimpleText()
						if curSimple != "" {
							output.SetSimpleText(curSimple + ", " + simple.GetText())
						} else {
							output.SetSimpleText(simple.GetText())
						}
						if simple.GetExtraDo() != "" {
							output.SetExtraDo(simple.GetExtraDo())
						}
					}
				}
			case "SimpleCode":
				slog.Debug("Hotstring Triggered (SimpleCode)", "trigger", match.key)
				strCode, ok := match.value.(string)
				if ok {
					output.AddOrderCode(strCode)
				}
			case "FollowUp":
				slog.Debug("Hotstring Triggered (FollowUp)", "trigger", match.key, "value", match.value)
				val, ok := match.value.(string)
				if ok {
					c.treatments.SetFollowUp(val)
				}
			}
		}
	}
	// 모든 매치가 끝난 후 처리되지 않은 문자 중 'c'가 있으면 'caudal'로 처리, 'p'가 남으면 모든 injection을 isP = true로 변경
	// Blocks 매치가 있을 때만 실행 (simple 입력만 있을 때 불필요한 caudal/isP 트리거 방지)
	leftoverP := false
	if hasBlocksMatch {
		for i := 0; i < len(c.buffer); i++ {
			if !processed[i] {
				// 'pe' 2글자 체크를 'p' 단독보다 먼저
				if i+1 < len(c.buffer) && !processed[i+1] && c.buffer[i] == 'p' && c.buffer[i+1] == 'e' {
					if lastBlockInjection != nil {
						if peVal, exists := hotstrings.K_PainEraser[lastBlockBaseKey]; exists {
							c.treatments.SetAddExtraTreatments(*peVal)
						} else if lastBlockInjection.GetEswtFocus() != "" {
							pe := models.NewPainEraser(lastBlockInjection.GetDirection(), lastBlockInjection.GetEswtFocus(), "pe0")
							c.treatments.SetAddExtraTreatments(*pe)
						}
					}
					processed[i] = true
					processed[i+1] = true
					i++
				} else if c.buffer[i] == 'c' {
					if caudalVal, exists := hotstrings.K_Blocks["caudal"]; exists {
						caudalKey := caudalVal.GetDirection() + "|" + caudalVal.GetSite()
						if addedInjectionKeys[caudalKey] {
							slog.Error("[DUPLICATE] caudal already added", "buffer", c.buffer)
							output.SetErrorMsg("중복된 항목: caudal")
						} else {
							slog.Debug("Hotstring Triggered (Leftover 'c' -> caudal)", "trigger", "caudal", "site", caudalVal.GetSite())
							addedInjectionKeys[caudalKey] = true
							c.treatments.SetAddInjection(*caudalVal)
						}
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
				} else if c.buffer[i] == 'e' {
					if i+1 < len(c.buffer) && !processed[i+1] && c.buffer[i+1] == 'f' {
						// ef 잔여 → K_FREE (ef) focus+radial, 코드 .+999_ef
						if lastBlockInjection != nil {
							if eswtVal, exists := hotstrings.K_ESWT_ONLY[lastBlockBaseKey]; exists {
								newEswt := *eswtVal
								newEswt.SetFeeType(constants.K_FREE)
								c.treatments.SetESWT(newEswt)
							} else if lastBlockInjection.GetEswtFocus() != "" {
								eswt := models.NewESWT(lastBlockInjection.GetDirection(), lastBlockInjection.GetEswtFocus(), lastBlockInjection.GetEswtRadial(), constants.K_FREE, false)
								c.treatments.SetESWT(*eswt)
							}
						}
						processed[i] = true
						processed[i+1] = true
						i++
					} else if i+1 < len(c.buffer) && !processed[i+1] && c.buffer[i+1] == 'r' {
						// er 잔여 → K_FREE_RADIAL_ONLY (ef) radial only, 코드 .+999_ef
						if lastBlockInjection != nil {
							if eswtVal, exists := hotstrings.K_ESWT_ONLY[lastBlockBaseKey]; exists {
								newEswt := *eswtVal
								newEswt.SetFeeType(constants.K_FREE_RADIAL_ONLY)
								c.treatments.SetESWT(newEswt)
							} else if lastBlockInjection.GetEswtFocus() != "" {
								eswt := models.NewESWT(lastBlockInjection.GetDirection(), lastBlockInjection.GetEswtFocus(), lastBlockInjection.GetEswtRadial(), constants.K_FREE_RADIAL_ONLY, false)
								c.treatments.SetESWT(*eswt)
							}
						}
						processed[i] = true
						processed[i+1] = true
						i++
					} else {
						// e 단독 → K_NORMAL (e) focus+radial
						if lastBlockInjection != nil {
							if eswtVal, exists := hotstrings.K_ESWT_ONLY[lastBlockBaseKey]; exists {
								c.treatments.SetESWT(*eswtVal)
							} else if lastBlockInjection.GetEswtFocus() != "" {
								eswt := models.NewESWT(lastBlockInjection.GetDirection(), lastBlockInjection.GetEswtFocus(), lastBlockInjection.GetEswtRadial(), "normal", false)
								c.treatments.SetESWT(*eswt)
							}
						}
						processed[i] = true
					}
				} else if c.buffer[i] == 's' {
					if lastBlockInjection != nil {
						sntCode := ".+999_s"
						if lastBlockInjection.GetDirection() == "Both" {
							sntCode = ".+999_sb"
						}
						if sntVal, exists := hotstrings.K_SonoStim[lastBlockBaseKey]; exists {
							c.treatments.SetAddExtraTreatments(*sntVal)
						} else if lastBlockInjection.GetSonoStim() != "" {
							snt := models.NewSonoStim(lastBlockInjection.GetDirection(), lastBlockInjection.GetSonoStim(), sntCode)
							c.treatments.SetAddExtraTreatments(*snt)
						}
					}
					processed[i] = true
				}
			}
		}
	}

	if leftoverP {
		c.treatments.SetAllInjectionsIsP(true)
	}

	// 모드 프리픽스 문자(z/x/s)는 별도로 processed에 표시
	if isFirstMeetingMode || isXrayMode || isSonoMode {
		processed[0] = true
	}

	// 매칭되지 않은 문자가 있으면 트리거 전체 취소 (100% 매칭이 되어야 출력)
	var unmatched strings.Builder
	for i := 0; i < len(c.buffer); i++ {
		if !processed[i] {
			unmatched.WriteByte(c.buffer[i])
		}
	}
	if unmatched.Len() > 0 {
		slog.Error("Hotstring trigger cancelled: unmatched characters",
			"trigger", c.buffer,
			"unmatched", unmatched.String(),
		)
		return models.NewOutputStuff(0, "", "", "", []string{}, "", "", "")
	}

	if isFirstMeetingMode && combinedFirstMeeting != nil {
		re := regexp.MustCompile(`f(\d+[dmyw]?)`)
		durationMatches := re.FindStringSubmatch(c.buffer)
		if len(durationMatches) > 1 {
			combinedFirstMeeting.SetDuration(durationMatches[1])
		}
	}

	if isFirstMeetingMode {
		output.SetSkipChartEnter(true)
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

		// standalone f/u(주사·ESWT·extraTreatments 없이 followUp만 있는 경우)는
		// chartWindow가 아닌 현재 커서 위치에 바로 입력 (simpleText)
		if c.treatments.HasOnlyFollowUp() {
			fuText := c.treatments.GetStandaloneFollowUpText()
			curSimple := output.GetSimpleText()
			if curSimple != "" {
				output.SetSimpleText(curSimple + "\n" + fuText)
			} else {
				output.SetSimpleText(fuText)
			}
		} else {
			chartText := output.GetChartText()
			newChartText := c.treatments.GetTextForChart()
			if newChartText != "" {
				if chartText != "" {
					output.SetChartText(chartText + "\n" + newChartText)
				} else {
					output.SetChartText(newChartText)
				}
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
		for _, ec := range etcOrderCodes {
			output.AddOrderCode(ec)
		}

		drug := c.treatments.GetDrug()
		if drug != "" {
			output.SetDrug(drug)
		}
	}

	// etc 코드가 .+999_pf_0 이면 "pt) 도수프리\n    자기장\n" 를 f/u) 앞에 삽입
	if hasCPrefix {
		ct := output.GetChartText()
		insert := "pt) 도수프리\n     자기장\n"
		if strings.Contains(ct, "f/u)") {
			ct = strings.Replace(ct, "f/u)", insert+"f/u)", 1)
		} else if ct != "" {
			ct += insert
		} else {
			ct = insert
		}
		output.SetChartText(ct)
	}

	// etc 코드가 .+999_pm_0 이면 "pt) 자기장\n" 를 f/u) 앞에 삽입
	if hasPPrefix {
		ct := output.GetChartText()
		insert := "pt) 자기장\n"
		if strings.Contains(ct, "f/u)") {
			ct = strings.Replace(ct, "f/u)", insert+"f/u)", 1)
		} else if ct != "" {
			ct += insert
		} else {
			ct = insert
		}
		output.SetChartText(ct)
	}

	// ctrl+enter 트리거 시 주사 조합 이상 여부 경고
	if isCtrlEnter {
		var realInj []models.Injection
		for _, inj := range c.treatments.GetTreatments() {
			if !inj.GetIsFromClipboard() {
				realInj = append(realInj, inj)
			}
		}
		if len(realInj) >= 2 {
			first := realInj[0]
			second := realInj[1]
			var warnings []string

			// Rule 1: 요추 치료 + 상지 주사
			hasLumbar := isLumbarSpineInj(first.GetSite()) || isLumbarSpineInj(second.GetSite())
			hasUpper := isUpperLimbInj(first.GetSite()) || isUpperLimbInj(second.GetSite())
			if hasLumbar && hasUpper {
				warnings = append(warnings, "⚠ 요추치료 + 상지주사 조합 경고")
				warnings = append(warnings, "⚠ 처방을 다시 확인하세요")
			}

			// Rule 2: 경추 치료 + 하지 주사
			hasCervical := isCervicalSpineInj(first.GetSite()) || isCervicalSpineInj(second.GetSite())
			hasLower := isLowerLimbInj(first.GetSite()) || isLowerLimbInj(second.GetSite())
			if hasCervical && hasLower {
				warnings = append(warnings, "⚠ 경추치료 + 하지주사 조합 경고")
				warnings = append(warnings, "⚠ 처방을 다시 확인하세요")
			}

			// Rule 3: 첫번째와 두번째 주사 방향 불일치
			firstDir := first.GetDirection()
			secondDir := second.GetDirection()
			if firstDir != "" && secondDir != "" &&
				firstDir != "Both" && secondDir != "Both" &&
				firstDir != secondDir {
				warnings = append(warnings, "⚠ 주사 방향 불일치 경고")
				warnings = append(warnings, "⚠ "+firstDir+" vs "+secondDir)
			}

			if len(warnings) > 0 {
				ct := output.GetChartText()
				if ct != "" {
					output.SetChartText(ct + strings.Join(warnings, "\n") + "\n")
				}
			}
		}
	}

	// 클립보드 트리거(Ctrl+*) 플래그 설정
	if isClipboard {
		output.SetIsClipboard(true)
	}

	// 매칭된 글자 수 + 트리거 키 1 = deleteCount
	if !isClipboard {
		deleteCount = len(processed)
		if deleteCount > 0 {
			deleteCount += 1 // 트리거 키
			// z/x/s prefix는 processed에 포함되지 않으므로 추가
			if isFirstMeetingMode || isXrayMode || isSonoMode {
				deleteCount += 1
			}
		}
		if deleteCount > 20 {
			deleteCount = 20
		}
		output.SetDeleteHostring(deleteCount)
	}

	return output
}
