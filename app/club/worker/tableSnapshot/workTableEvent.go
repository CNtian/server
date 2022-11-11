package tableSnapshot

import (
	"encoding/json"
	"github.com/golang/glog"
	clubProto "vvService/appClub/protoDefine"
	"vvService/appClub/worker/clubEvent"
	"vvService/appClub/wrapMQ"
	"vvService/commonPackge/mateProto"
)

// 同步 PutNewTable()    同步 PutNewTable()    同步 PutNewTable()
func onRecoverTable(tableInfo *clubProto.PushTable) {

	clubTable, ok := clubMap[tableInfo.ClubID]
	if ok == false {
		clubTable = NewClubTableData(tableInfo.ClubID)
		clubMap[tableInfo.ClubID] = clubTable
	}

	newTable := idleDeskPool.Get()
	newTable.TableNum = tableInfo.TableNumber
	newTable.ClubPlayID = tableInfo.ClubPlayID
	newTable.MaxPlayers = tableInfo.MaxPlayers
	newTable.PlayerArr = make([]PlayerInfo, 0, 10)
	for _, v := range tableInfo.UIDArr {
		tPlayer := clubEvent.LoadPlayerNick_Name(v)
		newTable.PlayerArr = append(newTable.PlayerArr, PlayerInfo{v, tPlayer.HeadURL, tPlayer.Nick})
	}
	//newTable.PlayerArr = tableInfo.UIDArr
	newTable.GameID = tableInfo.GameID
	if tableInfo.Status == 1 {
		newTable.CurChangeCategory = playingCC
	}
	newTable.OnlinePlayers = int32(len(tableInfo.UIDArr))
	newTable.ServiceID = tableInfo.ServiceID
	newTable.Inc = clubTable.inc

	clubTable.playerCount += int32(len(tableInfo.UIDArr))
	clubTable.tableCount += 1
	clubTable.inc += 1

	// wait
	if tableInfo.Status == 0 {
		tablesMap, ok1 := clubTable.waitingTableMap[tableInfo.ClubPlayID]
		if ok1 == false {
			tablesMap = make(TableMap)
			clubTable.waitingTableMap[tableInfo.ClubPlayID] = tablesMap
		}
		tablesMap[newTable.TableNum] = newTable
	}

	// gameID
	groupTable, ok := clubTable.gameTableMap[tableInfo.GameID]
	if ok == false {
		groupTable = &groupTableInfo{tableMap: make(map[int32]*TableData)}
		clubTable.gameTableMap[tableInfo.GameID] = groupTable
	}
	groupTable.tableMap[newTable.TableNum] = newTable
	// gameID

	// clubPlayID
	groupTable, ok = clubTable.clubPlayMap[tableInfo.ClubPlayID]
	if ok == false {
		groupTable = &groupTableInfo{tableMap: make(map[int32]*TableData)}
		clubTable.clubPlayMap[tableInfo.ClubPlayID] = groupTable
	}
	groupTable.tableMap[newTable.TableNum] = newTable
	// clubPlayID

	clubTable.allTableMap[newTable.TableNum] = newTable

	clubTable.lastChangedTableMap[newTable.TableNum] = newTable
}

func onPutNewTable(msg *mateProto.MessageMaTe) {
	msgBody := clubProto.SS_PutNewTable{}
	err := json.Unmarshal(msg.Data, &msgBody)
	if err != nil {
		glog.Warning("onPutNewTable() error :=", err.Error())
		return
	}

	clubTable, ok := clubMap[msgBody.ClubID]
	if ok == false {
		clubTable = NewClubTableData(msgBody.ClubID)
		clubMap[msgBody.ClubID] = clubTable
	}

	clubTable.PutNewTable(&msgBody)
}

func onPutPlayer(msg *mateProto.MessageMaTe) {
	msgBody := clubProto.SS_PutPlayerToTable{}
	err := json.Unmarshal(msg.Data, &msgBody)
	if err != nil {
		glog.Warning("onPutPlayer() error :=", err.Error())
		return
	}

	clubTable, ok := clubMap[msgBody.ClubID]
	if ok == false {
		glog.Warning("onPutPlayer() not find clubID. clubID:=", msgBody.ClubID)
		return
	}

	clubTable.PutPlayer(&msgBody)
}

func onDeletePlayer(msg *mateProto.MessageMaTe) {
	msgBody := clubProto.SS_DelPlayerInTable{}
	err := json.Unmarshal(msg.Data, &msgBody)
	if err != nil {
		glog.Warning("onDeletePlayer() error :=", err.Error())
		return
	}

	clubTable, ok := clubMap[msgBody.ClubID]
	if ok == false {
		glog.Warning("onDeletePlayer() not find clubID. clubID:=", msgBody.ClubID)
		return
	}

	clubTable.DeletePlayer(&msgBody)
}

func onTablePlaying(msg *mateProto.MessageMaTe) {
	msgBody := clubProto.SS_TableStatusChanged{}
	err := json.Unmarshal(msg.Data, &msgBody)
	if err != nil {
		glog.Warning("onTablePlaying() error :=", err.Error())
		return
	}

	clubTable, ok := clubMap[msgBody.ClubID]
	if ok == false {
		glog.Warning("onTablePlaying() not find clubID. clubID:=", msgBody.ClubID)
		return
	}

	clubTable.TableStatusChanged(&msgBody)
}

func onDeleteTable(msg *mateProto.MessageMaTe) {
	msgBody := clubProto.SS_DelTable{}
	err := json.Unmarshal(msg.Data, &msgBody)
	if err != nil {
		glog.Warning("onDeleteTable() error :=", err.Error())
		return
	}

	clubTable, ok := clubMap[msgBody.ClubID]
	if ok == false {
		glog.Warning("onDeleteTable() not find clubID. clubID:=", msgBody.ClubID)
		return
	}
	clubTable.DeleteTable(&msgBody)
}

func onGetTable(msg *mateProto.MessageMaTe) {

	msgBody := msg.MsgBody.(*clubProto.CS_GetTable)

	tableData := clubProto.SC_GetTable{VersionNum: msgBody.CurVersionNum, ClubVersionNumber: msgBody.ClubVersionNumber,
		BeginIndex: msgBody.BeginIndex}

	rspMsg := mateProto.MessageMaTe{Source: msg.Source, MessageID: msg.MessageID, SenderID: msg.SenderID}
	rspBody := mateProto.JsonResponse{Data: &tableData}

	clubTable, ok := clubMap[msgBody.ClubID]
	if ok == false {
		wrapMQ.SendToSource(&rspMsg, &rspBody)
		return
	}
	//tableData.WaitingTable = len(clubTable.waitingTableMap)
	//tableData.PlayingTable = len(clubTable.allTableMap) - tableData.WaitingTable

	f := func(arr []string) {
		if msg.MessageID == clubProto.ID_TableGet {
			if msgBody.BeginIndex < len(arr) {
				s := msgBody.StopIndex
				if s > len(arr) {
					s = len(arr)
				}
				tableData.Tables = arr[msgBody.BeginIndex:s]
				return
			}
		} else if msg.MessageID == clubProto.ID_PerSeconGetTables {
			if msgBody.StopIndex < len(arr) {
				tableData.Tables = arr[msgBody.BeginIndex:msgBody.StopIndex]
				return
			}
			t := msgBody.StopIndex - msgBody.BeginIndex
			if t < 1 || t > 100 {
				glog.Warning(t)
				return
			}

			if t < len(arr) {
				tableData.Tables = arr[len(arr)-t:]
				tableData.BeginIndex = len(arr) - t
			} else {
				tableData.Tables = arr
				tableData.BeginIndex = 0
			}
		}
	}

	if msgBody.ClubPlayID != 0 {
		clubTablesSet, ok := clubTable.clubPlayMap[msgBody.ClubPlayID]
		if ok == false {
			wrapMQ.SendToSource(&rspMsg, &rspBody)
			return
		}
		tableData.TableCount = len(clubTablesSet.allTable)
		f(clubTablesSet.allTable)
	} else if msgBody.GameID > 0 {
		clubTablesSet, ok := clubTable.gameTableMap[msgBody.GameID]
		if ok == false {
			wrapMQ.SendToSource(&rspMsg, &rspBody)
			return
		}
		tableData.TableCount = len(clubTablesSet.allTable)
		f(clubTablesSet.allTable)
	} else if msgBody.GameID == -1 { // 拿取所有
		tableData.TableCount = len(clubTable.allTable)
		f(clubTable.allTable)
	}

	wrapMQ.SendToSource(&rspMsg, &rspBody)
}

func onDeleteServiceIDTable(msg *mateProto.MessageMaTe) {

	msgBody := clubProto.SS_DeleteServiceIDTable{}
	err := json.Unmarshal(msg.Data, &msgBody)
	if err != nil {
		glog.Warning("onDeleteServiceIDTable(). error :=", err.Error())
		return
	}

	msgDelTable := clubProto.SS_DelTable{}
	for keyClubID, v := range clubMap {
		for _, t := range v.allTableMap {
			if t.ServiceID == msgBody.ServiceID {
				msgDelTable.ClubID = keyClubID
				msgDelTable.GameID = t.GameID
				msgDelTable.TableNumber = t.TableNum
				v.DeleteTable(&msgDelTable)
			}
		}
	}
}

func onPushServiceIDAllTable(msg *mateProto.MessageMaTe) {

	msgBody := clubProto.SS_PushAllTable{}
	err := json.Unmarshal(msg.Data, &msgBody)
	if err != nil {
		glog.Warning("onDeleteServiceIDTable() error :=", err.Error())
		return
	}

	for _, v := range msgBody.TableArr {
		onRecoverTable(&v)
	}
}

func onQuickStart(msg *mateProto.MessageMaTe) {
	msgBody := msg.MsgBody.(*clubProto.CS_ClubQuickStart)

	reply := clubProto.SS_ReplyQuickStart{}

	defer func() {
		reply.QuickStartData = msgBody
		msg.MessageID = clubProto.ID_ReplyQuickStart
		msg.MsgBody = &reply

		postEventToClub.PostMaTeEvent(msg)
	}()

	clubTables, ok := clubMap[msgBody.MZClubID]
	if ok == false {
		reply.TableNum = 0
		//reply.TableNum = clubProto.ErrClubNotExist
		return
	}

	// 可进入在玩
	if msgBody.GameID == 107 || msgBody.GameID == 103 {
		cpTables, ok := clubTables.clubPlayMap[msgBody.ClubPlayID]
		if ok == false {
			reply.TableNum = 0
			return
		}
		for k, v := range cpTables.tableMap {
			if v.OnlinePlayers >= v.MaxPlayers {
				continue
			}
			if msgBody.RobotMaxPlayers > 0 && v.OnlinePlayers >= msgBody.RobotMaxPlayers {
				continue
			}

			if v.CurChangeCategory&playingCC == playingCC {
				reply.TableNum = k
				return
			}
		}
	}

	tableMap, ok1 := clubTables.waitingTableMap[msgBody.ClubPlayID]
	if ok1 == false {
		reply.TableNum = 0
		//reply.TableNum = clubProto.ErrClubNotFindPlayID
		return
	}
	for k, _ := range tableMap {
		reply.TableNum = k
		return
	}

}

func onDeleteClubPlay(msg *mateProto.MessageMaTe) {
	param := msg.MsgBody.(*clubProto.CS_DeleteClubPlay)

	clubTable, ok := clubMap[param.ClubID]
	if ok == false {
		return
	}

	tabMap, ok := clubTable.clubPlayMap[param.ClubPlayID]
	if ok == false {
		return
	}
	tempMsg := mateProto.MessageMaTe{}
	tempBody := clubProto.SS_DelTable{ClubID: param.ClubID, GameID: param.GameID}
	for _, v := range tabMap.tableMap {
		tempBody.TableNumber = v.TableNum
		tempMsg.Data, _ = json.Marshal(&tempBody)
		onDeleteTable(&tempMsg)
	}
}

func onGetClubTableCount(msg *mateProto.MessageMaTe) {
	msgBody := clubProto.SS_GetClubTableCount{}
	err := json.Unmarshal(msg.Data, &msgBody)
	if err != nil {
		glog.Warning("onGetClubTableCount() error :=", err.Error())
		return
	}

	v, ok := clubMap[msgBody.ClubID]
	if ok == false {
		return
	}
	rsp := clubProto.SSRSP_GetClubTableCount{ClubID: msgBody.ClubID}
	rsp.Arr = make([]clubProto.ClubPlayTableCount, 0, 30)
	for pID, v1 := range v.clubPlayMap {
		rsp.Arr = append(rsp.Arr, clubProto.ClubPlayTableCount{pID, len(v1.tableMap)})
	}

	wrapMQ.SendToSource(msg, &rsp)
}

func onGetPlayerClubInfo(msg *mateProto.MessageMaTe) {
	rsp := msg.MsgBody.(*mateProto.JsonResponse)
	rspBody := rsp.Data.(*clubProto.SC_GetPlayerClubInfo)

	for _, v := range rspBody.ClubInfo {
		clubTable, _ := clubMap[v.ClubID]
		if clubTable == nil {
			break
		}
		v.TableCount, v.PlayerCount = clubTable.tableCount, clubTable.playerCount
	}

	wrapMQ.SendToSource(msg, rsp)
}
