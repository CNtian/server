package management

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"qpGame/commonDefine/mateProto"
	"qpGame/commonDefine/mateProto/protoGameBasic"
	"qpGame/commonDefine/mateProto/protoInnerServer"
	"qpGame/game/tableFactory"
	"qpGame/localConfig"
	"qpGame/wrapMQ"
	"runtime"
)

// 处理MQ数据
func HandleMQData(data []byte) {

	msg := &mateProto.MessageMaTe{}
	err := json.Unmarshal(data, &msg)
	if err != nil {
		glog.Warning("HandleNetMessage() error.", err.Error())
		return
	}

	//commonDef.LOG_Info("recv uid:=", msg.SenderID, ",msgID:=", msg.MessageID, ",from :=", msg.Source)

	switch msg.MessageID {
	case protoGameBasic.ID_ChangeGameServiceStatus:
		UpdateServiceStatus(msg)
	case protoInnerServer.ID_HallServiceLaunch:
		LoginToHallService("hall")
	case protoInnerServer.ID_ClubServiceLaunch:
		LoginToClubService(msg)
	case protoGameBasic.ID_PrivateCreateGameTable:
		rspCode, tablePtr := onPrivateCreateTable(msg)
		var tempValue []byte
		if rspCode >= 0 {
			table := tablePtr.GetBaseQPTable()
			joinRspData := protoGameBasic.SC_JoinTable{GameID: table.GameID, ClubPlayID: table.ClubPlayID, TableID: table.TableNum}
			tempValue, _ = json.Marshal(&joinRspData)
		}
		msg.MsgBody = nil
		wrapMQ.ReplyToSource(msg, &protoGameBasic.JsonResponse{Status: rspCode, Data: tempValue})
	case protoGameBasic.ID_PrivateJoinGameTable:
		rspCode := onPrivateJoinGameTable(msg)
		if rspCode != 0 {
			wrapMQ.ReplyToSource(msg, &protoGameBasic.JsonResponse{Status: rspCode, Data: msg.MsgBody})
		}
	case protoGameBasic.ID_ClubCreateTable:
		rspCode, tablePtr := onCreateClubTable(msg)

		var tempValue []byte
		if rspCode >= 0 {
			table := tablePtr.GetBaseQPTable()
			joinRspData := protoGameBasic.SC_JoinTable{GameID: table.GameID, ClubPlayID: table.ClubPlayID, TableID: table.TableNum}
			tempValue, _ = json.Marshal(&joinRspData)
		}
		msg.MsgBody = nil
		wrapMQ.ReplyToSource(msg, &protoGameBasic.JsonResponse{Status: rspCode, Data: tempValue})
	case protoGameBasic.ID_ClubJoinTable:
		rspCode := onJoinClubTable(msg)
		if rspCode != 0 {
			wrapMQ.ReplyToSource(msg, &protoGameBasic.JsonResponse{Status: rspCode, Data: msg.MsgBody})
		}
	case protoGameBasic.ID_ForceDissolveTable:
		rspCode := onForceDissolveTable(msg)
		if rspCode != 0 {
			wrapMQ.ReplyToSource(msg, &protoGameBasic.JsonResponse{Status: rspCode, Data: msg.MsgBody})
		}
	case protoInnerServer.ID_GameSignIn:
		return
	case protoGameBasic.ID_HelpPutClubPlay:
		checkPutClubPlay(msg)
	default:
		rootTab := getPlayer(msg.SenderID)
		if rootTab != nil {
			rootTab.msgChan <- msg
		} else {
			//commonDef.LOG_Info("uid:=", msg.SenderID, " not find table. msgID:=", msg.MessageID)

			ok, err := deletePlayer(msg.SenderID)
			if ok == true && err == nil {
				msg.MessageID = protoGameBasic.ID_PlayerNotInGame
				wrapMQ.ReplyToSource(msg, nil)
				return
			}
			glog.Error("find player but not find table. UID:=", msg.SenderID)
		}
	}
}

// 通知 俱乐部服务  游戏服 已启动
func LoginToClubService(msg *mateProto.MessageMaTe) {
	playIDArr := make([]protoInnerServer.SignInInfo, 0)

	for _, v := range localConfig.GetConfig().SupportPlaying {
		playIDArr = append(playIDArr, protoInnerServer.SignInInfo{PlayID: v.PlayingID, PlayName: v.Name})
	}

	replyMsg := mateProto.MessageMaTe{Source: localConfig.GetConfig().ID,
		To:        msg.Source,
		MessageID: protoInnerServer.ID_GameSignIn}

	err := wrapMQ.SendMsgTo(&replyMsg,
		&protoInnerServer.MsgGameSignIn{
			SupportPlayIDArr: playIDArr,
			Status:           int32(serviceStatus),
			Port_pprof:       localConfig.GetConfig().PprofPort})
	if err != nil {
		glog.Error("LoginToOtherService() queueName：=", msg.Source, ", err:=", err.Error())
	}

	defer func() {
		err := recover()
		if err == nil {
			return
		}

		buf := new(bytes.Buffer)
		fmt.Fprintf(buf, "%v\n", err)
		for i := 1; ; i++ {
			pc, file, line, ok := runtime.Caller(i)
			if !ok {
				break
			}
			fmt.Fprintf(buf, "%s:%d (0x%x)\n", file, line, pc)
		}
		glog.Warning("LoginToOtherService(),table exception.err:=", err, "\n", buf.String())
	}()

	// 多个 go 同时操作桌子 , 可能会引发 异常 todo...
	tableArr := make([]protoGameBasic.PushTable, 0, 100)
	tableMap.Range(func(key, value interface{}) bool {
		gameTable := value.(*rootTable).gameTable.GetBaseQPTable()

		if gameTable.MZClubID < 1 {
			return true
		}

		tableInfo := protoGameBasic.PushTable{
			ClubID:      gameTable.MZClubID,
			TableNumber: gameTable.TableNum,
			ClubPlayID:  gameTable.ClubPlayID,
			GameID:      gameTable.GameID,
			MaxPlayers:  gameTable.MaxPlayers,
			//CreateTime:  gameTable.CreateTime,
			ServiceID: localConfig.GetConfig().ID,
			UIDArr:    make([]int64, 0, 10),
		}

		seatArrBak := gameTable.SeatArr

		for _, v := range seatArrBak {
			tempBak := v
			if tempBak == nil {
				continue
			}
			tableInfo.UIDArr = append(tableInfo.UIDArr, int64(tempBak.GetSeatData().Player.ID))
		}

		if gameTable.CurXRound > 0 {
			tableInfo.Status = 1
		}

		tableArr = append(tableArr, tableInfo)

		return true
	})

	msg1 := mateProto.MessageMaTe{Source: localConfig.GetConfig().ID,
		To:        msg.Source,
		MessageID: protoGameBasic.ID_PushAllTable}

	err = wrapMQ.SendMsgTo(&msg1,
		&protoGameBasic.SS_PushAllTable{TableArr: tableArr})
	if err != nil {
		glog.Error("LoginToOtherService() queueName：=", msg.Source, ", err:=", err.Error())
	}
}

// 通知 大厅服务  游戏服 已启动
func LoginToHallService(queueName string) {
	playIDArr := make([]protoInnerServer.SignInInfo, 0)

	for _, v := range localConfig.GetConfig().SupportPlaying {
		playIDArr = append(playIDArr, protoInnerServer.SignInInfo{PlayID: v.PlayingID, PlayName: v.Name})
	}

	msg := mateProto.MessageMaTe{Source: localConfig.GetConfig().ID,
		To:        queueName,
		MessageID: protoInnerServer.ID_GameSignIn}

	err := wrapMQ.SendMsgTo(&msg,
		&protoInnerServer.MsgGameSignIn{
			SupportPlayIDArr: playIDArr,
			Status:           int32(serviceStatus),
			Port_pprof:       localConfig.GetConfig().PprofPort})
	if err != nil {
		glog.Error("LoginToOtherService() queueName：=", queueName, ", err:=", err.Error())
	}

	defer func() {
		err := recover()
		if err == nil {
			return
		}

		buf := new(bytes.Buffer)
		fmt.Fprintf(buf, "%v\n", err)
		for i := 1; ; i++ {
			pc, file, line, ok := runtime.Caller(i)
			if !ok {
				break
			}
			fmt.Fprintf(buf, "%s:%d (0x%x)\n", file, line, pc)
		}
		glog.Warning("LoginToOtherService(),table exception.err:=", err, "\n", buf.String())
	}()

	// 多个 go 同时操作桌子 , 可能会引发 异常 todo...
	tableArr := make([]protoGameBasic.HallRecoverTable, 0, 100)
	tableMap.Range(func(key, value interface{}) bool {
		gameTable := value.(*rootTable).gameTable.GetBaseQPTable()
		seatArrBak := gameTable.SeatArr

		tableInfo := protoGameBasic.HallRecoverTable{
			TableID: gameTable.TableNum,
			GameID:  gameTable.GameID,
			Players: make([]int64, 0, gameTable.GetCurSeatCount()),
		}

		for _, v := range seatArrBak {
			tempBak := v
			if tempBak == nil {
				continue
			}
			tableInfo.Players = append(tableInfo.Players, int64(tempBak.GetSeatData().Player.ID))
		}

		tableArr = append(tableArr, tableInfo)

		return true
	})

	msg1 := mateProto.MessageMaTe{Source: localConfig.GetConfig().ID,
		To:        queueName,
		MessageID: protoGameBasic.ID_HallRecoverTable}

	err = wrapMQ.SendMsgTo(&msg1,
		&protoGameBasic.SS_RecoverTable{tableArr})
	if err != nil {
		glog.Error("LoginToHallService() queueName：=", queueName, ", err:=", err.Error())
	}
}

// 广播服务器状态
func BroadGameServiceStatus() {
	msg := mateProto.MessageMaTe{Source: localConfig.GetConfig().ID,
		To:        "hall",
		MessageID: protoInnerServer.ID_BroadGameServiceStatus}

	msgStatus := protoInnerServer.MsgBroadGameServiceStatus{
		Status:     int32(serviceStatus),
		TableTotal: curTableCount}

	err := wrapMQ.SendMsgTo(&msg, &msgStatus)
	if err != nil {
		glog.Error("BroadGameServiceStatus() err:=", err.Error())
	}

	msg.To = "club"
	err = wrapMQ.SendMsgTo(&msg, &msgStatus)
	if err != nil {
		glog.Error("BroadGameServiceStatus() err:=", err.Error())
	}
}

func UpdateServiceStatus(msg *mateProto.MessageMaTe) {
	var cmd protoGameBasic.SS_CMD

	err := json.Unmarshal(msg.Data, &cmd)
	if err != nil {
		glog.Warning("/UpdateServiceStatus err:=", err.Error())
		return
	}

	SetServiceStatus(cmd.Status)
}

func checkPutClubPlay(msg *mateProto.MessageMaTe) {
	msgBody := protoGameBasic.CS_PutClubPlay{}
	err := json.Unmarshal(msg.Data, &msgBody)
	if err != nil {
		glog.Warning(err.Error())
		return
	}
	reply := protoGameBasic.JsonResponse{}
	reply.Status, reply.Detail = checkPlayOpt(msgBody.PlayID, msgBody.PlayCfg, msgBody.TableCfg, msgBody.ClubCfgText)
	reply.Data = msg.Data

	msg.MessageID = protoGameBasic.ID_PutClubPlay_RPC
	err = wrapMQ.SendMsgToClub(msg, &reply)
	if err != nil {
		glog.Warning(err.Error())
	}
}

func checkPlayOpt(playID int32, playOpt, tableOpt, clubOpt string) (int32, string) {
	_, rspCode, details := tableFactory.NewGameTable(0, playID, playOpt, tableOpt, clubOpt)
	//if table != nil && rspCode == 0 {
	//	return rspCode, details
	//}
	return rspCode, details
}
