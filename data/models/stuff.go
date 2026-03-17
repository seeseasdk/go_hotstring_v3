package models

import "fmt"

// urgent := true이면 바로 보내고 (취소 주문)
// urgent := false이면 500ms마다 보내고

type ChannelStuff struct {
	From   string
	To     string
	Do     string
	Urgent bool
	Object any
}

func NewChannelStuff(from, to, do string, urgent bool, object any) *ChannelStuff {
	return &ChannelStuff{
		From:   from,
		To:     to,
		Do:     do,
		Urgent: urgent,
		Object: object,
	}
}

func (c ChannelStuff) ToString() string {
	return fmt.Sprintf("From: %s\nTo: %s\nDo: %s\nUrgent: %v\nObject: %v\n",
		c.From, c.To, c.Do, c.Urgent, c.Object)
}

func (c ChannelStuff) IsEmpty() bool {
	return c.From == "" && c.To == "" && c.Do == "" && !c.Urgent && c.Object == nil
}

type ChannelStuffInterface interface {
	GetToChannelController() chan ChannelStuff
	GetToHotstringController() chan ChannelStuff
	GetToOutputController() chan ChannelStuff
	GetToInputController() chan ChannelStuff
}

type OutputStuff struct {
	deleteHostring int
	chartText      string
	specificText   string
	mx999Text      string
	orderCode      []string
	memoText       string
	simpleText     string // 윈도우 상관없이 커서에 단순한 텍스트를 입력할 때 사용
	drug           string // 약 날짜 (일수)
	drugCode       string // 약 주문 코드 (.+51 등)
	// followUp       string // 추적관찰 날짜
	extraDo string // 추가적인 동작을 수행할 때 사용
}

func NewOutputStuff(deleteHostring int, chartText, specificText, mx999Text string, orderCode []string, memoText, simpleText, drug string) *OutputStuff {
	return &OutputStuff{
		deleteHostring: deleteHostring,
		chartText:      chartText,
		specificText:   specificText,
		mx999Text:      mx999Text,
		orderCode:      orderCode,
		memoText:       memoText,
		simpleText:     simpleText,
		drug:           drug,
		// followUp:       followUp,
		extraDo: "",
	}
}
func (os *OutputStuff) SetDeleteHostring(deleteHostring int) {
	os.deleteHostring = deleteHostring
}
func (os *OutputStuff) SetChartText(chartText string) {
	os.chartText = chartText
}
func (os *OutputStuff) SetSpecificText(specificText string) {
	os.specificText = specificText
}
func (os *OutputStuff) SetMx999Text(mx999Text string) {
	os.mx999Text = mx999Text
}
func (os *OutputStuff) SetOrderCode(orderCode []string) {
	os.orderCode = orderCode
}
func (os *OutputStuff) AddOrderCode(orderCode string) {
	if os.orderCode == nil {
		os.orderCode = []string{}
	}
	os.orderCode = append(os.orderCode, orderCode)
}
func (os *OutputStuff) AddOrderCodeList(orderCode []string) {
	if os.orderCode == nil {
		os.orderCode = []string{}
	}
	os.orderCode = append(os.orderCode, orderCode...)
}
func (os *OutputStuff) SetMemoText(memoText string) {
	os.memoText = memoText
}
func (os *OutputStuff) SetSimpleText(simpleText string) {
	os.simpleText = simpleText
}
func (os *OutputStuff) SetDrug(drug string) {
	os.drug = drug
}
func (os *OutputStuff) SetDrugCode(drugCode string) {
	os.drugCode = drugCode
}
func (os *OutputStuff) GetDrugCode() string {
	return os.drugCode
}

//	func (os *OutputStuff) SetFollowUp(followUp string) {
//		os.followUp = followUp
//	}
func (os *OutputStuff) SetExtraDo(extraDo string) {
	os.extraDo = extraDo
}
func (os *OutputStuff) SetReset() {
	os.deleteHostring = 0
	os.chartText = ""
	os.specificText = ""
	os.mx999Text = ""
	os.orderCode = nil
	os.memoText = ""
	os.simpleText = ""
	os.drug = ""
	os.drugCode = ""
	// os.followUp = ""
}
func (os *OutputStuff) GetDeleteHostring() int {
	return os.deleteHostring
}
func (os *OutputStuff) GetChartText() string {
	return os.chartText
}
func (os *OutputStuff) GetSpecificText() string {
	return os.specificText
}
func (os *OutputStuff) GetMx999Text() string {
	return os.mx999Text
}
func (os *OutputStuff) GetOrderCode() []string {
	return os.orderCode
}
func (os *OutputStuff) GetMemoText() string {
	return os.memoText
}
func (os *OutputStuff) GetSimpleText() string {
	return os.simpleText
}
func (os *OutputStuff) GetDrug() string {
	return os.drug
}

//	func (os *OutputStuff) GetFollowUp() string {
//		return os.followUp
//	}
func (os *OutputStuff) GetExtraDo() string {
	return os.extraDo
}
func (os *OutputStuff) IsEmpty() bool {
	if os.deleteHostring == 0 &&
		os.chartText == "" &&
		os.specificText == "" &&
		os.mx999Text == "" &&
		len(os.orderCode) == 0 &&
		os.memoText == "" &&
		os.simpleText == "" &&
		os.drug == "" &&
		os.drugCode == "" &&
		// os.followUp == "" &&
		os.extraDo == "" {
		return true
	}
	return false
}
func (os *OutputStuff) IsEmptyWithoutDeleteHotstring() bool {
	if os.chartText == "" &&
		os.specificText == "" &&
		os.mx999Text == "" &&
		len(os.orderCode) == 0 &&
		os.memoText == "" &&
		os.simpleText == "" &&
		os.drug == "" &&
		os.drugCode == "" &&
		// os.followUp == "" &&
		os.extraDo == "" {
		return true
	}
	return false
}
func (os *OutputStuff) ToString() string {
	orderCode := ""
	for _, oc := range os.orderCode {
		orderCode += oc + ", "
	}
	return "DeleteHostring: " + fmt.Sprint(os.deleteHostring) + "\n" +
		"ChartText: " + os.chartText + "\n" +
		"SpecificText: " + os.specificText + "\n" +
		"MX999Text: " + os.mx999Text + "\n" +
		"OrderCode: " + orderCode + "\n" +
		"MemoText: " + os.memoText + "\n" +
		"SimpleText: " + os.simpleText + "\n" +
		"Drug: " + os.drug + " (code: " + os.drugCode + ")" + "\n" +
		// "FollowUp: " + os.followUp + "\n" +
		"ExtraDo: " + os.extraDo + "\n"
}
