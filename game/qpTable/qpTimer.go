package qpTable

import (
	"sort"
	"time"
)

type TimerFunc func()

type QPTimerItem struct {
	Timer   *time.Timer
	TimerID int32
	//TimeoutS int64
	TimeoutS time.Time
	SeatNum  int32
	DoFunc   TimerFunc
}

type QPTimerArr []*QPTimerItem
type QPTimer struct {
	TimerArr QPTimerArr
}

func (s QPTimerArr) Len() int           { return len(s) }
func (s QPTimerArr) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s QPTimerArr) Less(i, j int) bool { return s[i].TimeoutS.Before(s[j].TimeoutS) }

func (this *QPTimer) InitTimer() {
	this.TimerArr = make([]*QPTimerItem, 0)
}

func (this *QPTimer) PutSeatTimer(seatNum, timerID, millSecond int32, work TimerFunc) {

	timerItem := &QPTimerItem{
		Timer:    time.NewTimer(time.Duration(millSecond) * time.Millisecond),
		TimerID:  timerID,
		TimeoutS: time.Now().Add(time.Duration(millSecond) * time.Millisecond),
		SeatNum:  seatNum,
		DoFunc:   work,
	}
	this.TimerArr = append(this.TimerArr, timerItem)

	sort.Sort(this.TimerArr)

	//fmt.Println("timer length:=", len(this.TimerArr))
}

func (this *QPTimer) PutTableTimer(timerID, millSecond int32, work TimerFunc) {

	timerItem := &QPTimerItem{
		Timer:    time.NewTimer(time.Duration(millSecond) * time.Millisecond),
		TimerID:  timerID,
		TimeoutS: time.Now().Add(time.Duration(millSecond) * time.Millisecond),
		SeatNum:  -1,
		DoFunc:   work,
	}
	this.TimerArr = append(this.TimerArr, timerItem)

	sort.Sort(this.TimerArr)

	//fmt.Println("timer length:=", len(this.TimerArr))
}

func (this *QPTimer) GetMinTimer() *QPTimerItem {
	if len(this.TimerArr) > 0 {
		return this.TimerArr[0]
	}
	return nil
}

func (this *QPTimer) RemoveTimer(time *QPTimerItem) {

	changed := false
	for i, v := range this.TimerArr {
		if v != time {
			continue
		}

		changed = true
		v.Timer.Stop()
		this.TimerArr = append(this.TimerArr[:i], this.TimerArr[i+1:]...)
		break
	}

	if changed == true {
		sort.Sort(this.TimerArr)
	}

	//fmt.Println("timer length:=", len(this.TimerArr))
}

func (this *QPTimer) RemoveByTimeID(id int32) []*QPTimerItem {

	timerInfo := make([]*QPTimerItem, 0)

	changed := false
	for isOver := false; isOver == false; {
		isOver = true
		for i, v := range this.TimerArr {
			if v.TimerID != id {
				continue
			}
			timerInfo = append(timerInfo, v)
			changed = true
			v.Timer.Stop()
			isOver = false
			this.TimerArr = append(this.TimerArr[:i], this.TimerArr[i+1:]...)
			break
		}
	}

	if changed == true {
		sort.Sort(this.TimerArr)
	}

	//fmt.Println("timer length:=", len(this.TimerArr))

	return timerInfo
}

func (this *QPTimer) RemoveBySeatNum(value int32) []*QPTimerItem {

	timerInfo := make([]*QPTimerItem, 0)

	changed := false
	for isOver := false; isOver == false; {
		isOver = true
		for i, v := range this.TimerArr {
			if v.SeatNum != value {
				continue
			}
			timerInfo = append(timerInfo, v)
			changed = true
			v.Timer.Stop()
			isOver = false
			this.TimerArr = append(this.TimerArr[:i], this.TimerArr[i+1:]...)
			break
		}
	}

	if changed == true {
		sort.Sort(this.TimerArr)
	}

	//fmt.Println("timer length:=", len(this.TimerArr))

	return timerInfo
}

//func (this *QPTimer) RemoveTimer(timerID int32) {
//	n := this.timerList.Front()
//
//	for ; n != nil; n = n.Next() {
//		if n.Value.(*QPTimerItem).TimerID == timerID {
//			this.timerList.Remove(n)
//			return
//		}
//	}
//}
