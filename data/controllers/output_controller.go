package controllers

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/go-vgo/robotgo"
	"github.com/seeseasdk/go_hotstring_v3/data/constants"
	"github.com/seeseasdk/go_hotstring_v3/data/hotstrings"
	"github.com/seeseasdk/go_hotstring_v3/data/models"
)

type OutputController struct {
	cc *ChannelController
}

func NewOutputController(cc *ChannelController) *OutputController {
	return &OutputController{
		cc: cc,
	}
}

func (c *OutputController) Start() {
	slog.Info("OutputController started")

	go func() {
		outputCount := 0
		for stuff := range c.cc.OutputChan {
			// Check if we received an OutputStuff update
			if output, ok := stuff.Object.(*models.OutputStuff); ok {
				outputCount++
				slog.Warn("OutputController: Output Received", "outputCount", outputCount)

				// 중복 등 에러가 있으면 키보드 입력 없이 에러 로그만 남기고 스킵
				if errMsg := output.GetErrorMsg(); errMsg != "" {
					slog.Error("OutputController: 중복 감지 - 키보드 입력 건너뜀", "error", errMsg)
					c.cc.ResetChan <- true
					continue
				}

				// Show final output result before reset
				result := output.ToString()
				slog.Info("Final Output", "result", result)

				// 방금 입력한 트리거 문자열 길이만큼 백스페이스 눌러서 지우기
				if deleteSpace := output.GetDeleteHostring(); deleteSpace > 0 {
					slog.Debug("[BACKSPACE] Will tap backspace to delete hostring", "count", deleteSpace)

					// Ctrl 키가 물리적으로 눌려있어 Ctrl+Backspace(단어 단위 지우기)가 발생하는 것을 막기 위해 강제 해제
					robotgo.KeyToggle("ctrl", "up")
					time.Sleep(10 * time.Millisecond)

					for i := 0; i < deleteSpace; i++ {
						robotgo.KeyTap("backspace")
						time.Sleep(10 * time.Millisecond) // 백스페이스 연속 입력간 아주 짧은 딜레이 (원상복구)
					}
					slog.Debug("[BACKSPACE] Finished", "count", deleteSpace)
					time.Sleep(50 * time.Millisecond) // 다 지우고 약간 대기
				}

				// TypeStr 출력 시작 전 InputChan 경유 뮤팅 (채널 순서 보장으로 타이밍 버그 방지)
				c.cc.IsMuting.Store(true)
				c.cc.InputChan <- models.NewChannelStuff("OutputController", "HotstringController", "MuteStart", false, nil)

				if chartText := output.GetChartText(); chartText != "" {
					if coord, exists := hotstrings.K_Coordinates["chartWindow"]; exists {
						time.Sleep(50 * time.Millisecond) // 창 이동 전 딜레이 추가
						robotgo.Move(coord.X, coord.Y)
						robotgo.Click("left")
						time.Sleep(50 * time.Millisecond) // 클릭 후 포커스 딜레이

						// Ctrl + End 순차 입력 (동시 입력 방지)
						robotgo.KeyToggle("ctrl", "down")
						time.Sleep(10 * time.Millisecond)
						robotgo.KeyTap("end")
						time.Sleep(10 * time.Millisecond)
						robotgo.KeyToggle("ctrl", "up")
						time.Sleep(50 * time.Millisecond)

						if !output.GetSkipChartEnter() {
							robotgo.KeyTap("enter")
							time.Sleep(50 * time.Millisecond)
						}

						// 키보드 입력을 통해 글자 입력 (로봇고 사용)
						robotgo.TypeStr(strings.TrimRight(chartText, " \t\r\n") + "\n")
						time.Sleep(50 * time.Millisecond)
					}
				}

				if specificText := output.GetSpecificText(); specificText != "" {
					if coord, exists := hotstrings.K_Coordinates["specificWindow"]; exists {
						time.Sleep(50 * time.Millisecond) // 창 이동 전 딜레이 추가
						robotgo.Move(coord.X, coord.Y)
						robotgo.Click("left")
						time.Sleep(50 * time.Millisecond) // 클릭 후 포커스 딜레이

						// Ctrl + End 순차 입력 (동시 입력 방지)
						robotgo.KeyToggle("ctrl", "down")
						time.Sleep(10 * time.Millisecond)
						robotgo.KeyTap("end")
						time.Sleep(10 * time.Millisecond)
						robotgo.KeyToggle("ctrl", "up")
						time.Sleep(50 * time.Millisecond)

						robotgo.KeyTap("enter")
						time.Sleep(50 * time.Millisecond)

						robotgo.TypeStr(specificText)
						time.Sleep(50 * time.Millisecond)
					}
				}

				if mx999Text := output.GetMx999Text(); mx999Text != "" {
					if coord, exists := hotstrings.K_Coordinates["mx999Window"]; exists {
						time.Sleep(50 * time.Millisecond) // 창 이동 전 딜레이 추가
						robotgo.Move(coord.X, coord.Y)
						robotgo.Click("left")
						time.Sleep(50 * time.Millisecond) // 클릭 후 포커스 딜레이

						robotgo.KeyToggle("ctrl", "down")
						time.Sleep(10 * time.Millisecond)
						robotgo.KeyTap("end")
						time.Sleep(10 * time.Millisecond)
						robotgo.KeyToggle("ctrl", "up")
						time.Sleep(50 * time.Millisecond)

						robotgo.TypeStr(mx999Text)
						time.Sleep(50 * time.Millisecond)
					}
				}

				if simpleText := output.GetSimpleText(); simpleText != "" {
					// 윈도우 클릭 없이 바로 입력
					robotgo.TypeStr(simpleText)
					time.Sleep(50 * time.Millisecond)

					// ExtraDo가 same_input_memo_window이면 memoWindow에도 동일 입력 후 chartWindow로 복귀
					if output.GetExtraDo() == constants.K_SAME_INPUT_MEMO_WINDOW {
						if coord, exists := hotstrings.K_Coordinates["memoWindow"]; exists {
							time.Sleep(50 * time.Millisecond)
							robotgo.Move(coord.X, coord.Y)
							robotgo.Click("left")
							time.Sleep(50 * time.Millisecond)
							robotgo.TypeStr(simpleText)
							time.Sleep(50 * time.Millisecond)
						}
						if coord, exists := hotstrings.K_Coordinates["chartWindow"]; exists {
							time.Sleep(50 * time.Millisecond)
							robotgo.Move(coord.X, coord.Y)
							robotgo.Click("left")
							time.Sleep(50 * time.Millisecond)
						}
					}
				}

				drugCode := output.GetDrugCode()
				drugDays := output.GetDrug()
				orderCodes := output.GetOrderCode()
				if drugCode != "" || len(orderCodes) > 0 {
					if coord, exists := hotstrings.K_Coordinates["orderWindow"]; exists {
						time.Sleep(50 * time.Millisecond)
						robotgo.Move(coord.X, coord.Y)
						robotgo.Click("left")
						time.Sleep(50 * time.Millisecond)

						// 먼저 Down 키 20번 입력
						for i := 0; i < 20; i++ {
							robotgo.KeyTap("down")
							time.Sleep(10 * time.Millisecond)
						}
						time.Sleep(50 * time.Millisecond)

						// 약 코드: 코드 입력 → Enter → 딜레이 → 일수 입력 → Enter
						if drugCode != "" {
							robotgo.TypeStr(drugCode)
							time.Sleep(500 * time.Millisecond)
							robotgo.KeyTap("enter")
							time.Sleep(50 * time.Millisecond)
							if drugDays != "" {
								robotgo.TypeStr(drugDays)
								time.Sleep(500 * time.Millisecond)
								robotgo.KeyTap("enter")
								time.Sleep(500 * time.Millisecond)
							}
						}

						// 나머지 일반 주문 코드
						for _, code := range orderCodes {
							if code == "" {
								continue // 빈 코드 스킵
							}
							if strings.Contains(code, "#") {
								parts := strings.Split(code, "#")
								robotgo.TypeStr(parts[0])
								time.Sleep(500 * time.Millisecond)
								robotgo.KeyTap("enter")
								time.Sleep(500 * time.Millisecond)
								robotgo.TypeStr(parts[1])
								time.Sleep(500 * time.Millisecond)
								robotgo.KeyTap("enter")
								time.Sleep(500 * time.Millisecond)
							} else {
								robotgo.TypeStr(code)
								time.Sleep(500 * time.Millisecond)
								robotgo.KeyTap("enter")
								time.Sleep(500 * time.Millisecond)
							}
						}
						time.Sleep(50 * time.Millisecond)
					}
				}

				if memoText := output.GetMemoText(); memoText != "" {
					if coord, exists := hotstrings.K_Coordinates["memoWindow"]; exists {
						time.Sleep(50 * time.Millisecond) // 창 이동 전 딜레이 추가
						robotgo.Move(coord.X, coord.Y)
						robotgo.Click("left")
						time.Sleep(50 * time.Millisecond) // 클릭 후 포커스 딜레이

						robotgo.KeyToggle("ctrl", "down")
						time.Sleep(10 * time.Millisecond)
						robotgo.KeyTap("end")
						time.Sleep(10 * time.Millisecond)
						robotgo.KeyToggle("ctrl", "up")
						time.Sleep(50 * time.Millisecond)

						robotgo.TypeStr(memoText)
						time.Sleep(50 * time.Millisecond)
					}
				}

				// 출력 완료 후 뮤팅 해제 및 hotstring 버퍼 초기화 (ClearBuffer가 isMuting=false도 처리)
				c.cc.IsMuting.Store(false)
				c.cc.InputChan <- models.NewChannelStuff("OutputController", "HotstringController", "ClearBuffer", false, nil)

				// Ctrl+Enter 트리거이면 항상 chartWindow로 커서 복귀
				if !output.GetIsClipboard() && output.GetSimpleText() == "" {
					if coord, exists := hotstrings.K_Coordinates["chartWindow"]; exists {
						time.Sleep(50 * time.Millisecond)
						robotgo.Move(coord.X, coord.Y)
						robotgo.Click("left")
						time.Sleep(50 * time.Millisecond)
						robotgo.KeyToggle("ctrl", "down")
						time.Sleep(10 * time.Millisecond)
						robotgo.KeyTap("end")
						time.Sleep(10 * time.Millisecond)
						robotgo.KeyToggle("ctrl", "up")
					}
				}

				// Ctrl+* 트리거이면 마지막에 Ctrl + - 버튼 클릭
				if output.GetIsClipboard() {
					time.Sleep(50 * time.Millisecond)
					robotgo.KeyToggle("ctrl", "down")
					time.Sleep(10 * time.Millisecond)
					robotgo.KeyTap("-")
					time.Sleep(10 * time.Millisecond)
					robotgo.KeyToggle("ctrl", "up")
				}

				// Send reset signal to clear
				c.cc.ResetChan <- true
				slog.Info("OutputController: Reset signal sent")

				slog.Debug("--------------------------------------------------")
				continue
			}

			// Fallback for direct object passing (if ever used)
			slog.Debug("OutputController received raw object", "object_type", fmt.Sprintf("%T", stuff.Object))
		}
	}()
}
