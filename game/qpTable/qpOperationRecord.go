package qpTable

import "encoding/json"

//type RecSeatInfo struct {
//	SeatScore  int64  `json:"seatScore"`
//	HeadURL    string `json:"headURL"`
//	NickName   string `json:"nickName"`
//	SeatNumber int32  `json:"seatNumber"`
//	UID        int64  `json:"uid"`
//}

type RecPlayerOperation struct {
	SeatNumber int32       `json:"s"` // 座位号
	MsgID      int32       `json:"id"`
	MsgBody    interface{} `json:"bo"`
}

type RecBroadcastOperation struct {
	MsgID   int32       `json:"id"`
	MsgBody interface{} `json:"bo"`
}

type RecServiceOperation struct {
	SeatNumber int32       `json:"s"`
	MsgID      int32       `json:"id"`
	MsgBody    interface{} `json:"bo"`
}

type RecGameMessage struct {
	Type    string      `json:"t"` // c:客户端  b:广播  s:服务端
	Message interface{} `json:"m"`
}

type PlayerOperRec struct {
	TableNumber int32            `json:"tableNumber"`
	CurRound    int32            `json:"curRound"`
	PlayCfg     string           `json:"playCfg"`
	TableCfg    string           `json:"tableCfg"`
	SeatInfo    []*SeatData      `json:"seatInfo"`
	StepList    []RecGameMessage `json:"step"`
}

func (this *PlayerOperRec) SetTableInfo(tableNumber, curRound int32, playCfg, tableCfg string) {

	// 值 还原
	this.SeatInfo = make([]*SeatData, 0, 10)
	this.StepList = make([]RecGameMessage, 0, 200)

	this.TableNumber = tableNumber
	this.CurRound = curRound
	this.PlayCfg = playCfg
	this.TableCfg = tableCfg
}

func (this *PlayerOperRec) PutPlayer(seat *SeatData) {
	this.SeatInfo = append(this.SeatInfo, seat)
}

func (this *PlayerOperRec) DelPlayer(uid int64) {
	for i, v := range this.SeatInfo {
		if int64(v.Player.ID) == uid {
			this.SeatInfo = append(this.SeatInfo[:i], this.SeatInfo[i+1:]...)
			return
		}
	}
}

func (this *PlayerOperRec) PutPlayerStep(seatNumber, msgID int32, msgBody interface{}) {
	//if len(this.SeatInfo) < 1 {
	//	return
	//}

	rec := RecGameMessage{Type: "c",
		Message: &RecPlayerOperation{SeatNumber: seatNumber, MsgID: msgID, MsgBody: msgBody}}
	this.StepList = append(this.StepList, rec)
}

func (this *PlayerOperRec) PutBroadStep(msgID int32, msgBody interface{}) {
	//if len(this.SeatInfo) < 1 {
	//	return
	//}

	rec := RecGameMessage{Type: "b",
		Message: &RecBroadcastOperation{MsgID: msgID, MsgBody: msgBody}}
	this.StepList = append(this.StepList, rec)
}

func (this *PlayerOperRec) PutServiceStep(seatNumber int32, msgID int32, msgBody interface{}) {
	//if len(this.SeatInfo) < 1 {
	//	return
	//}

	rec := RecGameMessage{Type: "s",
		Message: &RecServiceOperation{SeatNumber: seatNumber, MsgID: msgID, MsgBody: msgBody}}
	this.StepList = append(this.StepList, rec)
}

func (this *PlayerOperRec) Pack() ([]byte, error) {
	return json.Marshal(this)
}
