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
		for stuff := range c.cc.OutputChan {
			// Check if we received an OutputStuff update
			if output, ok := stuff.Object.(*models.OutputStuff); ok {
				slog.Info("OutputController: Output Updated")

				// Show final output result before reset
				result := output.ToString()
				slog.Info("Final Output:", "result", result)
				fmt.Printf("✅ [COMPLETED] %s\n", result)

				// 방금 입력한 트리거 문자열 길이만큼 백스페이스 눌러서 지우기
				if deleteSpace := output.GetDeleteHostring(); deleteSpace > 0 {
					fmt.Printf("🛠  [DEBUG-BACKSPACE] Will tap backspace %d times to delete hostring.\n", deleteSpace)

					// Ctrl 키가 물리적으로 눌려있어 Ctrl+Backspace(단어 단위 지우기)가 발생하는 것을 막기 위해 강제 해제
					robotgo.KeyToggle("ctrl", "up")
					time.Sleep(10 * time.Millisecond)

					for i := 0; i < deleteSpace; i++ {
						fmt.Printf("   -> Tapping backspace (%d/%d)\n", i+1, deleteSpace)
						robotgo.KeyTap("backspace")
						time.Sleep(10 * time.Millisecond) // 백스페이스 연속 입력간 아주 짧은 딜레이 (원상복구)
					}
					fmt.Printf("🛠  [DEBUG-BACKSPACE] Finished tapping backspace %d times.\n", deleteSpace)
					time.Sleep(50 * time.Millisecond) // 다 지우고 약간 대기
				}

				if chartText := output.GetChartText(); chartText != "" {
					if coord, exists := hotstrings.K_Coordinates["chartWindow"]; exists {
						time.Sleep(50 * time.Millisecond) // 창 이동 전 딜레이 추가
						robotgo.Move(coord.X, coord.Y)
						robotgo.Click("left")
						time.Sleep(50 * time.Millisecond) // 클릭 후 포커스 딜레이

						robotgo.KeyTap("end", "ctrl") // Ctrl + End 입력
						time.Sleep(50 * time.Millisecond)

						// 키보드 입력을 통해 글자 입력 (로봇고 사용)
						robotgo.TypeStr(chartText)
						time.Sleep(50 * time.Millisecond)
					}
				}

				if specificText := output.GetSpecificText(); specificText != "" {
					if coord, exists := hotstrings.K_Coordinates["specificWindow"]; exists {
						time.Sleep(50 * time.Millisecond) // 창 이동 전 딜레이 추가
						robotgo.Move(coord.X, coord.Y)
						robotgo.Click("left")
						time.Sleep(50 * time.Millisecond) // 클릭 후 포커스 딜레이

						robotgo.KeyTap("end", "ctrl") // Ctrl + End 입력
						time.Sleep(50 * time.Millisecond)

						robotgo.TypeStr(specificText)
						time.Sleep(50 * time.Millisecond)
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

						// 약 코드: 코드 입력 → 딜레이 → 일수 입력 → Enter
						if drugCode != "" {
							robotgo.TypeStr(drugCode)
							time.Sleep(500 * time.Millisecond)
							if drugDays != "" {
								robotgo.TypeStr(drugDays)
								time.Sleep(500 * time.Millisecond)
							}
							robotgo.KeyTap("enter")
							time.Sleep(500 * time.Millisecond)
						}

						// 나머지 일반 주문 코드
						for _, code := range orderCodes {
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

				if mx999Text := output.GetMx999Text(); mx999Text != "" {
					if coord, exists := hotstrings.K_Coordinates["mx999Window"]; exists {
						time.Sleep(50 * time.Millisecond) // 창 이동 전 딜레이 추가
						robotgo.Move(coord.X, coord.Y)
						robotgo.Click("left")
						time.Sleep(50 * time.Millisecond) // 클릭 후 포커스 딜레이
						robotgo.TypeStr(mx999Text)
						time.Sleep(50 * time.Millisecond)
					}
				}

				if memoText := output.GetMemoText(); memoText != "" {
					if coord, exists := hotstrings.K_Coordinates["memoWindow"]; exists {
						time.Sleep(50 * time.Millisecond) // 창 이동 전 딜레이 추가
						robotgo.Move(coord.X, coord.Y)
						robotgo.Click("left")
						time.Sleep(50 * time.Millisecond) // 클릭 후 포커스 딜레이
						robotgo.TypeStr(memoText)
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
