package qpTable

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"math"
	"qpGame/commonDefine/mateProto"
	"qpGame/commonDefine/mateProto/protoGameBasic"
	"qpGame/db"
	"time"
)

type TableConfigJson struct {
	// 托管时间 0(不托管)\15\20\30 秒
	TuoGuanTime int32 `json:"tuoGuanTime"`
	// 首局超时准备 0(不做超时处理)\15(15秒后自动准备)\30(30秒后自动准备)\16(16秒后未准备踢出)\31(31秒后未准备踢出)
	FRReadyTimeout int32 `json:"readyTimeout"`
	// 解散方式 0(不允许解散) 1(超过一半解散) 2(全部同意)
	JieSanMode int32 `json:"jieSanMode"`
	// 托管局数 0(不托管)\1(1局结束)\3(3局结束)\100(100局结束)
	TuoGuanRoundCount int32 `json:"tuoGuanRound"`

	IsIPCheck     bool `json:"IPCheck"`  // 是否IP检测
	IsCRCheck     bool `json:"CRCheck"`  // 是否近距离检查
	LimitDistance int  `json:"distance"` // 距离限制
	BrokeType     int  `json:"broke"`    // 破产 0:按牌型分算  1:不可负分

	//IsStopVoice     bool `json:"voice"`     // 禁止语音
	//IsStopEmoji     bool `json:"emoji"`     // 禁止表情
	//IsShowClubScore bool `json:"showScore"` // 隐藏俱乐部分
}

type TableRuleConfig struct {
	TableConfigJson
	Table        QPGameTable
	TableCfgJson string
}

func ParseTableConfig(cfg string) (*TableRuleConfig, error) {
	v := &TableRuleConfig{}
	err := json.Unmarshal([]byte(cfg), v)
	if err != nil {
		return nil, err
	}

	switch v.TuoGuanTime {
	case 0:
	case 15:
	case 20:
	case 30:
	default:
		return nil, fmt.Errorf("tuoGuanTime := %d", v.TuoGuanTime)
	}

	switch v.FRReadyTimeout {
	case 0:
	case 10:
	case 11:
	case 15:
	case 16:
	case 30:
	case 31:
	default:
		return nil, fmt.Errorf("readyTimeout := %d", v.FRReadyTimeout)
	}

	if v.JieSanMode != 0 && v.JieSanMode != 1 && v.JieSanMode != 2 {
		return nil, fmt.Errorf("jieSanMode := %d", v.JieSanMode)
	}

	switch v.TuoGuanRoundCount {
	case 0, 1, 2, 3, 100:
	default:
		return nil, fmt.Errorf("tuoGuanRound := %d", v.TuoGuanRoundCount)
	}

	switch v.LimitDistance {
	case 0, 100, 300, 800:
	default:
		return nil, fmt.Errorf("distance := %d", v.LimitDistance)
	}

	v.TableCfgJson = cfg
	return v, nil
}

func (this *TableRuleConfig) CheckGPS(uid int64, newLat, newLng float64) int32 {
	if this.IsCRCheck == false {
		return 0
	}

	if newLat < 1.0 || newLng < 1.0 {
		return mateProto.Err_GPSNotOpen
	}

	baseTable := this.Table.GetBaseQPTable()
	if this.Table.GetBaseQPTable().MaxPlayers < 3 {
		return 0
	}

	if this.Table.GetBaseQPTable().GetCurSeatCount() < 1 {
		return 0
	}

	funcGetRadian := func(d float64) float64 {
		return (d * math.Pi) / 180.0 //角度1? = π / 180
	}

	//计算距离 米
	funcGetDistance := func(lat1, lng1, lat2, lng2 float64) float64 {
		radLat1 := funcGetRadian(lat1)
		radLat2 := funcGetRadian(lat2)
		a := radLat1 - radLat2
		b := funcGetRadian(lng1) - funcGetRadian(lng2)

		dst := 2 * math.Asin(math.Sqrt(math.Pow(math.Sin(a/2), 2)+math.Cos(radLat1)*math.Cos(radLat2)*math.Pow(math.Sin(b/2), 2)))
		dst = dst * 6378.137 * 1000

		return dst
	}

	baseTable.GpsInfo = make([]PlayerPosInfo, 0, 6)
	for i, v := range baseTable.SeatArr {
		if v == nil {
			continue
		}
		a := v.GetSeatData()

		distance := funcGetDistance(a.Lat, a.Lng, newLat, newLng)
		if distance < float64(this.LimitDistance) {
			return mateProto.Err_FindGPSFail
		}
		//glog.Warning(distance, ",", a.Lat, ",", a.Lng, ",", newLat, ",", newLng)

		for j := i + 1; j < len(baseTable.SeatArr); j++ {
			if baseTable.SeatArr[j] == nil {
				continue
			}
			b := baseTable.SeatArr[j].GetSeatData()
			if a.Player.ID == b.Player.ID {
				continue
			}

			distance := funcGetDistance(a.Lat, a.Lng, b.Lat, b.Lng)
			baseTable.GpsInfo = append(baseTable.GpsInfo,
				PlayerPosInfo{AUID: int64(a.Player.ID), BUID: int64(b.Player.ID), Distance: distance})
		}
		baseTable.GpsInfo = append(baseTable.GpsInfo,
			PlayerPosInfo{AUID: int64(a.Player.ID), BUID: uid, Distance: distance})
	}

	return 0
}

func (this *TableRuleConfig) CheckIP(ip string) bool {
	if this.IsIPCheck == false {
		return true
	}
	if this.Table.GetBaseQPTable().MaxPlayers < 3 {
		return true
	}

	if this.Table.GetBaseQPTable().GetCurSeatCount() < 1 {
		return true
	}

	for _, v := range this.Table.GetBaseQPTable().SeatArr {
		if v == nil {
			continue
		}

		if v.GetSeatData().Player.IP == ip {
			return false
		}
	}

	return true
}

func (this *TableRuleConfig) CheckPlayerMutex(uid int64) (map[int64]bool, bool) {
	mutexMap, err := db.GetPlayerMutex(uid)
	if err != nil {
		glog.Warning("GetPlayerMutex() err.err:=", err.Error(), " ,uid:=", uid)
		return mutexMap, false
	}

	for _, v := range this.Table.GetBaseQPTable().SeatArr {
		if v == nil {
			continue
		}
		uid := v.GetSeatData().Player.ID
		if _, ok := mutexMap[int64(uid)]; ok == true {
			return nil, false
		}
	}

	return mutexMap, true
}

func (this *TableRuleConfig) OnJoinTable() {
	if this.Table.GetBaseQPTable().GetCurSeatCount() != this.Table.GetBaseQPTable().MaxPlayers {
		return
	}
	switch this.FRReadyTimeout {
	case 10, 15, 30:
		this.Table.GetBaseQPTable().FirstRoundReadTime = time.Now().Unix()
		this.Table.GetBaseQPTable().GameTimer.PutTableTimer(protoGameBasic.TIMER_FirstRoundReady, this.FRReadyTimeout*1000, func() {

			if this.Table.GetBaseQPTable().IsAssignTableState(TS_WaitingPlayerEnter) == false {
				return
			}
			for _, v := range this.Table.GetBaseQPTable().SeatArr {
				if v == nil {
					continue
				}
				if v.GetSeatData().IsAssignSeatState(SS_Ready) == true {
					continue
				}

				msgReady := mateProto.MessageMaTe{
					SenderID:  int64(v.GetSeatData().Player.ID),
					MessageID: protoGameBasic.ID_GameReady,
				}

				this.Table.GetBaseQPTable().RootTable.OnMessage(&msgReady)
			}
		})
	case 11, 16, 31:
		this.Table.GetBaseQPTable().GameTimer.PutTableTimer(protoGameBasic.TIMER_Leave, (this.FRReadyTimeout-1)*1000, func() {
			if this.Table.GetBaseQPTable().IsAssignTableState(TS_WaitingPlayerEnter) == false {
				return
			}
			for _, v := range this.Table.GetBaseQPTable().SeatArr {
				if v == nil {
					continue
				}
				if v.GetSeatData().IsAssignSeatState(SS_Ready) == true {
					continue
				}
				msgLeaveTable := mateProto.MessageMaTe{
					SenderID:  int64(v.GetSeatData().Player.ID),
					MessageID: protoGameBasic.ID_ReqLeaveTable,
				}

				this.Table.GetBaseQPTable().RootTable.OnMessage(&msgLeaveTable)
			}
		})
	}
}

func (this *TableRuleConfig) OnDissolveTableVote() {
	// 检测是否 可以取消 解散状态
	agree, oppose := this.Table.GetBaseQPTable().DissolveTableStatus()
	//if (agree + oppose) < this.Table.GetBaseQPTable().GetCurSeatCount() {
	//	return
	//}

	dissolveSuccessFunc := func() {
		baseTable := this.Table.GetBaseQPTable()

		baseTable.DissolveType = DT_Vote
		baseTable.DelTableState(TS_Dissolve)
		baseTable.GameTimer.RemoveByTimeID(protoGameBasic.TIMER_DissolveTable)
		this.Table.GetBaseQPTable().SendToAllPlayer(protoGameBasic.ID_DissolveTableVoteReslut,
			&protoGameBasic.DissolveTableVoteResult{IsDissolveTable: true})

		baseTable.SetTableState(TS_Invalid)
	}

	dissolveFailedFunc := func() {
		baseTable := this.Table.GetBaseQPTable()
		baseTable.DelTableState(TS_Dissolve)
		baseTable.GameTimer.RemoveByTimeID(protoGameBasic.TIMER_DissolveTable)

		for _, v := range baseTable.SeatArr {
			if v == nil {
				continue
			}
			v.GetSeatData().DissolveVote = 0
		}
		baseTable.SendToAllPlayer(protoGameBasic.ID_DissolveTableVoteReslut,
			&protoGameBasic.DissolveTableVoteResult{IsDissolveTable: false})
	}

	switch this.JieSanMode {
	case 1:
		if agree > this.Table.GetBaseQPTable().GetCurSeatCount()/2 {
			dissolveSuccessFunc()
			return
		}
		if (agree + oppose) < this.Table.GetBaseQPTable().GetCurSeatCount() {
			return
		}

		dissolveFailedFunc()
	case 2:
		if agree == this.Table.GetBaseQPTable().GetCurSeatCount() {
			dissolveSuccessFunc()
			return
		}
		if oppose > 0 {
			dissolveFailedFunc()
			return
		}
	default:

	}
}

func (this *TableRuleConfig) TimerAutoReady() {
	baseTable := this.Table.GetBaseQPTable()

	baseTable.GameTimer.PutTableTimer(protoGameBasic.TIMER_AutoRedy, 8*1000, func() {

		if baseTable.IsAssignTableState(TS_WaitingPlayerEnter) == false &&
			baseTable.IsAssignTableState(TS_WaitingReady) == false {
			return
		}

		for _, v := range baseTable.SeatArr {
			if v == nil {
				continue
			}
			if v.GetSeatData().IsContainSeatState(SS_Ready|SS_Playing|SS_Looker) == true {
				continue
			}

			msgReady := mateProto.MessageMaTe{
				SenderID:  int64(v.GetSeatData().Player.ID),
				MessageID: protoGameBasic.ID_GameReady,
			}

			baseTable.RootTable.OnMessage(&msgReady)
		}
	})
}
