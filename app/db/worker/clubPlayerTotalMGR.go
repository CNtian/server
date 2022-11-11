package worker

import (
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"vvService/appDB/db"
	"vvService/appDB/protoDefine"
	"vvService/dbCollectionDefine"
)

type clubPlayerTotalData map[int32]*dbCollectionDefine.DBClubPlayerTotal // key:俱乐部ID

var (
//RecDate int
//playerClubTotalMap = map[int64]clubPlayerTotalData{} // key:玩家ID
)

func updateClubPlayerTotalData(curDate int, playerData protoDefine.SortPlayerGameScore, mzCID int32) error {

	writeArr := make([]mongo.WriteModel, 0, len(playerData))

	for _, playerV := range playerData {
		//clubV, ok := playerClubTotalMap[playerV.UID]
		//if ok == false {
		//	clubV = make(clubPlayerTotalData)
		//	playerClubTotalMap[playerV.UID] = clubV
		//}
		//
		//totalV, ok1 := clubV[playerV.ClubID]
		//if ok1 == false {
		//	totalV = new(dbCollectionDefine.DBClubPlayerTotal)
		//	totalV.Date, totalV.PlayerClubID = curDate, playerV.ClubID
		//	//totalV.JiangLi = make([]*dbCollectionDefine.ClubJiangli, 0, 6)
		//	clubV[playerV.ClubID] = totalV
		//}
		//
		//totalV.GameScore += playerV.SScore
		//totalV.GameCount += 1
		//totalV.HaoKa += playerV.HaoKa
		//totalV.GongXian += playerV.GongXian
		//totalV.XiaoHaoScore += playerV.XiaoHao

		//for _, v := range totalV.JiangLi {
		//	jl, ok3 := playerV.JiangLiMap[v.ClubID]
		//	if ok3 == true {
		//		v.JiangLi += jl.JiangLi
		//		delete(playerV.JiangLiMap, v.ClubID)
		//	}
		//}
		//
		//for k, v := range playerV.JiangLiMap {
		//	totalV.JiangLi = append(totalV.JiangLi, &dbCollectionDefine.ClubJiangli{ClubID: k, JiangLi: v.JiangLi})
		//}
		updateM := bson.M{"$inc": bson.M{"game_score": playerV.SScore,
			"game_count":    1,
			"haoKa":         playerV.HaoKa,
			"gongXian":      playerV.GongXian,
			"xiaoHao_score": playerV.XiaoHao,
			"bao_di":        playerV.BaoDi,
		}}
		totalV, ok := _recPlayerToClubJL[fmt.Sprintf("%d_%d_%d", curDate, playerV.ClubID, playerV.UID)]
		if ok {
			updateM["$set"] = bson.M{"club_jl": totalV.jlItem, "game_total": totalV.GameTotal, "mzCID": mzCID}
		}

		dbWrite := mongo.NewUpdateOneModel()
		dbWrite.SetUpsert(true)
		dbWrite.SetFilter(bson.M{"date": curDate, "uid": playerV.UID, "player_clubID": playerV.ClubID})
		dbWrite.SetUpdate(updateM)
		writeArr = append(writeArr, dbWrite)
	}

	return db.UpdateClubPlayerTotal(curDate, writeArr)
}

/*
func deleteAllTotal() {
	playerClubTotalMap = make(map[int64]clubPlayerTotalData)
}

func ReadTodayClubPlayerTotal(date int) {

	err := db.ReadClubPlayerTotal(date, func(total *dbCollectionDefine.DBClubPlayerTotal) {
		clubV, ok := playerClubTotalMap[total.PlayerID]
		if ok == false {
			clubV = make(clubPlayerTotalData)
			playerClubTotalMap[total.PlayerID] = clubV
		}

		clubV[total.PlayerClubID] = total
	})
	if err != nil {
		glog.Warning("ReadTodayClubPlayerTotal() err.", err.Error())
	}
}
*/
