package controllers

import (
	"log/slog"
	"regexp"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/go-vgo/robotgo"
	hook "github.com/robotn/gohook"

	"github.com/seeseasdk/go_hotstring_v3/data/hotstrings"
	"github.com/seeseasdk/go_hotstring_v3/data/models"
)

var (
	user32                        = syscall.NewLazyDLL("user32.dll")
	kernel32                      = syscall.NewLazyDLL("kernel32.dll")
	procGetForegroundWindow       = user32.NewProc("GetForegroundWindow")
	procGetWindowThreadProcessId  = user32.NewProc("GetWindowThreadProcessId")
	procOpenProcess               = kernel32.NewProc("OpenProcess")
	procQueryFullProcessImageName = kernel32.NewProc("QueryFullProcessImageNameW")
	procCloseHandle               = kernel32.NewProc("CloseHandle")
)

const PROCESS_QUERY_LIMITED_INFORMATION = 0x1000

func getForegroundExeName() string {
	hwnd, _, _ := procGetForegroundWindow.Call()
	if hwnd == 0 {
		return ""
	}
	var pid uint32
	procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if pid == 0 {
		return ""
	}
	handle, _, _ := procOpenProcess.Call(PROCESS_QUERY_LIMITED_INFORMATION, 0, uintptr(pid))
	if handle == 0 {
		return ""
	}
	defer procCloseHandle.Call(handle)

	buf := make([]uint16, 260)
	size := uint32(len(buf))
	procQueryFullProcessImageName.Call(handle, 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)))
	fullPath := syscall.UTF16ToString(buf[:size])

	// 경로에서 파일명만 추출
	idx := strings.LastIndexAny(fullPath, `\/`)
	if idx >= 0 {
		return fullPath[idx+1:]
	}
	return fullPath
}

var activatedWindows = []string{"FForm.exe", "FwChart.exe"}

func isActivatedWindow() bool {
	exeName := getForegroundExeName()
	for _, w := range activatedWindows {
		if strings.EqualFold(exeName, w) {
			return true
		}
	}
	return false
}

type InputController struct {
	cc *ChannelController
}

func NewInputController(cc *ChannelController) *InputController {
	return &InputController{
		cc: cc,
	}
}

func (c *InputController) Start() {
	slog.Info("InputController started. Listening for global input...")
	slog.Info("[TIP] Ctrl+Enter to flush, Ctrl+C or ESC to exit")

	// 글로벌 키보드 훅 시작
	EvChan := hook.Start()
	defer hook.End()

	ctrlPressed := false
	shiftPressed := false

	// 이벤트 대기 루프
	for ev := range EvChan {
		if ev.Kind == hook.KeyDown {
			// Ctrl 키 상태 확인 (VK_CONTROL=17, VK_LCONTROL=162, VK_RCONTROL=163)
			if ev.Rawcode == 17 || ev.Rawcode == 162 || ev.Rawcode == 163 {
				ctrlPressed = true
				continue
			}

			// Shift 키 상태 확인 (VK_SHIFT=16, VK_LSHIFT=160, VK_RSHIFT=161)
			if ev.Rawcode == 16 || ev.Rawcode == 160 || ev.Rawcode == 161 {
				shiftPressed = true
				continue
			}

			// 활성화된 창에서만 동작하도록 체크
			if !isActivatedWindow() {
				continue
			}

			// Ctrl + - 액션 (pacsButton 클릭)
			// Windows에서 대시키(VK_OEM_MINUS)는 189, 텐키패드 마이너스(VK_SUBTRACT)는 109
			if ctrlPressed && (ev.Rawcode == 189 || ev.Rawcode == 109) {
				slog.Info("[CTRL+-] PACS Button Clicked!")
				if coord, exists := hotstrings.K_Coordinates["pacsButton"]; exists {
					robotgo.Move(coord.X, coord.Y)
					robotgo.Click("left")
				}
				continue
			}

			// Ctrl + + 액션 (completeButton 클릭)
			// Windows에서 플러스키(VK_OEM_PLUS)는 187, 텐키패드 플러스(VK_ADD)는 107
			if ctrlPressed && (ev.Rawcode == 187 || ev.Rawcode == 107) {
				slog.Info("[CTRL++] Complete Button Clicked!")
				if coord, exists := hotstrings.K_Coordinates["completeButton"]; exists {
					robotgo.Move(coord.X, coord.Y)
					robotgo.Click("left")
				}
				continue
			}

			// Ctrl + / 액션 (클립보드 복사 후 날짜 없는 줄에 공백 9칸 추가 → specificWindow 입력)
			// 메인키보드 /(VK_OEM_2)=191, 넘패드 /(VK_DIVIDE)=111
			if ctrlPressed && (ev.Rawcode == 191 || ev.Rawcode == 111) {
				slog.Info("[CTRL+/] Copying clipboard and formatting for specificWindow!")

				go func() {
					// 1. Ctrl + X 로 클립보드 복사
					robotgo.KeyTap("x", "ctrl")
					time.Sleep(100 * time.Millisecond)

					// 2. 클립보드 읽기
					text, err := robotgo.ReadAll()
					if err != nil || text == "" {
						return
					}

					// 3. 줄 단위로 처리: 빈 줄 제거, 날짜(YYYY-MM-DD)로 시작하면 그대로, 아니면 공백 추가
					datePrefix := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}`)
					knownPrefixes := []string{"c) ", "s) ", "p) ", "snt) ", "ef) ", "e) ", "pt) ", "pe) ", "f/u) "}
					lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
					var processed []string
					for _, line := range lines {
						if strings.TrimSpace(line) == "" {
							continue // 빈 줄 제거
						}
						if datePrefix.MatchString(line) {
							processed = append(processed, line)
						} else {
							trimmed := strings.TrimLeft(line, " \t")
							hasKnownPrefix := false
							for _, pfx := range knownPrefixes {
								if strings.HasPrefix(trimmed, pfx) {
									hasKnownPrefix = true
									break
								}
							}
							if hasKnownPrefix {
								processed = append(processed, "                "+trimmed) // 16칸
							} else {
								processed = append(processed, "                    "+trimmed) // 20칸
							}
						}
					}
					formatted := strings.Join(processed, "\n")

					// 4. OutputStuff 생성 후 specificText에 담아 OutputChan으로 직접 전송
					output := models.NewOutputStuff(0, "", formatted, "", nil, "", "", "")
					c.cc.OutputChan <- models.NewChannelStuff(
						"InputController",
						"OutputController",
						"UpdateOutput",
						true,
						output,
					)
				}()
				continue
			}

			// Ctrl + * 액션 (선택영역 복사 후 specificText로 입력)
			// 텐키패드 별표(VK_MULTIPLY)=106, 메인키보드 별표(Shift+8)=56
			if ctrlPressed && (ev.Rawcode == 106 || (shiftPressed && ev.Rawcode == 56)) {
				slog.Info("[CTRL+*] Copying and pasting as treatment!")

				go func() {
					// 1. Ctrl + C 입력해서 클립보드로 복사
					robotgo.KeyTap("c", "ctrl")
					time.Sleep(100 * time.Millisecond) // 클립보드에 담길 시간 대기

					// 2. 클립보드 텍스트 읽기
					text, err := robotgo.ReadAll()
					if err == nil && text != "" {
						// 3. 바로 전송하지 않고 HotstringController의 트리트먼트에 복사 내용 추가 명령
						c.cc.InputChan <- models.NewChannelStuff(
							"InputController",
							"HotstringController",
							"AddClipboardMemo",
							true,
							text,
						)

						// 4. 추가된 후 곧바로 출력을 원하므로 Flush명령도 전송
						time.Sleep(50 * time.Millisecond)
						c.cc.InputChan <- models.NewChannelStuff(
							"InputController",
							"HotstringController",
							"FlushTreatments",
							true,
							nil,
						)
					}
				}()
				continue
			}

			// Ctrl + Enter (트리거) (VK_RETURN=13)
			if ctrlPressed && ev.Rawcode == 13 {
				slog.Info("[CTRL+ENTER] Triggered!")
				c.cc.InputChan <- models.NewChannelStuff(
					"InputController",
					"HotstringController",
					"FlushTreatments",
					true,
					"CtrlEnter",
				)
				continue
			}
			// 일반 Enter (VK_RETURN=13) - buffer 초기화
			if !ctrlPressed && ev.Rawcode == 13 {
				c.cc.InputChan <- models.NewChannelStuff(
					"InputController",
					"HotstringController",
					"ClearBuffer",
					false,
					nil,
				)
				continue
			}

			// Home (VK_HOME=36), End (VK_END=35), Shift+Home, Shift+End - buffer 초기화
			if ev.Rawcode == 35 || ev.Rawcode == 36 {
				c.cc.InputChan <- models.NewChannelStuff(
					"InputController",
					"HotstringController",
					"ClearBuffer",
					false,
					nil,
				)
				continue
			}

			// Space (VK_SPACE=32), 방향키/PgUp/PgDn (33~34, 37~40), Delete(46) - buffer 초기화
			if ev.Rawcode == 32 || (ev.Rawcode >= 33 && ev.Rawcode <= 40) || ev.Rawcode == 46 {
				c.cc.InputChan <- models.NewChannelStuff(
					"InputController",
					"HotstringController",
					"ClearBuffer",
					false,
					nil,
				)
				continue
			}
			// 백스페이스 (VK_BACK=8, 삭제키 VK_DELETE=46)
			if ev.Rawcode == 8 {
				c.cc.InputChan <- models.NewChannelStuff(
					"InputController",
					"HotstringController",
					"Backspace",
					false,
					nil,
				)
				continue
			}

			// 일반 문자 입력 처리 (핫스트링 패턴 매칭용)
			if !ctrlPressed {
				// TypeStr 출력 중에는 hotstring 버퍼에 추가하지 않음
				if c.cc.IsMuting.Load() {
					continue
				}

				var charRune rune = 0

				// A-Z (Rawcode 65-90) -> 소문자로 변환
				if ev.Rawcode >= 65 && ev.Rawcode <= 90 {
					charRune = rune(ev.Rawcode + 32)
				} else if ev.Rawcode >= 48 && ev.Rawcode <= 57 {
					// 숫자 0-9 (Rawcode 48-57)
					charRune = rune(ev.Rawcode)
				} else if ev.Rawcode >= 96 && ev.Rawcode <= 105 {
					// 넘패드 숫자 0-9 (Rawcode 96-105)
					charRune = rune(ev.Rawcode - 96 + 48)
				}

				if charRune != 0 {
					c.cc.InputChan <- models.NewChannelStuff(
						"InputController",
						"HotstringController",
						"AddChar",
						false,
						map[string]interface{}{
							"char": charRune,
						},
					)
				}
			}

		} else if ev.Kind == hook.KeyUp {
			// Ctrl 키 떼면 상태 원복
			if ev.Rawcode == 17 || ev.Rawcode == 162 || ev.Rawcode == 163 {
				ctrlPressed = false
			}
			// Shift 키 떼면 상태 원복
			if ev.Rawcode == 16 || ev.Rawcode == 160 || ev.Rawcode == 161 {
				shiftPressed = false
			}
		}
	}
}
