package db

import (
	"context"
	"fmt"
	"github.com/golang/glog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"strconv"
	"time"
	"vvService/dbCollectionDefine"
	collClub "vvService/dbCollectionDefine/club"
	collPlayer "vvService/dbCollectionDefine/playerInfo"
)

func GetProxy(uid int64) ([]int32, error) {

	var playerInfo collPlayer.PlayerInfo

	col := mongoDBClient.Database(databaseName).Collection(collPlayer.CollPlayerInfo)

	opt := options.FindOneOptions{}
	opt.SetProjection(bson.M{"proxy_club": 1})
	err := col.FindOne(nil, bson.M{"uid": uid}, &opt).Decode(&playerInfo)
	return playerInfo.ProxyClubArr, err
}

func CancelProxy(uid int64, clubIDArr []int32) error {
	col := mongoDBClient.Database(databaseName).Collection(collPlayer.CollPlayerInfo)

	_, err := col.UpdateMany(nil, bson.M{"uid": uid}, bson.M{"$pull": bson.M{"proxy_club": bson.M{"$in": clubIDArr}}})
	if err != nil {
		return err
	}

	collClubInfo := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo)

	_, err = collClubInfo.UpdateMany(nil, bson.M{"club_id": bson.M{"$in": clubIDArr}}, bson.M{"$set": bson.M{"proxy": 0}})
	if err != nil {
		return err
	}
	return nil
}

func AddProxy(uid int64, clubID int32) error {
	col := mongoDBClient.Database(databaseName).Collection(collPlayer.CollPlayerInfo)

	_, err := col.UpdateOne(nil, bson.M{"uid": uid}, bson.M{"$push": bson.M{"proxy_club": clubID}})
	if err != nil {
		return err
	}

	collClubInfo := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo)

	_, err = collClubInfo.UpdateMany(nil, bson.M{"club_id": clubID}, bson.M{"$set": bson.M{"proxy": uid}})
	if err != nil {
		return err
	}

	return nil
}

func GetMengZhuDaily(date int, clubID []int32, vMap *map[int32]*dbCollectionDefine.DBDailyMengZHuPlayer) error {
	coll_ := mongoDBClient.Database(databaseName).Collection(dbCollectionDefine.CollDailyMengZHuPlayer)

	ctx := context.Background()
	cur, err := coll_.Find(ctx, bson.M{"date": date, "mzClubID": bson.M{"$in": clubID}})
	if err != nil {
		glog.Warning("GetMengZhuDaily() err := ", err.Error())
		return err
	}
	defer cur.Close(ctx)

	for cur.Next(ctx) {
		t := dbCollectionDefine.DBDailyMengZHuPlayer{}
		err = cur.Decode(&t)
		if err != nil {
			glog.Warning(err.Error(), ", date:=", date, ",clubID:=", clubID)
			continue
		}
		(*vMap)[t.ClubID] = &t
	}

	return nil
}

func GetCurrentMengZhuDaily(clubID []int32, arr *[]dbCollectionDefine.DBDailyMengZHuPlayer) error {
	coll_ := mongoDBClient.Database(databaseName).Collection(dbCollectionDefine.CollDailyMengZHuPlayer)

	filter := bson.M{"mzClubID": bson.M{"$in": clubID}}

	fatcet := bson.M{"$group": bson.M{"_id": "$date",
		"gC": bson.M{"$sum": "$g_r_count"},
		"dC": bson.M{"$sum": "$daily_players"},
		"cC": bson.M{"$sum": "$consumables"}}}

	ctx := context.Background()
	cur, err := coll_.Aggregate(ctx, bson.A{bson.D{{"$match", filter}}, fatcet}) //bson.A{bson.M{"mzClubID": bson.M{"$in": clubID}}}, &opt)
	if err != nil {
		glog.Warning("GetCurrentMengZhuDaily() err := ", err.Error())
		return err
	}
	defer cur.Close(ctx)

	type TempAggregate struct {
		ID             int   `bson:"_id"`
		GameRoundCount int32 `bson:"gC"`
		DailyPlayers   int   `bson:"dC"`
		Consumables    int32 `bson:"cC"`
	}

	t := dbCollectionDefine.DBDailyMengZHuPlayer{}
	for cur.Next(ctx) {
		v := TempAggregate{}
		err = cur.Decode(&v)
		if err != nil {
			return err
		}
		t.Date = v.ID
		t.GameRoundCount = v.GameRoundCount
		t.Consumables = v.Consumables
		t.DailyPlayers = v.DailyPlayers

		*arr = append(*arr, t)
	}

	return nil
}

func GiveRoomCard(operatorID, toUID int64, changeValue int32) error {

	wcMajority := writeconcern.New(writeconcern.WMajority(), writeconcern.WTimeout(3*time.Second))
	wcMajorityCollectionOpts := options.Collection().SetWriteConcern(wcMajority)

	collDealLog := mongoDBClient.Database(databaseName).Collection(collClub.CollRoomCardDealLog, wcMajorityCollectionOpts)
	collPlayerInfo := mongoDBClient.Database(databaseName).Collection(collPlayer.CollPlayerInfo, wcMajorityCollectionOpts)

	ctx := context.Background()

	session, outSideErr := mongoDBClient.StartSession()
	if outSideErr != nil {
		return outSideErr
	}
	defer session.EndSession(ctx)

	_, outSideErr = session.WithTransaction(ctx, func(sctx mongo.SessionContext) (interface{}, error) {

		dealLog := collClub.DBRoomCardDealLog{
			CreateTime: time.Now(),
			SrcUID:     operatorID,
			ToPlayerID: toUID,
			Value:      changeValue,
		}

		opt := options.FindOneAndUpdateOptions{}
		opt.SetProjection(bson.M{"basic_info": 1})
		opt.SetReturnDocument(options.After)
		update := bson.M{"$inc": bson.M{"basic_info.room_card": -changeValue}}
		sR := collPlayerInfo.FindOneAndUpdate(sctx,
			bson.M{"uid": operatorID, "basic_info.room_card": bson.M{"$gte": changeValue}}, update, &opt)
		if sR.Err() != nil {
			return nil, sR.Err()
		}
		p := collPlayer.PlayerInfo{}
		sE := sR.Decode(&p)
		if sE != nil {
			return nil, sE
		}
		dealLog.SrcCurrentValue = p.BasicInfo.RoomCardCount
		dealLog.SrcPlayerName = p.BasicInfo.Nick

		update = bson.M{"$inc": bson.M{"basic_info.room_card": changeValue}}
		sR = collPlayerInfo.FindOneAndUpdate(sctx,
			bson.M{"uid": toUID}, update, &opt)
		if sR.Err() != nil {
			return nil, sR.Err()
		}
		p = collPlayer.PlayerInfo{}
		sE = sR.Decode(&p)
		if sE != nil {
			return nil, sE
		}
		dealLog.ToPlayerName = p.BasicInfo.Nick
		_, sE = collDealLog.InsertOne(sctx, &dealLog)
		if sE != nil {
			return nil, sE
		}
		return nil, nil
	})
	return outSideErr
}

func GiveRoomCardList(uid int64) ([]collClub.DBRoomCardDealLog, error) {
	collDealLog := mongoDBClient.Database(databaseName).Collection(collClub.CollRoomCardDealLog)

	ctx := context.Background()
	opt := options.FindOptions{}
	opt.SetLimit(100)
	opt.SetSort(bson.M{"create_time": -1})
	cur, err := collDealLog.Find(ctx, bson.M{"src_uid": uid}, &opt)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	arr := []collClub.DBRoomCardDealLog{}
	err = cur.All(ctx, &arr)
	return arr, err
}

func PutMengZhuActivity(v *collClub.DBClubActivity) error {
	_coll := mongoDBClient.Database(databaseName).Collection(collClub.CollMZActivity)

	opt := options.UpdateOptions{}
	opt.SetUpsert(true)
	_, err := _coll.UpdateOne(nil, bson.M{"club_id": v.ClubID}, bson.M{"$set": bson.M{"rule": v.Rule}}, &opt)
	return err
}

//func DelMengZhuActivity(clubID int32) error {
//	_coll := mongoDBClient.Database(databaseName).Collection(collClub.CollMZActivity)
//
//	_, err := _coll.DeleteOne(nil, bson.M{"club_id": clubID})
//	return err
//}

func GetMengZhuActivity(clubID int32, v *collClub.DBClubActivity) error {
	_coll := mongoDBClient.Database(databaseName).Collection(collClub.CollMZActivity)

	s := _coll.FindOne(nil, bson.M{"club_id": clubID})
	if s.Err() != nil {
		return s.Err()
	}
	return s.Decode(v)
}

func GetMengZhuAllPlayers(clubID int32) (int, error) {

	collClubInfo := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo)

	tempOtp := options.FindOneOptions{}
	tempOtp.SetProjection(bson.M{"subordinates": 1})

	s := collClubInfo.FindOne(nil, bson.M{"club_id": clubID}, &tempOtp)
	if s.Err() != nil {
		return 0, s.Err()
	}
	tempClub := collClub.DBClubData{}
	err := s.Decode(&tempClub)
	if err != nil {
		return 0, err
	}

	tempClub.Subordinates = append(tempClub.Subordinates, clubID)
	ctx := context.Background()
	opt := options.FindOptions{}
	opt.SetProjection(bson.M{"count": bson.M{"$size": "$members"}, "_id": 0})
	cur, err := collClubInfo.Find(ctx, bson.M{"club_id": bson.M{"$in": tempClub.Subordinates}}, &opt)
	if err != nil {
		return 0, err
	}
	defer cur.Close(ctx)

	allPlayers := int32(0)
	for cur.Next(ctx) {
		v, err := cur.Current.Values()
		if err != nil {
			glog.Warning("Decode Hello .", err.Error())
			continue
		}
		if len(v) < 1 {
			glog.Warning("Decode Hello .", len(v))
			continue
		}
		allPlayers += v[0].AsInt32()
	}

	return int(allPlayers), nil
}

func PutBlackList(clubID int32, item []*collClub.DBBlacklistItem) error {
	_coll := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo)

	_, err := _coll.UpdateOne(nil, bson.M{"club_id": clubID}, bson.M{"$set": bson.M{"blackList": item}})
	return err
}

func GetBlackList(clubID int32, item *[]collClub.DBBlacklistItem) error {
	_coll := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo)

	opt := options.FindOneOptions{}
	opt.SetProjection(bson.M{"blackList": 1})
	s := _coll.FindOne(nil, bson.M{"club_id": clubID}, &opt)
	if s.Err() != nil {
		return s.Err()
	}

	_t := collClub.DBClubData{}
	err := s.Decode(&_t)
	if err != nil {
		return err
	}
	*item = _t.BlackList

	return err
}

func DeleteClubScoreLog(clubID int32) {

	tt := time.Now().Unix()
	for i := int64(7); i < 10; i++ {
		year, month, day := time.Unix(tt-60*60*24*i, 0).Date()
		date__, _ := strconv.Atoi(fmt.Sprintf("%d%02d%02d", year, month, day))
		_name := fmt.Sprintf("%s_%d_%d", collClub.CollClubScoreLog, clubID, date__)
		_coll := mongoDBClient.Database(databaseName).Collection(_name)
		err := _coll.Drop(nil)
		if err != nil {
			glog.Warning("DeleteClubScoreLog.", _name, ",err:=", err.Error())
		}

		_name = fmt.Sprintf("index_%s", _name)
		_coll.Indexes().DropOne(nil, _name)
	}
}
