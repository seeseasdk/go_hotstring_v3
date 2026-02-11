package models

import "strconv"

type KeyMouseEvent struct {
	char       rune
	keyCode    uint16
	mouseClick bool
}

func NewKeyObject(char rune) KeyMouseEvent {
	return KeyMouseEvent{
		char:       char,
		keyCode:    0,
		mouseClick: false,
	}
}

func (k *KeyMouseEvent) SetChar(char rune) {
	k.char = char
}

func (k *KeyMouseEvent) SetKeyCode(keyCode uint16) {
	k.keyCode = keyCode
}

func (k *KeyMouseEvent) SetMouseClick(mouseClick bool) {
	k.mouseClick = mouseClick
}

func (k KeyMouseEvent) GetChar() rune {
	return k.char
}

func (k KeyMouseEvent) GetKeyCode() uint16 {
	return k.keyCode
}

func (k KeyMouseEvent) GetMouseClick() bool {
	return k.mouseClick
}

func (k KeyMouseEvent) ToString() string {
	return "KeyMouseEvent: char: " + string(k.char) + " keyCode: " + strconv.Itoa(int(k.keyCode)) + " mouseClick: " + strconv.FormatBool(k.mouseClick)
}
