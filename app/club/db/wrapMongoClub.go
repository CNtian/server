package db

import (
	"context"
	"fmt"
	"github.com/bwmarrin/snowflake"
	"github.com/golang/glog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"math/rand"
	"time"
	collClub "vvService/dbCollectionDefine/club"
	collConfig "vvService/dbCollectionDefine/config"
	collPlayer "vvService/dbCollectionDefine/playerInfo"
)

var snowflakeNode *snowflake.Node

func getClubID() (int32, error) {

	randomNumb := rand.Intn(100) + 1
	if randomNumb < 0 {
		glog.Warning("rand.Intn()  randomNumb", randomNumb)
		randomNumb = 89
	}

	coll := mongoDBClient.Database(databaseName).Collection(collConfig.CollConfig)
	ctx := context.Background()

	cfg := collConfig.NextValue{}
	opt := options.FindOneAndUpdateOptions{}
	opt.SetReturnDocument(options.After)
	err := coll.FindOneAndUpdate(ctx,
		bson.M{"name": collConfig.FidldName_NextValue},
		bson.M{"$inc": bson.M{"next_clubID": randomNumb}},
		&opt).Decode(&cfg)
	if err != nil {
		return 0, err
	}
	return cfg.NextClubID, err
}

//func NewClub(creatorID, topProxy int64, name, mzNotice string, level int, isKickOutLeague, isKickOutMember bool) (*collClub.DBClubData, error) {
func NewClub(creatorID, topProxy int64, name string, mzClubData *collClub.DBClubData) (*collClub.DBClubData, error) {

	clubID, err := getClubID()
	if err != nil {
		return nil, err
	}

	firstMember := collClub.DBClubMember{ID: creatorID, JoinClubTime: time.Now()}
	memberArr := []*collClub.DBClubMember{&firstMember}
	instanceClub := collClub.DBClubData{
		CreateTime:        time.Now(),
		CreatorID:         creatorID,
		ClubID:            clubID,
		Name:              name,
		PlayArr:           []*collClub.DBClubPlay{},
		MemberArr:         memberArr,
		DirectSupervisor:  collClub.DBClubMerge{ShowPercentage: 100, RealPercentage: 100},
		DirectSubordinate: []collClub.DBClubMerge{},
		IsOpen:            true,
		IsFrozen:          false,
		IsFreeExit:        true,
		Subordinates:      []int32{},
		PlayerMutex:       []*collClub.DBMemberMutexGroup{},
		//IsKickOutLeague:   isKickOutLeague,
		//IsKickOutMember:   isKickOutMember,
		//MZNotice:          mzNotice,
		//Level:             level,
		ProxyUp:   topProxy,
		BlackList: []collClub.DBBlacklistItem{},
	}
	if mzClubData != nil {
		instanceClub.IsKickOutLeague = false
		instanceClub.IsKickOutMember = false
		instanceClub.Level = mzClubData.Level
		instanceClub.MZNotice = mzClubData.MZNotice
		instanceClub.GongXianWay = mzClubData.GongXianWay
		instanceClub.IsShowRankList = mzClubData.IsShowRankList
		instanceClub.IsShowScoreWater = mzClubData.IsShowScoreWater
		instanceClub.IsShowBaoMingFee = mzClubData.IsShowBaoMingFee
		instanceClub.BiLiShowWay = mzClubData.BiLiShowWay
		instanceClub.MaxTZCount = mzClubData.MaxTZCount
	} else {
		instanceClub.IsKickOutLeague = true
		instanceClub.IsKickOutMember = true
		instanceClub.Level = 10
	}

	wcMajority := writeconcern.New(writeconcern.WMajority(), writeconcern.WTimeout(3*time.Second))
	wcMajorityCollectionOpts := options.Collection().SetWriteConcern(wcMajority)

	colClub := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo, wcMajorityCollectionOpts)
	colPlayerInfo := mongoDBClient.Database(databaseName).Collection(collPlayer.CollPlayerInfo, wcMajorityCollectionOpts)

	ctx := context.Background()

	var session mongo.Session
	session, err = mongoDBClient.StartSession()
	if err != nil {
		return nil, err
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(sctx mongo.SessionContext) (interface{}, error) {

		// 添加俱乐部
		_, err := colClub.InsertOne(sctx, instanceClub)
		if err != nil {
			return nil, err
		}

		// 更新个人俱乐部
		playerClub := collPlayer.ClubInfo{ClubID: clubID, Score: 0, LastPlayTime: time.Now()}
		filter := bson.M{"uid": creatorID}
		update := bson.M{"$push": bson.M{"club_info": playerClub}}

		var updateRes *mongo.UpdateResult
		updateRes, err = colPlayerInfo.UpdateOne(sctx, filter, update)
		if err != nil {
			return nil, err
		}
		if updateRes.ModifiedCount < 1 && updateRes.MatchedCount < 1 {
			return nil, fmt.Errorf("not match uid:=%d", creatorID)
		}
		if topProxy != 0 {
			filter = bson.M{"uid": topProxy}
			update = bson.M{"$push": bson.M{"proxy_club": clubID}}

			var updateRes *mongo.UpdateResult
			updateRes, err = colPlayerInfo.UpdateOne(sctx, filter, update)
			if err != nil {
				return nil, err
			}
			if updateRes.ModifiedCount < 1 && updateRes.MatchedCount < 1 {
				return nil, fmt.Errorf("not match uid:=%d", topProxy)
			}
		}
		return nil, nil
	})
	return &instanceClub, err
}

func RealDelClubData(playerID int64, clubID int32) {
	coll := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo)
	_, err := coll.DeleteOne(nil, bson.M{"club_id": clubID})
	if err != nil {
		glog.Warning(err.Error(), ",", clubID, playerID)
	}

	coll = mongoDBClient.Database(databaseName).Collection(collPlayer.CollPlayerInfo)
	_, err = coll.UpdateOne(nil,
		bson.M{"uid": playerID},
		bson.M{"$pull": bson.M{"club_info": bson.M{"clubID": clubID}}})

	if err != nil {
		glog.Warning(err.Error(), ",", clubID, playerID)
	}
}

func LoadClub(clubID int32) (*collClub.DBClubData, error) {

	//wcMajorityCollectionOpts := options.Collection().SetReadConcern(readconcern.Majority())

	coll := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo) //, wcMajorityCollectionOpts)

	clubData := collClub.DBClubData{}
	return &clubData, coll.FindOne(nil, bson.M{"club_id": clubID}).Decode(&clubData)
}

// 获取俱乐部总分
func GetClubCountScore(clubID int32) (int64, error) {
	coll := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo)
	ctx := context.Background()

	opt := options.FindOneOptions{}
	opt.SetProjection(bson.M{"score_count": 1})
	clubData := collClub.DBClubData{}
	return clubData.ClubScoreCount, coll.FindOne(ctx, bson.M{"club_id": clubID}).Decode(&clubData)
}

func GetXClubCountScore(clubIDArr []int32) (map[int32]int64, error) {
	coll := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo)
	ctx := context.Background()

	opt := options.FindOptions{}
	opt.SetProjection(bson.M{"score_count": 1, "club_id": 1})

	cur, err := coll.Find(ctx, bson.M{"club_id": bson.M{"$in": clubIDArr}}, &opt)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	map_ := make(map[int32]int64)
	for cur.Next(ctx) {
		temp_ := collClub.DBClubData{}
		err = cur.Decode(&temp_)
		if err != nil {
			glog.Warning(err.Error())
			continue
		}
		map_[temp_.ClubID] = temp_.ClubScoreCount
	}
	return map_, nil
}

// 获取俱乐部总分
func GetSomeClubScore(clubID []int32) (map[int32][]int64, error) {
	coll := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo)
	ctx := context.Background()

	opt := options.FindOptions{}
	opt.SetProjection(bson.M{"score_count": 1, "club_id": 1, "unusable_score": 1})
	cur, err := coll.Find(ctx, bson.M{"club_id": bson.M{"$in": clubID}}, &opt)
	if err != nil {
		return nil, err
	}

	scoreMap := make(map[int32][]int64)

	for cur.Next(ctx) {
		clubData := collClub.DBClubData{}
		err = cur.Decode(&clubData)
		if err != nil {
			glog.Warning(err.Error())
			continue
		}
		scoreMap[clubData.ClubID] = []int64{clubData.ClubScoreCount, clubData.UnusableScore}
	}

	return scoreMap, nil
}

func GetClubPlayPercent(mzClubID, clubID int32, playID int64, v *collClub.DBClubPlayPercentage) error {
	coll := mongoDBClient.Database(databaseName).Collection(collClub.CollClubPlayPercentage)
	rsp := coll.FindOne(nil, bson.D{{"mz_club_id", mzClubID}, {"play_id", playID}, {"club_id", clubID}})

	err := rsp.Decode(v)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil
		}
		return err
	}

	return nil
}

func GetClubAllPlayPercent(mzClubID, clubID int32, v *[]collClub.DBClubPlayPercentage) error {
	coll := mongoDBClient.Database(databaseName).Collection(collClub.CollClubPlayPercentage)
	ctx := context.Background()
	cur, err := coll.Find(ctx, bson.D{{"mz_club_id", mzClubID}, {"club_id", clubID}})
	if err != nil {
		return nil
	}
	defer cur.Close(ctx)

	return cur.All(ctx, v)
}

func GetSubClubPlayPercent(mzClubID int32, clubID []int32, playID int64, value *map[int32]*collClub.DBClubPlayPercentage) error {
	coll := mongoDBClient.Database(databaseName).Collection(collClub.CollClubPlayPercentage)
	ctx := context.Background()
	cur, err := coll.Find(ctx, bson.D{{"mz_club_id", mzClubID}, {"play_id", playID}, {"club_id", bson.M{"$in": clubID}}})
	if err != nil {
		return nil
	}
	defer cur.Close(ctx)

	for cur.Next(ctx) {
		t := collClub.DBClubPlayPercentage{}
		err := cur.Decode(&t)
		if err != nil {
			continue
		}
		(*value)[t.ClubID] = &t
	}
	return nil
}

func SetClubPlayPercent(mzClubID, clubID int32, playID int64, parentReal, value float64, isBaoDi bool) error {

	realValue := (parentReal * value) / 100

	update := bson.M{}
	if isBaoDi == true {
		update = bson.M{"$set": bson.M{"real_bd_per": realValue, "show_bd_per": value}}
	} else {
		update = bson.M{"$set": bson.M{"percent": realValue, "show_per": value}}
	}

	coll := mongoDBClient.Database(databaseName).Collection(collClub.CollClubPlayPercentage)

	opt := options.UpdateOptions{}
	opt.SetUpsert(true)
	_, err := coll.UpdateOne(nil,
		bson.D{{"mz_club_id", mzClubID}, {"play_id", playID}, {"club_id", clubID}},
		update, &opt)
	return err
}

func SetClubPercent(clubID int32, showPercent, parentRealPercent int32) error {

	realValue := (parentRealPercent * showPercent) / 100

	update := bson.M{"$set": bson.M{"d_superior.per": showPercent, "d_superior.real_per": realValue}}

	coll := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo)

	_, err := coll.UpdateOne(nil,
		bson.D{{"club_id", clubID}},
		update)
	return err
}

func SetSubClubPlayPercent(mzClubID int32, clubID []int32, playID int64, parentReal, value int32, isBaoDi bool) error {

	realValue := (parentReal * value) / 100

	update := bson.M{}
	if isBaoDi == true {
		update = bson.M{"$set": bson.M{"real_bd_per": realValue, "show_bd_per": value}}
	} else {
		update = bson.M{"$set": bson.M{"percent": realValue, "show_per": value}}
	}

	coll := mongoDBClient.Database(databaseName).Collection(collClub.CollClubPlayPercentage)

	opt := options.UpdateOptions{}
	opt.SetUpsert(true)
	_, err := coll.UpdateMany(nil,
		bson.D{{"mz_club_id", mzClubID}, {"play_id", playID}, {"club_id", bson.M{"$in": clubID}}},
		update, &opt)
	return err
}

func DelClubPlayPercent(mzClubID int32, playID int64) error {
	coll := mongoDBClient.Database(databaseName).Collection(collClub.CollClubPlayPercentage)

	_, err := coll.UpdateMany(nil,
		bson.D{{"mz_club_id", mzClubID}, {"play_id", playID}},
		bson.M{"$set": bson.M{"is_del": true, "del_time": time.Now()}})
	return err
}

func KickOutDelPercent(clubID int32) error {
	coll := mongoDBClient.Database(databaseName).Collection(collClub.CollClubPlayPercentage)

	_, err := coll.UpdateMany(nil,
		bson.D{{"clubID", clubID}},
		bson.M{"$set": bson.M{"is_del": true, "del_time": time.Now()}})
	return err
}

func UpdateClubLevel(clubID int32, level int32) error {
	cClubInfo := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo)

	ctx := context.Background()

	var mzClubData collClub.DBClubData
	err := cClubInfo.FindOne(ctx, bson.M{"club_id": clubID}).Decode(&mzClubData)
	if err != nil {
		return err
	}
	mzClubData.Subordinates = append(mzClubData.Subordinates, clubID)

	_, err = cClubInfo.UpdateMany(nil,
		bson.M{"club_id": bson.M{"$in": mzClubData.Subordinates}},
		bson.M{"$set": bson.M{"level": level}})
	return err
}
