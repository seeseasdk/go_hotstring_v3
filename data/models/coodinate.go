package models

import "fmt"

type WindowCoordinates struct {
	Chart    Coordinate
	Specific Coordinate
	Order    Coordinate
	Mx999    Coordinate
	Memo     Coordinate
	Drug     Coordinate
	Complete Coordinate
	Pacs     Coordinate
}

func NewWindowCoordinates(chart, specific, order, mx999, memo, drug, complete, pacs Coordinate) *WindowCoordinates {
	return &WindowCoordinates{
		Chart:    chart,
		Specific: specific,
		Order:    order,
		Mx999:    mx999,
		Memo:     memo,
		Drug:     drug,
		Complete: complete,
		Pacs:     pacs,
	}
}

func (wc *WindowCoordinates) ToString() string {
	return "Chart: " + wc.Chart.ToString() +
		", Specific: " + wc.Specific.ToString() +
		", Order: " + wc.Order.ToString() +
		", Mx999: " + wc.Mx999.ToString() +
		", Memo: " + wc.Memo.ToString() +
		", Drug: " + wc.Drug.ToString() +
		", Complete: " + wc.Complete.ToString() +
		", Pacs: " + wc.Pacs.ToString()
}

type Coordinate struct {
	X int
	Y int
}

func (c *Coordinate) ToString() string {
	return "X: " + fmt.Sprint(c.X) + ", Y: " + fmt.Sprint(c.Y)
}
