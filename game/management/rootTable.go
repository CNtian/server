package management

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"io/ioutil"
	"math/rand"
	commonDef "qpGame/commonDefine"
	"qpGame/commonDefine/mateProto"
	"qpGame/commonDefine/mateProto/protoGameBasic"
	"qpGame/commonDefine/mateProto/protoInnerServer"
	"qpGame/db"
	"qpGame/game/tableFactory"
	"qpGame/qpTable"
	"qpGame/wrapMQ"
	"runtime"
	"sync/atomic"
	"time"
)

type rootTable struct {
	gameTable qpTable.QPGameTable
	msgChan   chan *mateProto.MessageMaTe

	noticeClub bool // 是否通知过俱乐部
}

func (this *rootTable) ReleaseResource() {

	baseTable := this.gameTable.GetBaseQPTable()
	if baseTable.MZClubID > 0 && this.noticeClub == true {
		// 转发给 俱乐部
		var msg mateProto.MessageMaTe
		msg.To, msg.MessageID = fmt.Sprintf("%d", baseTable.MZClubID), protoGameBasic.ID_TableDelete
		msgBody := protoGameBasic.SS_DelTable{
			ClubID:      baseTable.MZClubID,
			GameID:      baseTable.GameID,
			TableNumber: baseTable.TableNum,
		}

		err := wrapMQ.SendMsgTo(&msg, &msgBody)
		if err != nil {
			glog.Warning("TableDelete() err. err :=", err.Error())
		}
	}

	tableMap.Delete(baseTable.TableNum)

	seatArr := make([]int64, 0, 4)
	for _, v := range baseTable.SeatArr {
		if v != nil {
			// 观战玩家 提前离场了
			if v.GetSeatData().IsLeave != 0 {
				continue
			}
			playerID := int64(v.GetSeatData().Player.ID)
			seatArr = append(seatArr, playerID)
			_, err := deletePlayer(playerID)
			if err != nil {
				glog.Warning("deletePlayer() err:=", err.Error(), " ,uid:=", playerID)
			}
		}
	}
	err := db.RemoveTableInfo(baseTable.TableNum)
	if err != nil {
		glog.Warning("RemoveTableInfo() err:=", err.Error())
	}

	NoticeHallDeleteTable(baseTable.TableNum, seatArr)

	if this.msgChan != nil {
		close(this.msgChan)
		this.msgChan = nil
	}
}

func (this *rootTable) GetBaseQPTable() *qpTable.QPTable {
	return this.gameTable.GetBaseQPTable()
}

func (this *rootTable) GetMaxRound() int32 {
	return this.gameTable.GetMaxRound()
}

func (this *rootTable) OnMessage(msg *mateProto.MessageMaTe) int32 {

	var rspCode int32

	//t1 := time.Now()

	switch msg.MessageID {
	case protoGameBasic.ID_PrivateJoinGameTable:
		if this.gameTable.GetBaseQPTable().GetCurSeatCount() < 1 {
			return this.onPlayerJoinPrivateTable(msg)
		} else {
			rspCode = this.onPlayerJoinPrivateTable(msg)
		}
	case protoGameBasic.ID_ClubJoinTable:
		if this.gameTable.GetBaseQPTable().GetCurSeatCount() < 1 {
			return this.onPlayerJoinClubTable(msg)
		} else {
			rspCode = this.onPlayerJoinClubTable(msg)
		}
	case protoGameBasic.ID_ReqLeaveTable:
		rspCode = this.onPlayerLeave(msg)
	case protoGameBasic.ID_ForceDissolveTable:
		baseTable := this.gameTable.GetBaseQPTable()
		baseTable.DissolveType = qpTable.DT_Enforcement
		baseTable.SetTableState(qpTable.TS_Invalid)
		rspCode = this.gameTable.OnMessage(msg)
	case protoGameBasic.ID_LookerLeave:
		rspCode = this.onLookerLeave(msg)
	default:
		rspCode = this.gameTable.OnMessage(msg)
	}
	if rspCode < 0 {
		wrapMQ.ReplyToSource(msg, &protoGameBasic.JsonResponse{Status: rspCode})
		//glog.Warning("debug table:=", this.gameTable.GetBaseQPTable().TableNum,
		//	",rspCode:=", rspCode, ",senderID:=", msg.SenderID, ",msgID:=", msg.MessageID, ",data:=", string(msg.Data))
	}
	//sub := time.Now().Sub(t1).Milliseconds()
	//if sub > 50 && msg.MessageID != 3003 { // 跑得快出牌有sleep
	//	glog.Warning("expend time too long.", sub, "ms",
	//		",gameID:=", this.gameTable.GetBaseQPTable().GameID, ",tableID:=", this.gameTable.GetBaseQPTable().TableNum,
	//		",round:=", this.gameTable.GetBaseQPTable().CurXRound,
	//		",MsgID:=", msg.MessageID)
	//}

	return rspCode
}

func (this *rootTable) onPlayerJoinPrivateTable(msg *mateProto.MessageMaTe) int32 {

	joinTable := protoGameBasic.CS_PrivateJoinGameTable{}
	err := json.Unmarshal(msg.Data, &joinTable)
	if err != nil {
		return mateProto.Err_ProtocolDataErr
	}
	msg.MsgBody = &joinTable

	rspCode := this.gameTable.OnMessage(msg)

	baseTable := this.GetBaseQPTable()
	// 避免创建时,通知
	if baseTable.GetCurSeatCount() > 1 {
		joinRspData := protoGameBasic.SC_JoinTable{GameID: baseTable.GameID, ClubPlayID: baseTable.ClubPlayID, TableID: baseTable.TableNum}
		tempValue, _ := json.Marshal(&joinRspData)
		wrapMQ.ReplyToSource(msg, &protoGameBasic.JsonResponse{Status: rspCode, Data: tempValue})
		NoticeHallPlayerJoinTable(baseTable.TableNum, msg.SenderID)
	}

	if rspCode < 0 {
		return rspCode
	}

	putPlayer(msg.SenderID, this)

	{
		seatData := baseTable.SeatArr[rspCode].GetSeatData()
		sendMsg := protoGameBasic.BroadcastNewPlayerJoin{
			UID:     int64(seatData.Player.ID),
			SeatNum: rspCode,
			Status:  uint32(seatData.Status),
			IP:      seatData.Player.IP,
			Head:    seatData.Player.Head, Nick: seatData.Player.Nick, Sex: seatData.Player.Sex,
			ClubScore: commonDef.Float64ToString(seatData.ClubScore),
		}
		if seatData.Lng > 1.0 && seatData.Lat > 1.0 {
			sendMsg.Location = true
		}
		baseTable.SendToAllPlayer(protoGameBasic.ID_NewPlayerJoin, &sendMsg)
	}

	return rspCode
}

func (this *rootTable) onPlayerJoinClubTable(msg *mateProto.MessageMaTe) int32 {
	msgSource := msg.Source

	clubJoinTable := protoGameBasic.CS_ClubJoinTable{}
	err := json.Unmarshal(msg.Data, &clubJoinTable)
	if err != nil {
		return mateProto.Err_ProtocolDataErr
	}
	msg.MsgBody = &clubJoinTable

	rspCode := this.gameTable.OnMessage(msg)

	baseTable := this.GetBaseQPTable()
	// 避免创建时,通知
	if baseTable.GetCurSeatCount() > 1 {
		joinRspData := protoGameBasic.SC_JoinTable{GameID: baseTable.GameID, ClubPlayID: baseTable.ClubPlayID, TableID: baseTable.TableNum}
		tempValue, _ := json.Marshal(&joinRspData)
		wrapMQ.ReplyToSource(msg, &protoGameBasic.JsonResponse{Status: rspCode, Data: tempValue})
		NoticeHallPlayerJoinTable(baseTable.TableNum, msg.SenderID)
	}

	if rspCode < 0 {
		return rspCode
	}

	seatData := baseTable.SeatArr[rspCode].GetSeatData()
	seatData.ClubID = clubJoinTable.ClubID

	putPlayer(msg.SenderID, this)

	{
		noticeGateway := mateProto.MessageMaTe{SenderID: msg.SenderID,
			MessageID: protoInnerServer.ID_NotiePlayerInTable,
			To:        msgSource}
		err = wrapMQ.SendMsgTo(&noticeGateway, nil)
		if err != nil {
			glog.Warning("TablePutPlayer() err. err :=", err.Error())
		}
	}

	// 首次 不用通知
	if baseTable.GetCurSeatCount() > 1 {
		// 转发给 俱乐部
		msg.To, msg.MessageID = fmt.Sprintf("%d", baseTable.MZClubID), protoGameBasic.ID_TablePutPlayer
		msgBody := protoGameBasic.SS_PutPlayerToTable{
			ClubID:      baseTable.MZClubID,
			GameID:      baseTable.GameID,
			TableNumber: baseTable.TableNum,
			UID:         msg.SenderID,
		}

		err = wrapMQ.SendMsgTo(msg, &msgBody)
		if err != nil {
			glog.Warning("TablePutPlayer() err. err :=", err.Error())
		}

		{
			sendMsg := protoGameBasic.BroadcastNewPlayerJoin{
				UID:     int64(seatData.Player.ID),
				SeatNum: rspCode,
				Status:  uint32(seatData.Status),
				IP:      seatData.Player.IP,
				Head:    seatData.Player.Head, Nick: seatData.Player.Nick, Sex: seatData.Player.Sex,
				ClubScore: commonDef.Float64ToString(seatData.ClubScore),
			}
			if seatData.Lng > 1.0 && seatData.Lat > 1.0 {
				sendMsg.Location = true
			}
			this.gameTable.GetBaseQPTable().SendToAllPlayer(protoGameBasic.ID_NewPlayerJoin, &sendMsg)
		}
	}

	return rspCode
}

func (this *rootTable) onPlayerLeave(msg *mateProto.MessageMaTe) int32 {
	rspCode := this.gameTable.OnMessage(msg)
	if rspCode < 0 {
		return rspCode
	}

	if rspCode == 0 {
		deletePlayer(msg.SenderID)
		NoticeHallPlayerLeaveTable(this.gameTable.GetBaseQPTable().TableNum, msg.SenderID)
	}

	baseTable := this.GetBaseQPTable()
	// 人数为0时,释放桌子
	if baseTable.GetCurSeatCount() < 1 {
		baseTable.SetTableState(qpTable.TS_Invalid)
	} else if baseTable.MZClubID > 0 {

		// 转发给 俱乐部
		msg.To, msg.MessageID = fmt.Sprintf("%d", baseTable.MZClubID), protoGameBasic.ID_TableDelPlayer
		msgBody := protoGameBasic.SS_DelPlayerInTable{
			ClubID:      baseTable.MZClubID,
			GameID:      baseTable.GameID,
			TableNumber: baseTable.TableNum,
			UID:         msg.SenderID,
		}

		err := wrapMQ.SendMsgTo(msg, &msgBody)
		if err != nil {
			glog.Warning("TablePutPlayer() err. err :=", err.Error())
		}
	}

	return rspCode
}

func (this *rootTable) RecordGameOver() {
	table := this.gameTable.GetBaseQPTable()
	if table.CurXRound < 1 {
		return
	}

	recPlayerGameScoreArr := make([]*protoGameBasic.PlayerGameScore, 0, 3)

	for _, v := range table.SeatArr {
		if v == nil {
			continue
		}

		seat := v.GetSeatData()
		if seat.IsPlayed == false {
			continue
		}

		temp := protoGameBasic.PlayerGameScore{
			UID:     int64(seat.Player.ID),
			Nick:    seat.Player.Nick,
			ClubID:  seat.ClubID,
			SScore:  commonDef.Float64Mul1000ToService(seat.SeatScore),
			IsLeave: seat.IsLeave,
		}
		recPlayerGameScoreArr = append(recPlayerGameScoreArr, &temp)

		v.CleanRoundData()
	}

	// 大局记录
	gameOvercData := protoGameBasic.SS_GameOverRecord{
		RoundID:      table.RecordID,
		TableID:      table.TableNum,
		Begin:        table.CreateTime,
		End:          time.Now(),
		PlayerScore:  recPlayerGameScoreArr,
		GameID:       table.GameID,
		GameName:     tableFactory.GetPlayName(table.GameID),
		ClubID:       table.MZClubID,
		ClubPlayID:   table.ClubPlayID,
		PayPlayerID:  table.PayUID,
		ConsumeCount: table.Consumables,
		RuleRound:    this.GetMaxRound(),
		ActualRound:  table.CurXRound,
	}
	wrapMQ.SendMsgTo(&mateProto.MessageMaTe{To: "db", MessageID: protoGameBasic.ID_GameOver}, &gameOvercData)
}

func (this *rootTable) StartWork(tableNumber int32) {
	go func(tableNum int32) {
		defer func() {
			atomic.AddInt32(&curTableCount, -1)

			if err := recover(); err != nil {
				buf := new(bytes.Buffer)
				fmt.Fprintf(buf, "%v\n", err)
				for i := 1; ; i++ {
					pc, file, line, ok := runtime.Caller(i)
					if !ok {
						break
					}
					fmt.Fprintf(buf, "%s:%d (0x%x)\n", file, line, pc)
				}
				exceptionRec := fmt.Sprintf("tableNumber:=%d,  table exception :=%s \n", tableNum, buf.String())
				glog.Warning("tableNumber:=", tableNum, ",table exception.err:=", err, "\n", buf.String())

				recData, packErr := this.gameTable.GetBaseQPTable().OperateRec.Pack()
				if packErr != nil {
					glog.Warning("OperateRec.Pack() err. err:=", packErr.Error())
				} else {
					curTime := time.Now()
					fileName := fmt.Sprintf("%d_%d_%02d_%02d_%02d_%02d_%02d",
						this.gameTable.GetBaseQPTable().TableNum,
						curTime.Year(), curTime.Month(), curTime.Day(), curTime.Hour(), curTime.Minute(), curTime.Second())

					fileData := []byte(exceptionRec)
					fileData = append(fileData, recData...)
					writeErr := ioutil.WriteFile(fileName, fileData, 0666)
					if writeErr != nil {
						glog.Warning("OperateRec.Pack() err. err:=", writeErr.Error(), ",data:=", string(recData))
					}
				}
			}

			baseTable := this.gameTable.GetBaseQPTable()
			// 游戏未开局
			if baseTable.CurXRound < 1 && baseTable.Consumables > 0 {
				if baseTable.MZClubID > 0 {
					//db.ChangePlayerDiamondCount(baseTable.PayUID, baseTable.Consumables)
					db.ChangePlayerRoomCardCount(baseTable.PayUID, baseTable.Consumables)
				} else {
					db.ChangePlayerRoomCardCount(baseTable.PayUID, baseTable.Consumables)
				}
			}

			baseTable.RecordNonNormalGameOverRound()
			this.RecordGameOver()
			this.ReleaseResource()
		}()

		baseTable := this.GetBaseQPTable()
		baseTable.RootTable = this

		tableMap.Store(baseTable.TableNum, this)
		atomic.AddInt32(&curTableCount, 1)
		NoticeHallNewTable(baseTable.GameID, baseTable.TableNum, int64(baseTable.SeatArr[0].GetSeatData().Player.ID))

		var (
			gameTimeChan    <-chan time.Time
			tableExpireTime = 10
			tableExpireChan = time.After(time.Minute * 10) // 如果桌子没开始 10分钟
			robotTimer      <-chan time.Time
		)

		if baseTable.RobotInviteTimer > 0 {
			if baseTable.RobotJoinPlaying > 0 || baseTable.RobotJoinReady > 0 {
				if baseTable.RobotJoinPlaying > 0 {
					baseTable.RobotJoinPlaying += int32(rand.Intn(3))
				}
				if baseTable.RobotJoinReady > 0 {
					baseTable.RobotJoinReady += int32(rand.Intn(3))
				}
				robotTimer = time.After(time.Second * time.Duration(baseTable.RobotInviteTimer))
			}
		}

		for baseTable.IsAssignTableState(qpTable.TS_Invalid) == false {

			minTime := baseTable.GameTimer.GetMinTimer()
			if minTime != nil {
				gameTimeChan = minTime.Timer.C
			} else {
				gameTimeChan = nil
			}

			select {
			case gameMsg := <-this.msgChan:
				this.OnMessage(gameMsg)
			case <-gameTimeChan:
				minTime.DoFunc()
				baseTable.GameTimer.RemoveTimer(minTime)
			case <-tableExpireChan: // 到期 解散桌子
				// 10分钟后 桌子 是否 是 非等待玩家进入状体
				if tableExpireTime == 10 &&
					baseTable.IsAssignTableState(qpTable.TS_WaitingPlayerEnter) == false {
					// 桌子最长有效期 2个小时
					tableExpireChan = time.After(time.Hour * 2)
					tableExpireTime = 60 * 2
					break
				}
				this.gameTable.GetBaseQPTable().SetTableState(qpTable.TS_Invalid)

				msg := mateProto.MessageMaTe{
					MessageID: protoGameBasic.ID_TableExpire,
					MsgBody:   &protoGameBasic.CS_DissolveTable{TableNumber: tableNum},
				}
				this.OnMessage(&msg)
			case <-robotTimer:
				baseTable.TimerInvitationRobot()
				robotTimer = time.After(time.Second * time.Duration(baseTable.RobotInviteTimer))
			}
		} // for baseTable.IsAssignTableState(qpTable.TS_Invalid) == false {

		//// 游戏未开局
		//if baseTable.CurXRound < 1 && baseTable.Consumables > 0 {
		//	if baseTable.MZClubID > 0 {
		//		db.ChangePlayerDiamondCount(baseTable.PayUID, baseTable.Consumables)
		//	} else {
		//		db.ChangePlayerRoomCardCount(baseTable.PayUID, baseTable.Consumables)
		//	}
		//}

		//this.gameTable.GetBaseQPTable().RecordNonNormalGameOverRound()
		//this.RecordGameOver()
		//
		//// 释放 桌子
		//this.ReleaseResource()

	}(tableNumber)
}

func (this *rootTable) onLookerLeave(msg *mateProto.MessageMaTe) int32 {
	rsp := this.gameTable.OnMessage(msg)
	if rsp < 0 {
		return rsp
	}

	_, err := deletePlayer(msg.SenderID)
	if err != nil {
		glog.Warning("deletePlayer() err:=", err.Error(), " ,uid:=", msg.SenderID)
	}

	seat := this.gameTable.GetBaseQPTable().GetSeatDataByPlayerID(qpTable.PlayerID(msg.SenderID)).GetSeatData()
	db.ChangePlayerClubScore(seat.ClubID, msg.SenderID, commonDef.Float64Mul1000ToService(seat.SeatScore))

	wrapMQ.ReplyToSource(msg, &protoGameBasic.JsonResponse{Status: 0})
	return 0
}
