package controllers

import (
	"log/slog"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/seeseasdk/go_hotstring_v3/data/constants"
	"github.com/seeseasdk/go_hotstring_v3/data/hotstrings"
	"github.com/seeseasdk/go_hotstring_v3/data/models"
)

type HotstringController struct {
	cc                 *ChannelController
	buffer             string
	treatments         *models.Treatments
	isMuting           bool   // TypeStr 출력 중 버퍼 추가 차단
	clipboardChartText string // Ctrl+* 트리거시 f/u) 날짜 갱신된 chartText 임시 저장
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

					// Ctrl+Enter 트리거인 경우: 이전 트리거의 treatments가 ResetChan을 아직 받지
					// 못한 상태에서 다시 트리거되면 누적되어 이중 출력이 발생하므로 즉시 초기화
					isCtrlEnter := !isClipboard
					if isCtrlEnter {
						c.treatments = models.NewTreatments()
					}

					// 트리거 시점의 버퍼 내용을 디버그 로그에 기록
					slog.Debug("HotstringController: Triggered", "buffer", c.buffer)

					// Trigger processing of the accumulated buffer
					output := c.processBuffer(isClipboard, isCtrlEnter)

					// Ctrl+* 트리거에서 미리 계산된 chartText가 있으면 output에 설정
					if c.clipboardChartText != "" {
						existing := output.GetChartText()
						if existing != "" {
							output.SetChartText(existing + "\n" + c.clipboardChartText)
						} else {
							output.SetChartText(c.clipboardChartText)
						}
						c.clipboardChartText = ""
					}

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
						lastDirection := ""
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

							// f/u) 라인은 주사가 아니고 specificText에도 포함하지 않으므로 skip
							if strings.HasPrefix(line, "f/u)") {
								continue
							}

							// e) 나 ef) 혹은 snt) 나 pe) 나 pt) 로 시작하는 건 주사가 아니므로 제외
							if strings.HasPrefix(line, "e) ") || strings.HasPrefix(line, "ef) ") || strings.HasPrefix(line, "snt) ") || strings.HasPrefix(line, "pe) ") || strings.HasPrefix(line, "pt) ") {
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
								isDP := false
								if strings.HasSuffix(line, " dp") {
									isDP = true
									line = strings.TrimSuffix(line, " dp")
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

								// caudal 라인은 방향이 없으므로 lastDirection(Rt./Lt.)을 상속
								if site == "caudal" && direction == "" &&
									(lastDirection == constants.K_RT || lastDirection == constants.K_LT) {
									direction = lastDirection
								}

								// direction이 새로 추출된 경우 lastDirection 갱신
								if direction != "" {
									lastDirection = direction
								}

								// site prefix 정규화: "mbb/fjb/snrb/drgb C/L/T..." → "c/l/t + prefix..."
								lowerSite := strings.ToLower(site)
								for _, pfx := range []string{"mbb", "fjb", "snrb", "drgb"} {
									if strings.HasPrefix(lowerSite, pfx+" c") {
										site = "c" + site
										break
									} else if strings.HasPrefix(lowerSite, pfx+" l") {
										site = "l" + site
										break
									} else if strings.HasPrefix(lowerSite, pfx+" t") {
										site = "t" + site
										break
									}
								}

								// 매칭되는 코드를 K_Blocks에서 검색 (정확히 일치하거나, site가 "known_site " 로 시작하는 경우)
								code := ""
								mx999 := site
								for _, v := range hotstrings.K_Blocks {
									siteName := v.GetSite()
									if strings.EqualFold(siteName, site) || strings.HasPrefix(strings.ToLower(site), strings.ToLower(siteName)+" ") {
										code = v.GetCode()
										mx999 = v.GetMx999()
										break
									}
								}

								// Injection 생성
								inj := models.NewInjection(direction, site, code, "", "", isWithCarm, isP, isDP, isN, nil)
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

				// SetClipboardChartText: Ctrl+* 트리거에서 f/u) 날짜가 갱신된 chartText를 저장
				if stuff.Do == "SetClipboardChartText" {
					if txt, ok := stuff.Object.(string); ok {
						c.clipboardChartText = txt
					}
					continue
				}

				// MuteStart message processing - TypeStr 출력 중 버퍼 추가 차단 시작
				if stuff.Do == "MuteStart" {
					c.isMuting = true
					continue
				}

				// ClearBuffer message processing - 버퍼 초기화 및 뮤팅 해제
				if stuff.Do == "ClearBuffer" {
					c.buffer = ""
					c.isMuting = false
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

	// 0. K_Drugs 매치 찾기: K_Drugs 키를 길이 내림차순 정렬해 regex 동적 생성 (긴 키 우선 매치)
	drugKeys := make([]string, 0, len(hotstrings.K_Drugs))
	for k := range hotstrings.K_Drugs {
		drugKeys = append(drugKeys, k)
	}
	sort.Slice(drugKeys, func(i, j int) bool {
		return len(drugKeys[i]) > len(drugKeys[j])
	})
	drugRegex := regexp.MustCompile(`(` + strings.Join(drugKeys, "|") + `)(\d+)`)
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
	// m으로 시작하되 K_Simples/K_SIMPLE_CODE 키가 먼저 매칭되면 일반 모드로 처리
	isManualMode := strings.HasPrefix(c.buffer, "m")
	if isManualMode {
		for k := range hotstrings.K_SIMPLE_CODE {
			if strings.HasPrefix(c.buffer, k) {
				isManualMode = false
				break
			}
		}
	}
	if isManualMode {
		for k := range hotstrings.K_Simples {
			if strings.HasPrefix(c.buffer, k) {
				isManualMode = false
				break
			}
		}
	}
	// e로 시작하더라도 K_SIMPLE_CODE, K_Simples 키가 버퍼 앞에서부터 매칭되면 일반 모드로 처리
	isEswtMode := strings.HasPrefix(c.buffer, "e")
	if isEswtMode {
		for k := range hotstrings.K_SIMPLE_CODE {
			if strings.HasPrefix(c.buffer, k) {
				isEswtMode = false
				break
			}
		}
	}
	if isEswtMode {
		for k := range hotstrings.K_Simples {
			if strings.HasPrefix(c.buffer, k) {
				isEswtMode = false
				break
			}
		}
	}

	if isFirstMeetingMode {
		// z로 시작하면 나머지 문자열에서는 K_FirstMeeting + K_Simples을 연속으로 찾는다 (예: zcvbshb -> cvb, shb)
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
		// z모드에서도 K_Simples 검색 (mm, hh, dd 등 → pmhx 처리)
		for k, v := range hotstrings.K_Simples {
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
				matches = append(matches, Match{actualPos, len(k), "Simples", k, k, v})
				idx = actualPos + len(k)
			}
		}
	} else if isXrayMode {
		for k, v := range hotstrings.K_Xrays {
			// xray 모드에서는 's'로 끝나는 키는 매칭하지 않음 (trailing 's'로 일괄 처리)
			if strings.HasSuffix(k, "s") {
				continue
			}
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
		// xrayMode에서도 "s"+key 패턴의 K_Sonos 검색 허용 (예: xshrsshr → xray shr + sono shr)
		for k, v := range hotstrings.K_Sonos {
			target := "s" + k
			idx := 1 // 'x' 이후부터 검색
			for {
				if idx >= len(c.buffer) {
					break
				}
				foundIdx := strings.Index(c.buffer[idx:], target)
				if foundIdx == -1 {
					break
				}
				actualPos := idx + foundIdx
				matches = append(matches, Match{actualPos, len(target), "Sonos", target, k, v})
				idx = actualPos + len(target)
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
	} else if isManualMode {
		// m으로 시작하면 K_Manual에서 전체 키(m8, m13 등)로 검색
		for k, v := range hotstrings.K_Manual {
			if idx := strings.Index(c.buffer, k); idx != -1 {
				matches = append(matches, Match{idx, len(k), "Manual", k, k, v})
			}
		}
	} else if isEswtMode {
		// e로 시작하면 K_ESWT_ONLY에서만 찾는다 (연속 다부위 지원, 예: elvbshls → lvb eswt + shl eswt)
		for k, v := range hotstrings.K_ESWT {
			idx := 1 // 'e' 이후부터 검색
			for {
				if idx >= len(c.buffer) {
					break
				}
				foundIdx := strings.Index(c.buffer[idx:], k)
				if foundIdx == -1 {
					break
				}
				actualPos := idx + foundIdx
				matches = append(matches, Match{actualPos, len(k), "ESWT_MODE", k, k, v})
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
		for k, v := range hotstrings.K_ESWT {
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

	// Blocks 매치가 있을 때 p+숫자 패턴의 Simples 매치(예: p7 → "PRS 7")는 제거:
	// 이 경우 'p'는 leftoverP(isP=true), 숫자는 leftover 숫자로 처리되어야 함
	hasBlocksInMatches := false
	for _, m := range matches {
		if m.category == "Blocks-Direct" || m.category == "Blocks-C" || m.category == "Blocks-P" {
			hasBlocksInMatches = true
			break
		}
	}
	if hasBlocksInMatches {
		pDigitRe := regexp.MustCompile(`^p\d+$`)
		filtered := matches[:0]
		for _, m := range matches {
			if m.category == "Simples" && pDigitRe.MatchString(m.key) {
				slog.Debug("Skipping p+digit Simples match due to Blocks context", "key", m.key)
				continue
			}
			filtered = append(filtered, m)
		}
		matches = filtered
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

	// ESWT 금지 부위 경고 메시지를 모아서 chartText 맨 마지막에 추가
	var eswtForbiddenWarnings []string

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
	var specificSonoStimInj *models.Injection // interscapular 등 specific sonostim을 가진 injection (cm5/cf5 등)
	var matchedXrayBaseKeys []string
	var matchedXrayValues []*models.Xray
	var matchedEswtBaseKeys []string
	var matchedFirstMeetingBaseKeys []string

	// 다음 매치의 시작 위치 집합: suffix 루프에서 다음 매치를 잘못 소비하지 않도록
	matchStartPositions := make(map[int]bool)
	for _, m := range matches {
		matchStartPositions[m.position] = true
	}

	// peri-p 컨텍스트 감지: 첫 번째 Blocks 매치의 키가 'p'로 시작하고 isWithCarm=false이면 true
	// 이 경우 이후 isWithCarm=true 인젝션은 p-prefix 버전으로 대체하거나 isWithCarm=false로 강제,
	// etc(추가 주사)의 isWithCarm=true는 건너뜀
	isPeriPContext := false
	for _, m := range matches {
		if m.category == "Blocks-Direct" || m.category == "Blocks-C" || m.category == "Blocks-P" {
			if inj, ok := m.value.(*models.Injection); ok {
				if strings.HasPrefix(m.key, "p") && !inj.GetIsWithCarm() {
					isPeriPContext = true
				}
			}
			break
		}
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
			// ESWT_MODE는 trailing 's'로 trailing sonostim을 일괄 처리하므로 suffix loop 제외
			if match.category != "Sonos" && match.category != "Xrays" && match.category != "FirstMeeting" && match.category != "ESWT_MODE" {
				remainderPos := match.position + match.length
				for remainderPos < len(c.buffer) && !processed[remainderPos] {
					// 현재 위치가 다른 매치의 시작 위치라면 이 match의 suffix 처리를 중단
					if matchStartPositions[remainderPos] {
						break
					}
					remainder := c.buffer[remainderPos:]
					matchedSuffix := false

					if strings.HasPrefix(remainder, "ef") && !matchStartPositions[remainderPos+1] {
						if inj, ok := match.value.(*models.Injection); ok && inj.GetEswtKey() == "" {
							eswtForbiddenWarnings = append(eswtForbiddenWarnings, "★★★ "+match.baseKey+" eswt는 금지 부위 입니다")
							processed[remainderPos] = true
							processed[remainderPos+1] = true
							remainderPos += 2
							matchedSuffix = true
						} else if eswtVal, exists := hotstrings.K_ESWT[match.baseKey]; exists {
							slog.Debug("Hotstring Triggered (ESWT-ef Suffix)", "trigger", match.key+"ef", "baseKey", match.baseKey)
							newEswt := *eswtVal
							newEswt.SetFeeType(constants.K_FREE)
							c.treatments.SetESWT(newEswt)
							processed[remainderPos] = true
							processed[remainderPos+1] = true
							remainderPos += 2
							matchedSuffix = true
						} else if inj, ok := match.value.(*models.Injection); ok && inj.GetEswtKey() != "" {
							useInj := inj
							injEswt := hotstrings.K_ESWT[inj.GetEswtKey()]
							if injEswt != nil && injEswt.GetFocus() == injEswt.GetRadial() && lastBlockInjection != nil && lastBlockInjection.GetEswtKey() != "" {
								lastEswt := hotstrings.K_ESWT[lastBlockInjection.GetEswtKey()]
								if lastEswt != nil && lastEswt.GetFocus() != lastEswt.GetRadial() {
									useInj = lastBlockInjection
									injEswt = lastEswt
								}
							}
							if injEswt != nil {
								eswt := models.NewESWT(useInj.GetDirection(), injEswt.GetFocus(), injEswt.GetRadial(), constants.K_FREE, false, injEswt.GetAddCode())
								c.treatments.SetESWT(*eswt)
								processed[remainderPos] = true
								processed[remainderPos+1] = true
								remainderPos += 2
								matchedSuffix = true
							}
						} else if _, ok := match.value.(*models.Injection); ok {
							eswtForbiddenWarnings = append(eswtForbiddenWarnings, "★★★ "+match.baseKey+" eswt는 금지 부위 입니다")
							processed[remainderPos] = true
							processed[remainderPos+1] = true
							remainderPos += 2
							matchedSuffix = true
						}
					} else if strings.HasPrefix(remainder, "er") && !matchStartPositions[remainderPos+1] {
						if inj, ok := match.value.(*models.Injection); ok && inj.GetEswtKey() == "" {
							eswtForbiddenWarnings = append(eswtForbiddenWarnings, "★★★ "+match.baseKey+" eswt는 금지 부위 입니다")
							processed[remainderPos] = true
							processed[remainderPos+1] = true
							remainderPos += 2
							matchedSuffix = true
						} else if eswtVal, exists := hotstrings.K_ESWT[match.baseKey]; exists {
							slog.Debug("Hotstring Triggered (ESWT-er Suffix)", "trigger", match.key+"er", "baseKey", match.baseKey)
							newEswt := *eswtVal
							newEswt.SetFeeType(constants.K_FREE_RADIAL_ONLY)
							c.treatments.SetESWT(newEswt)
							processed[remainderPos] = true
							processed[remainderPos+1] = true
							remainderPos += 2
							matchedSuffix = true
						} else if inj, ok := match.value.(*models.Injection); ok && inj.GetEswtKey() != "" {
							useInj := inj
							injEswt := hotstrings.K_ESWT[inj.GetEswtKey()]
							if injEswt != nil && injEswt.GetFocus() == injEswt.GetRadial() && lastBlockInjection != nil && lastBlockInjection.GetEswtKey() != "" {
								lastEswt := hotstrings.K_ESWT[lastBlockInjection.GetEswtKey()]
								if lastEswt != nil && lastEswt.GetFocus() != lastEswt.GetRadial() {
									useInj = lastBlockInjection
									injEswt = lastEswt
								}
							}
							if injEswt != nil {
								eswt := models.NewESWT(useInj.GetDirection(), injEswt.GetFocus(), injEswt.GetRadial(), constants.K_FREE_RADIAL_ONLY, false, injEswt.GetAddCode())
								c.treatments.SetESWT(*eswt)
								processed[remainderPos] = true
								processed[remainderPos+1] = true
								remainderPos += 2
								matchedSuffix = true
							}
						} else if _, ok := match.value.(*models.Injection); ok {
							eswtForbiddenWarnings = append(eswtForbiddenWarnings, "★★★ "+match.baseKey+" eswt는 금지 부위 입니다")
							processed[remainderPos] = true
							processed[remainderPos+1] = true
							remainderPos += 2
							matchedSuffix = true
						}
					} else if strings.HasPrefix(remainder, "e") {
						if inj, ok := match.value.(*models.Injection); ok && inj.GetEswtKey() == "" {
							eswtForbiddenWarnings = append(eswtForbiddenWarnings, "★★★ "+match.baseKey+" eswt는 금지 부위 입니다")
							processed[remainderPos] = true
							remainderPos += 1
							matchedSuffix = true
						} else if eswtVal, exists := hotstrings.K_ESWT[match.baseKey]; exists {
							slog.Debug("Hotstring Triggered (ESWT Suffix)", "trigger", match.key+"e", "baseKey", match.baseKey)
							c.treatments.SetESWT(*eswtVal)
							processed[remainderPos] = true
							remainderPos += 1
							matchedSuffix = true
						} else if inj, ok := match.value.(*models.Injection); ok && inj.GetEswtKey() != "" {
							useInj := inj
							injEswt := hotstrings.K_ESWT[inj.GetEswtKey()]
							if injEswt != nil && injEswt.GetFocus() == injEswt.GetRadial() && lastBlockInjection != nil && lastBlockInjection.GetEswtKey() != "" {
								lastEswt := hotstrings.K_ESWT[lastBlockInjection.GetEswtKey()]
								if lastEswt != nil && lastEswt.GetFocus() != lastEswt.GetRadial() {
									useInj = lastBlockInjection
									injEswt = lastEswt
								}
							}
							if injEswt != nil {
								eswt := models.NewESWT(useInj.GetDirection(), injEswt.GetFocus(), injEswt.GetRadial(), "normal", false, injEswt.GetAddCode())
								c.treatments.SetESWT(*eswt)
								processed[remainderPos] = true
								remainderPos += 1
								matchedSuffix = true
							}
						} else if _, ok := match.value.(*models.Injection); ok {
							eswtForbiddenWarnings = append(eswtForbiddenWarnings, "★★★ "+match.baseKey+" eswt는 금지 부위 입니다")
							processed[remainderPos] = true
							remainderPos += 1
							matchedSuffix = true
						}
					} else if strings.HasPrefix(remainder, "s") {
						if inj, ok := match.value.(*models.Injection); ok && inj.GetEswtKey() == "" {
							// eswt 금지 부위는 sonostim도 금지
							processed[remainderPos] = true
							remainderPos += 1
							matchedSuffix = true
						} else if sntVal, exists := hotstrings.K_SonoStim[match.baseKey]; exists {
							slog.Debug("Hotstring Triggered (SonoStim Suffix)", "trigger", match.key+"s", "baseKey", match.baseKey)
							// interscapular 등 specific sonostim이 있으면 generic(TPZ/lower back) 대신 사용
							if specificSonoStimInj != nil && (sntVal.GetSite() == "TPZ" || sntVal.GetSite() == "lower back") {
								sntCode := constants.K_SINGLE_SNT
								if specificSonoStimInj.GetDirection() == constants.K_BOTH {
									specEswt := hotstrings.K_ESWT[specificSonoStimInj.GetEswtKey()]
									if specEswt != nil && specEswt.GetFocus() != "TPZ" && specEswt.GetFocus() != "lower back" {
										sntCode = constants.K_BOTH_SNT
									}
								}
								specSnt := hotstrings.K_SonoStim[specificSonoStimInj.GetSonoStim()]
								if specSnt != nil {
									overrideSnt := models.NewSonoStim(specificSonoStimInj.GetDirection(), specSnt.GetSite(), sntCode)
									c.treatments.SetAddExtraTreatments(*overrideSnt)
								}
							} else {
								c.treatments.SetAddExtraTreatments(*sntVal)
							}
							processed[remainderPos] = true
							remainderPos += 1
							matchedSuffix = true
						} else if inj, ok := match.value.(*models.Injection); ok && inj.GetSonoStim() != "" {
							useInj := inj
							injSnt := hotstrings.K_SonoStim[inj.GetSonoStim()]
							// interscapular 등 specific sonostim이 있으면 generic(TPZ/lower back) 대신 사용
							if specificSonoStimInj != nil && injSnt != nil && (injSnt.GetSite() == "TPZ" || injSnt.GetSite() == "lower back") {
								useInj = specificSonoStimInj
								injSnt = hotstrings.K_SonoStim[useInj.GetSonoStim()]
							}
							if injSnt != nil {
								sntCode := constants.K_SINGLE_SNT
								if useInj.GetDirection() == constants.K_BOTH && (injSnt.GetSite() != "TPZ" && injSnt.GetSite() != "lower back" && injSnt.GetSite() != "interscapular") {
									sntCode = constants.K_BOTH_SNT
								}
								snt := models.NewSonoStim(useInj.GetDirection(), injSnt.GetSite(), sntCode)
								c.treatments.SetAddExtraTreatments(*snt)
								processed[remainderPos] = true
								remainderPos += 1
								matchedSuffix = true
							}
						}
					} else if remainder[0] == 'p' || remainder[0] == 'c' {
						// 'p'(isP 속성)나 'c'(caudal) 문자가 중간에 끼어 있어도 뒤의 leftover 루프에서 처리할 수 있도록, 무시하고 다음 접미사 탐색을 계속함
						remainderPos += 1
						matchedSuffix = true
					} else if strings.HasPrefix(remainder, "pe") && !matchStartPositions[remainderPos+1] {
						// pe(PainEraser)는 e/ef/er/s/p 검색이 모두 끝난 후 마지막으로 검색
						if peVal, exists := hotstrings.K_PainEraser[match.baseKey]; exists {
							slog.Debug("Hotstring Triggered (PainEraser Suffix)", "trigger", match.key+"pe", "baseKey", match.baseKey)
							c.treatments.SetAddExtraTreatments(*peVal)
							processed[remainderPos] = true
							processed[remainderPos+1] = true
							remainderPos += 2
							matchedSuffix = true
						} else if inj, ok := match.value.(*models.Injection); ok && inj.GetEswtKey() != "" {
							injEswt := hotstrings.K_ESWT[inj.GetEswtKey()]
							if injEswt != nil {
								pe := models.NewPainEraser(inj.GetDirection(), injEswt.GetFocus(), "pe0")
								c.treatments.SetAddExtraTreatments(*pe)
								processed[remainderPos] = true
								processed[remainderPos+1] = true
								remainderPos += 2
								matchedSuffix = true
							}
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
			case "ESWT_MODE":
				slog.Debug("Hotstring Triggered (ESWT_MODE)", "trigger", match.key, "baseKey", match.baseKey)
				eswtVal := match.value.(*models.ESWT)
				if len(matchedEswtBaseKeys) == 0 {
					c.treatments.SetESWT(*eswtVal)
				} else {
					c.treatments.SetAddExtraTreatments(*eswtVal)
				}
				matchedEswtBaseKeys = append(matchedEswtBaseKeys, match.baseKey)
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
				matchedFirstMeetingBaseKeys = append(matchedFirstMeetingBaseKeys, match.baseKey)
			case "Xrays":
				slog.Debug("Hotstring Triggered (Xrays)", "trigger", match.key)
				xray, ok := match.value.(*models.Xray)
				if ok {
					if isXrayMode {
						// xray 모드에서는 trailing 's' 확인 후 출력하므로 수집만
						matchedXrayBaseKeys = append(matchedXrayBaseKeys, match.baseKey)
						matchedXrayValues = append(matchedXrayValues, xray)
					} else {
						curChart := output.GetChartText()
						if curChart != "" {
							output.SetChartText(curChart + "\n" + xray.GetText())
						} else {
							output.SetChartText(xray.GetText())
						}
						output.AddOrderCode(xray.GetCode())
						matchedXrayBaseKeys = append(matchedXrayBaseKeys, match.baseKey)
					}
				}
			case "Sonos":
				slog.Debug("Hotstring Triggered (Sonos)", "trigger", match.key)
				sono, ok := match.value.(*models.Sono)
				if ok {
					curSimple := output.GetSimpleText()
					if curSimple != "" {
						output.SetSimpleText(curSimple + "\n" + sono.GetText())
					} else {
						output.SetSimpleText(sono.GetText())
					}
					output.AddOrderCode(sono.GetCode())
				}
			case "Blocks-Direct", "Blocks-C", "Blocks-P":
				injection := match.value.(*models.Injection)

				// isPeriPContext: isWithCarm=true인 injection을 p-prefix 버전으로 대체하거나 isWithCarm=false로 강제
				if isPeriPContext && injection.GetIsWithCarm() {
					if pInj, exists := hotstrings.K_Blocks["p"+match.key]; exists && !pInj.GetIsWithCarm() {
						injection = pInj
					} else {
						injCopy := *injection
						injCopy.SetIsWithCarm(false)
						injection = &injCopy
					}
				}

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
						// isPeriPContext이면 isWithCarm=true인 etc injection은 건너뜀
						if isPeriPContext && etcVal.GetIsWithCarm() {
							// 주사는 건너뛰지만 etc 코드(pf/pm)는 여전히 반영
							switch nestedEtc := etcVal.GetEtc().(type) {
							case string:
								etcOrderCodes = append(etcOrderCodes, nestedEtc)
								switch nestedEtc {
								case ".+999_pf_0":
									hasCPrefix = true
								case ".+999_pm_0":
									hasPPrefix = true
								}
							case []string:
								for _, nestedCode := range nestedEtc {
									etcOrderCodes = append(etcOrderCodes, nestedCode)
									switch nestedCode {
									case ".+999_pf_0":
										hasCPrefix = true
									case ".+999_pm_0":
										hasPPrefix = true
									}
								}
							}
							break
						}
						etcKey := etcVal.GetDirection() + "|" + etcVal.GetSite()
						if addedInjectionKeys[etcKey] {
							slog.Error("[DUPLICATE] etc injection already added", "key", etcKey, "buffer", c.buffer)
							output.SetErrorMsg("중복된 항목: " + etcKey)
						} else {
							slog.Debug("Hotstring Triggered (Blocks etc)", "trigger", match.key, "site", etcVal.GetSite())
							addedInjectionKeys[etcKey] = true
							c.treatments.SetAddInjection(*etcVal)
							// 중첩 injection의 etc 코드도 확인
							switch nestedEtc := etcVal.GetEtc().(type) {
							case string:
								etcOrderCodes = append(etcOrderCodes, nestedEtc)
								switch nestedEtc {
								case ".+999_pf_0":
									hasCPrefix = true
								case ".+999_pm_0":
									hasPPrefix = true
								}
							case []string:
								for _, nestedCode := range nestedEtc {
									etcOrderCodes = append(etcOrderCodes, nestedCode)
									switch nestedCode {
									case ".+999_pf_0":
										hasCPrefix = true
									case ".+999_pm_0":
										hasPPrefix = true
									}
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
					case []string:
						for _, code := range etcVal {
							etcOrderCodes = append(etcOrderCodes, code)
							switch code {
							case ".+999_pf_0":
								hasCPrefix = true
							case ".+999_pm_0":
								hasPPrefix = true
							}
						}
					}
				}
				hasBlocksMatch = true
				lastBlockBaseKey = match.baseKey
				lastBlockInjection = injection
				// interscapular sonostim (cm5/cf5 계열) 기록
				if sntObj := hotstrings.K_SonoStim[injection.GetSonoStim()]; sntObj != nil && sntObj.GetSite() == "interscapular" {
					specificSonoStimInj = injection
				}
			case "Simples":
				slog.Debug("Hotstring Triggered (Simples)", "trigger", match.key)
				simple, ok := match.value.(*models.SimpleInput)
				if ok {
					if addedSimpleTexts[simple.GetText()] {
						slog.Error("[DUPLICATE] simple text already added", "text", simple.GetText(), "buffer", c.buffer)
						output.SetErrorMsg("중복된 항목: " + simple.GetText())
					} else {
						addedSimpleTexts[simple.GetText()] = true
						if isFirstMeetingMode {
							// z모드: pmhx 항목으로 수집 (combinedFirstMeeting이 없으면 나중에 생성)
							if combinedFirstMeeting == nil {
								combinedFirstMeeting = models.NewFirstMeeting([]string{}, "", []models.Xray{}, []any{})
							}
							combinedFirstMeeting.AddPmhxItem(simple.GetText())
							// memoText에도 추가
							curMemo := output.GetMemoText()
							if curMemo != "" {
								output.SetMemoText(curMemo + ", " + simple.GetText())
							} else {
								output.SetMemoText(simple.GetText())
							}
						} else {
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
			case "Manual":
				slog.Debug("Hotstring Triggered (Manual)", "trigger", match.key)
				manual, ok := match.value.(*models.Manual)
				if ok {
					c.treatments.SetManual(*manual)
					output.AddOrderCode(".+999_pt1")
				}
			}
		}
	}
	// 모든 매치가 끝난 후 처리되지 않은 문자 중 'c'가 있으면 'caudal'로 처리, 'p'가 남으면 모든 injection을 isP = true로 변경
	// Blocks 매치가 있을 때만 실행 (simple 입력만 있을 때 불필요한 caudal/isP 트리거 방지)
	leftoverP := false
	leftoverN := false
	if hasBlocksMatch {
		for i := 0; i < len(c.buffer); i++ {
			if !processed[i] {
				// 'pe' 2글자 체크를 'p' 단독보다 먼저
				if i+1 < len(c.buffer) && !processed[i+1] && c.buffer[i] == 'p' && c.buffer[i+1] == 'e' {
					if lastBlockInjection != nil {
						if peVal, exists := hotstrings.K_PainEraser[lastBlockBaseKey]; exists {
							c.treatments.SetAddExtraTreatments(*peVal)
						} else if lastBlockInjection.GetEswtKey() != "" {
							peEswt := hotstrings.K_ESWT[lastBlockInjection.GetEswtKey()]
							if peEswt != nil {
								pe := models.NewPainEraser(lastBlockInjection.GetDirection(), peEswt.GetFocus(), "pe0")
								c.treatments.SetAddExtraTreatments(*pe)
							}
						}
					}
					processed[i] = true
					processed[i+1] = true
					i++
				} else if c.buffer[i] == 'c' {
					if i == 0 {
						// 버퍼 첫 글자 'c'는 isWithCarm 프리픽스 의미 → caudal 없이 소비
						slog.Debug("Hotstring Triggered (Leading 'c' consumed, not caudal)")
						processed[i] = true
					} else if caudalVal, exists := hotstrings.K_Blocks["caudal"]; exists {
						caudalInj := *caudalVal
						if lastBlockInjection != nil &&
							(lastBlockInjection.GetDirection() == constants.K_RT || lastBlockInjection.GetDirection() == constants.K_LT) {
							caudalInj.SetDirection(lastBlockInjection.GetDirection())
						}
						caudalKey := caudalInj.GetDirection() + "|" + caudalInj.GetSite()
						if addedInjectionKeys[caudalKey] {
							slog.Error("[DUPLICATE] caudal already added", "buffer", c.buffer)
							output.SetErrorMsg("중복된 항목: caudal")
						} else {
							slog.Debug("Hotstring Triggered (Leftover 'c' -> caudal)", "trigger", "caudal", "site", caudalInj.GetSite(), "direction", caudalInj.GetDirection())
							addedInjectionKeys[caudalKey] = true
							c.treatments.SetAddInjection(caudalInj)
						}
						processed[i] = true
					}
				} else if c.buffer[i] == 'p' {
					slog.Debug("Hotstring Triggered (Leftover 'p' -> isP=true)")
					leftoverP = true
					processed[i] = true
				} else if c.buffer[i] == 'n' {
					slog.Debug("Hotstring Triggered (Leftover 'n' -> isN=true)")
					leftoverN = true
					processed[i] = true
				} else if c.buffer[i] == '7' {
					slog.Debug("Hotstring Triggered (Leftover '7' -> hasSeven=true)")
					c.treatments.SetHasSeven(true)
					processed[i] = true
				} else if c.buffer[i] == 'e' {
					if i+1 < len(c.buffer) && !processed[i+1] && c.buffer[i+1] == 'f' {
						// ef 잔여 → K_FREE (ef) focus+radial, 코드 .+999_ef
						if lastBlockInjection != nil {
							if lastBlockInjection.GetEswtKey() != "" {
								lbEswt := hotstrings.K_ESWT[lastBlockInjection.GetEswtKey()]
								if lbEswt != nil {
									eswt := models.NewESWT(lastBlockInjection.GetDirection(), lbEswt.GetFocus(), lbEswt.GetRadial(), constants.K_FREE, false, []string{})
									c.treatments.SetESWT(*eswt)
								}
							} else {
								eswtForbiddenWarnings = append(eswtForbiddenWarnings, "★★★ "+lastBlockBaseKey+" eswt는 금지 부위 입니다")
							}
						}
						processed[i] = true
						processed[i+1] = true
						i++
					} else if i+1 < len(c.buffer) && !processed[i+1] && c.buffer[i+1] == 'r' {
						// er 잔여 → K_FREE_RADIAL_ONLY (ef) radial only, 코드 .+999_ef
						if lastBlockInjection != nil {
							if lastBlockInjection.GetEswtKey() != "" {
								lbEswt := hotstrings.K_ESWT[lastBlockInjection.GetEswtKey()]
								if lbEswt != nil {
									eswt := models.NewESWT(lastBlockInjection.GetDirection(), lbEswt.GetFocus(), lbEswt.GetRadial(), constants.K_FREE_RADIAL_ONLY, false, []string{})
									c.treatments.SetESWT(*eswt)
								}
							} else {
								eswtForbiddenWarnings = append(eswtForbiddenWarnings, "★★★ "+lastBlockBaseKey+" eswt는 금지 부위 입니다")
							}
						}
						processed[i] = true
						processed[i+1] = true
						i++
					} else {
						// e 단독 → K_NORMAL (e) focus+radial
						if lastBlockInjection != nil {
							if lastBlockInjection.GetEswtKey() != "" {
								lbEswt := hotstrings.K_ESWT[lastBlockInjection.GetEswtKey()]
								if lbEswt != nil {
									eswt := models.NewESWT(lastBlockInjection.GetDirection(), lbEswt.GetFocus(), lbEswt.GetRadial(), "normal", false, []string{})
									c.treatments.SetESWT(*eswt)
								}
							} else {
								eswtForbiddenWarnings = append(eswtForbiddenWarnings, "★★★ "+lastBlockBaseKey+" eswt는 금지 부위 입니다")
							}
						}
						processed[i] = true
					}
				} else if c.buffer[i] == 's' {
					if lastBlockInjection != nil && lastBlockInjection.GetEswtKey() != "" {
						// interscapular 등 specific sonostim 우선 적용
						useInj := lastBlockInjection
						lbSnt := hotstrings.K_SonoStim[lastBlockInjection.GetSonoStim()]
						// interscapular 등 specific sonostim 우선 적용
						if specificSonoStimInj != nil && lbSnt != nil && (lbSnt.GetSite() == "TPZ" || lbSnt.GetSite() == "lower back") {
							useInj = specificSonoStimInj
							lbSnt = hotstrings.K_SonoStim[useInj.GetSonoStim()]
						}
						sntCode := constants.K_SINGLE_SNT
						if useInj.GetDirection() == constants.K_BOTH && lbSnt != nil && lbSnt.GetSite() != "TPZ" && lbSnt.GetSite() != "lower back" && lbSnt.GetSite() != "interscapular" {
							sntCode = constants.K_BOTH_SNT
						}
						if sntVal, exists := hotstrings.K_SonoStim[lastBlockBaseKey]; exists {
							if specificSonoStimInj != nil && (sntVal.GetSite() == "TPZ" || sntVal.GetSite() == "lower back") {
								specSnt := hotstrings.K_SonoStim[specificSonoStimInj.GetSonoStim()]
								if specSnt != nil {
									newSnt := models.NewSonoStim(specificSonoStimInj.GetDirection(), specSnt.GetSite(), sntCode)
									c.treatments.SetAddExtraTreatments(*newSnt)
								}
							} else {
								c.treatments.SetAddExtraTreatments(*sntVal)
							}
						} else if lastBlockInjection.GetSonoStim() != "" && lbSnt != nil {
							snt := models.NewSonoStim(useInj.GetDirection(), lbSnt.GetSite(), sntCode)
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
	if leftoverN {
		c.treatments.SetAllInjectionsIsN(true)
	}

	// 모드 프리픽스 문자(z/x/s/e)는 별도로 processed에 표시
	if isFirstMeetingMode || isXrayMode || isSonoMode || isEswtMode {
		processed[0] = true
	}

	// xray 모드: 수집된 xray를 출력
	// trailing 's' 개수에 따라:
	//   1개: key+"s" 버전 xray (예: xshrs → shrs)
	//   2개: key+"s" 버전 xray + sono (예: xshrss → shrs + sono shr)
	if isXrayMode {
		// 끝에서부터 미처리 's'가 연속으로 몇 개인지 센다
		trailingSCount := 0
		for i := len(c.buffer) - 1; i >= 1; i-- {
			if c.buffer[i] == 's' && !processed[i] {
				trailingSCount++
			} else {
				break
			}
		}
		hasTrailingS := trailingSCount >= 1
		hasSonoS := trailingSCount >= 2

		// 'ms' 패턴 감지: 버퍼 끝이 미처리 's'이고 그 앞이 미처리 'm'이면
		// 코드에 s를 추가하지 않고 sono만 추가 (예: xshrms → shr xray + shr sono)
		hasMSono := false
		if trailingSCount == 1 {
			sPos := len(c.buffer) - 1
			for i := len(c.buffer) - 1; i >= 1; i-- {
				if c.buffer[i] == 's' && !processed[i] {
					sPos = i
					break
				}
			}
			if sPos >= 2 && c.buffer[sPos-1] == 'm' && !processed[sPos-1] {
				hasMSono = true
				processed[sPos-1] = true // 'm' 소비
				hasTrailingS = false     // 's'를 xray 코드 변환에 쓰지 않음
				hasSonoS = false
			}
		}

		// 미처리 trailing 's' 모두 소비
		consumed := 0
		for i := len(c.buffer) - 1; i >= 1 && consumed < trailingSCount; i-- {
			if c.buffer[i] == 's' && !processed[i] {
				processed[i] = true
				consumed++
			}
		}

		for i, key := range matchedXrayBaseKeys {
			var xrayToAdd *models.Xray
			if hasTrailingS {
				sKey := key + "s"
				if xrayVal, exists := hotstrings.K_Xrays[sKey]; exists {
					slog.Debug("Hotstring Triggered (Xray trailing-s Xray variant)", "key", sKey)
					xrayToAdd = xrayVal
				} else {
					xrayToAdd = matchedXrayValues[i]
				}
			} else {
				xrayToAdd = matchedXrayValues[i]
			}
			if xrayToAdd != nil {
				curSimple := output.GetSimpleText()
				if curSimple != "" {
					output.SetSimpleText(curSimple + "\n" + xrayToAdd.GetText())
				} else {
					output.SetSimpleText(xrayToAdd.GetText())
				}
				output.AddOrderCode(xrayToAdd.GetCode())
			}
		}

		// 's' 2개: sono 추가 (예: xshrss → shrs xray + shr sono)
		if hasSonoS {
			for _, key := range matchedXrayBaseKeys {
				if sonoVal, exists := hotstrings.K_Sonos[key]; exists {
					slog.Debug("Hotstring Triggered (Xray extra-s Sono)", "key", key)
					curSimple := output.GetSimpleText()
					if curSimple != "" {
						output.SetSimpleText(curSimple + "\n" + sonoVal.GetText())
					} else {
						output.SetSimpleText(sonoVal.GetText())
					}
					output.AddOrderCode(sonoVal.GetCode())
				}
			}
		}

		// 'ms' 패턴: 코드에 s 없이 sono만 추가 (예: xshrms → shr xray + shr sono)
		if hasMSono {
			for _, key := range matchedXrayBaseKeys {
				if sonoVal, exists := hotstrings.K_Sonos[key]; exists {
					slog.Debug("Hotstring Triggered (Xray ms-Sono)", "key", key)
					curSimple := output.GetSimpleText()
					if curSimple != "" {
						output.SetSimpleText(curSimple + "\n" + sonoVal.GetText())
					} else {
						output.SetSimpleText(sonoVal.GetText())
					}
					output.AddOrderCode(sonoVal.GetCode())
				}
			}
		}
	}

	// ESWT 모드에서 trailing 's'가 미처리 상태이면 → 매칭된 모든 ESWT 키에 대해 K_SonoStim도 추가
	if isEswtMode && len(c.buffer) > 1 && c.buffer[len(c.buffer)-1] == 's' && !processed[len(c.buffer)-1] {
		lastPos := len(c.buffer) - 1
		processed[lastPos] = true
		for _, key := range matchedEswtBaseKeys {
			if sntVal, exists := hotstrings.K_SonoStim[key]; exists {
				slog.Debug("Hotstring Triggered (ESWT_MODE trailing-s SonoStim)", "key", key)
				c.treatments.SetAddExtraTreatments(*sntVal)
			}
		}
	}

	// FirstMeeting 모드(z prefix)에서 trailing 's'가 미처리 상태이면 → combinedFirstMeeting의 xray 리스트를 's' 버전으로 교체
	// 's'는 버퍼 맨 끝에 오거나 duration(f+숫자) 앞에 올 수 있음 (예: zshrf1s, zshrsf1 모두 허용)
	if isFirstMeetingMode && combinedFirstMeeting != nil {
		trailingSPos := -1
		for i := 1; i < len(c.buffer); i++ {
			if !processed[i] && c.buffer[i] == 's' {
				trailingSPos = i
				break
			}
		}
		if trailingSPos != -1 {
			processed[trailingSPos] = true
			// K_Xrays를 (text,code) → non-s key 역방향 맵으로 구성
			type xrayValKey struct{ text, code string }
			xrayValueToKey := make(map[xrayValKey]string)
			for k, v := range hotstrings.K_Xrays {
				if !strings.HasSuffix(k, "s") {
					xrayValueToKey[xrayValKey{v.GetText(), v.GetCode()}] = k
				}
			}
			xrays := combinedFirstMeeting.GetXray()
			newXrays := make([]models.Xray, 0, len(xrays))
			for _, xr := range xrays {
				xKey := xrayValKey{xr.GetText(), xr.GetCode()}
				if k, ok := xrayValueToKey[xKey]; ok {
					sKey := k + "s"
					if sXray, exists := hotstrings.K_Xrays[sKey]; exists {
						slog.Debug("Hotstring Triggered (FirstMeeting trailing-s Xray replace)", "key", sKey)
						newXrays = append(newXrays, *sXray)
					} else {
						newXrays = append(newXrays, xr)
					}
				} else {
					newXrays = append(newXrays, xr)
				}
			}
			combinedFirstMeeting.SetXray(newXrays)
		}
	}

	// FirstMeeting 모드(z prefix)에서 f+숫자+[dmyw]? 패턴을 duration으로 처리하여 processed에 표시
	if isFirstMeetingMode {
		durationRe := regexp.MustCompile(`f(\d+[dmyw]?|o|d|m|y)`)
		if durLoc := durationRe.FindStringIndex(c.buffer); durLoc != nil {
			for pos := durLoc[0]; pos < durLoc[1]; pos++ {
				processed[pos] = true
			}
		}
		// 날짜 패턴 (예: 5/5, 12/12) 처리
		dateRe := regexp.MustCompile(`\d{1,2}/\d{1,2}`)
		if dateLoc := dateRe.FindStringIndex(c.buffer); dateLoc != nil {
			for pos := dateLoc[0]; pos < dateLoc[1]; pos++ {
				processed[pos] = true
			}
		}
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
		re := regexp.MustCompile(`f(\d+[dmyw]?|o|d|m|y)`)
		durationMatches := re.FindStringSubmatch(c.buffer)
		if len(durationMatches) > 1 {
			combinedFirstMeeting.SetDuration(durationMatches[1])
		}
		dateRe2 := regexp.MustCompile(`(\d{1,2}/\d{1,2})`)
		dateMatches := dateRe2.FindStringSubmatch(c.buffer)
		if len(dateMatches) > 1 {
			combinedFirstMeeting.SetDate(dateMatches[1])
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
		for _, ac := range c.treatments.GetESWTAddCodes() {
			output.AddOrderCode(ac)
		}

		drug := c.treatments.GetDrug()
		if drug != "" {
			output.SetDrug(drug)
		}
	}

	// etc 코드가 .+999_pf_0 이면 "pt) saso\n    magnetic\n" 를 f/u) 앞에 삽입
	if hasCPrefix {
		ct := output.GetChartText()
		magneticArea := ""
		if area, exists := hotstrings.K_Magnetics[lastBlockBaseKey]; exists {
			magneticArea = " " + area
		}
		insert := "pt) saso\n     magnetic" + magneticArea + "\n"
		if strings.Contains(ct, "f/u)") {
			ct = strings.Replace(ct, "f/u)", insert+"f/u)", 1)
		} else if ct != "" {
			ct += insert
		} else {
			ct = insert
		}
		output.SetChartText(ct)
	}

	// etc 코드가 .+999_pm_0 이면 "pt) magnetic\n" 를 f/u) 앞에 삽입
	if hasPPrefix {
		ct := output.GetChartText()
		magneticArea := ""
		if area, exists := hotstrings.K_Magnetics[lastBlockBaseKey]; exists {
			magneticArea = " " + area
		}
		insert := "pt) magnetic" + magneticArea + "\n"
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
				warnings = append(warnings, "★★★ 요추치료 + 상지주사 조합 경고")
				warnings = append(warnings, "★★★ 처방을 다시 확인하세요")
			}

			// Rule 2: 경추 치료 + 하지 주사
			hasCervical := isCervicalSpineInj(first.GetSite()) || isCervicalSpineInj(second.GetSite())
			hasLower := isLowerLimbInj(first.GetSite()) || isLowerLimbInj(second.GetSite())
			if hasCervical && hasLower {
				warnings = append(warnings, "★★★ 경추치료 + 하지주사 조합 경고")
				warnings = append(warnings, "★★★ 처방을 다시 확인하세요")
			}

			// Rule 3: 첫번째와 두번째 주사 방향 불일치
			firstDir := first.GetDirection()
			secondDir := second.GetDirection()
			if firstDir != "" && secondDir != "" &&
				firstDir != "Both" && secondDir != "Both" &&
				firstDir != secondDir {
				warnings = append(warnings, "★★★ 주사 방향 불일치 경고")
				warnings = append(warnings, "★★★ "+firstDir+" vs "+secondDir)
			}

			if len(warnings) > 0 {
				ct := output.GetChartText()
				if ct != "" {
					output.SetChartText(ct + strings.Join(warnings, "\n") + "\n")
				}
			}
		}
	}

	// ESWT 금지 부위 경고를 chartText 맨 마지막에 추가
	if len(eswtForbiddenWarnings) > 0 {
		ct := output.GetChartText()
		warn := strings.Join(eswtForbiddenWarnings, "\n")
		if ct != "" {
			output.SetChartText(ct + "\n" + warn)
		} else {
			output.SetChartText(warn)
		}
	}

	// 클립보드 트리거(Ctrl+*) 플래그 설정
	if isClipboard {
		output.SetIsClipboard(true)
	}

	// 매칭된 글자 수 + 트리거 키 1 = deleteCount
	// 한국어 등 멀티바이트 문자 처리: 바이트 수 대신 룬(문자) 단위로 카운트
	if !isClipboard {
		deleteCount = 0
		{
			byteIdx := 0
			for _, r := range c.buffer {
				if processed[byteIdx] {
					deleteCount++
				}
				byteIdx += utf8.RuneLen(r)
			}
		}
		slog.Info("[DELETE] processed map size", "processedLen", deleteCount, "buffer", c.buffer, "bufferLen", len(c.buffer))
		if deleteCount > 0 {
			deleteCount += 1 // 트리거 키
			// z/x/s/e prefix는 processed[0]=true로 포함되어 있으므로 별도 추가 불필요
			if isFirstMeetingMode {
				deleteCount += 1
			}
		}
		slog.Info("[DELETE] deleteCount before cap", "deleteCount", deleteCount,
			"isFirstMeetingMode", isFirstMeetingMode, "isXrayMode", isXrayMode,
			"isSonoMode", isSonoMode, "isEswtMode", isEswtMode)
		if deleteCount > 20 {
			deleteCount = 20
		}
		output.SetDeleteHostring(deleteCount)
		slog.Info("[DELETE] final deleteCount set", "deleteCount", deleteCount)
	}

	return output
}
