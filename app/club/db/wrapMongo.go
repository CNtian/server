package db

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"vvService/dbCollectionDefine/playerInfo"
)

var (
	mongoDBClient *mongo.Client
	databaseName  string
)

func SetMongo(client *mongo.Client, database string) {
	mongoDBClient = client
	databaseName = database
}

// 获取玩家俱乐部信息
func GetPlayerClubInfo(id int64) (*collPlayer.PlayerInfo, error) {

	playerInfo := collPlayer.PlayerInfo{}

	coll := mongoDBClient.Database(databaseName).Collection(collPlayer.CollPlayerInfo)
	ctx := context.Background()

	findOpt := options.FindOneOptions{}
	findOpt.SetProjection(bson.M{"club_info": 1, "is_create_club": 1})
	err := coll.FindOne(ctx, bson.M{"uid": id}, &findOpt).Decode(&playerInfo)
	return &playerInfo, err
}

// 获取玩家俱乐部 分
func GetSomePlayerClubScore(uid []int64, playerMap *map[int64]collPlayer.PlayerInfo) error {

	col := mongoDBClient.Database(databaseName).Collection(collPlayer.CollPlayerInfo)
	ctx := context.Background()

	opt := options.FindOptions{}
	opt.SetProjection(bson.M{"club_info": 1, "uid": 1})
	cur, err := col.Find(ctx, bson.M{"uid": bson.M{"$in": uid}}, &opt)
	if err != nil {
		return err
	}
	defer cur.Close(ctx)

	for cur.Next(ctx) {
		temp := collPlayer.PlayerInfo{}
		err = cur.Decode(&temp)
		if err != nil {
			continue
		}
		(*playerMap)[temp.UID] = temp
	}

	return nil
}

// 获取玩家俱乐部 分
func GetPlayerClubScore(clubID int32, uid int64) (int64, error) {

	var playerInfo collPlayer.PlayerInfo

	col := mongoDBClient.Database(databaseName).Collection(collPlayer.CollPlayerInfo)
	ctx := context.Background()

	opt := options.FindOneOptions{}
	opt.SetProjection(bson.M{"club_info": 1})
	err := col.FindOne(ctx, bson.M{"uid": uid, "club_info.clubID": clubID}, &opt).Decode(&playerInfo)
	if err != nil {
		return 0, err
	}
	for _, v := range playerInfo.ClubData {
		if v.ClubID == clubID {
			return v.Score, nil
		}
	}
	return 0, fmt.Errorf("not match club")
}

// 获取玩家俱乐部 分
func GetPlayerClub(uid int64) ([]*collPlayer.ClubInfo, error) {

	var playerInfo collPlayer.PlayerInfo

	col := mongoDBClient.Database(databaseName).Collection(collPlayer.CollPlayerInfo)
	ctx := context.Background()

	opt := options.FindOneOptions{}
	opt.SetProjection(bson.M{"club_info": 1})
	err := col.FindOne(ctx, bson.M{"uid": uid}, &opt).Decode(&playerInfo)
	if err != nil {
		return nil, err
	}

	return playerInfo.ClubData, nil
}

func IsAutoAgreeEmail(uid int64) (bool, error) {

	var playerInfo collPlayer.PlayerInfo

	col := mongoDBClient.Database(databaseName).Collection(collPlayer.CollPlayerInfo)
	ctx := context.Background()

	opt := options.FindOneOptions{}
	opt.SetProjection(bson.M{"aaEmail": 1})
	err := col.FindOne(ctx, bson.M{"uid": uid}, &opt).Decode(&playerInfo)
	if err != nil {
		return false, err
	}

	return playerInfo.AutoAgreeEmail == 0, nil
}

func GetPlayerProperty(uid int64) (int64, int64, error) {

	var playerInfo collPlayer.PlayerInfo

	col := mongoDBClient.Database(databaseName).Collection(collPlayer.CollPlayerInfo)
	ctx := context.Background()

	opt := options.FindOneOptions{}
	opt.SetProjection(bson.M{"basic_info": 1})
	err := col.FindOne(ctx, bson.M{"uid": uid}, &opt).Decode(&playerInfo)
	if err != nil {
		return 0, 0, err
	}

	return playerInfo.BasicInfo.RoomCardCount, playerInfo.BasicInfo.DiamondCount, nil
}

func CheckTopProxy(uid int64) error {
	col := mongoDBClient.Database(databaseName).Collection(collPlayer.CollPlayerInfo)

	c, err := col.CountDocuments(nil, bson.M{"uid": uid, "top_proxy": true})
	if err != nil {
		return err
	}
	if c > 0 {
		return nil
	}
	return mongo.ErrNoDocuments
}

// 钻石
func UpdatePlayerDiamond(uid int64, count int32) error {
	coll := mongoDBClient.Database(databaseName).Collection(collPlayer.CollPlayerInfo)
	ctx := context.Background()

	var (
		result *mongo.UpdateResult
		err    error
	)

	if count < 0 {
		result, err = coll.UpdateOne(ctx,
			bson.M{"uid": uid, "basic_info.diamond": bson.M{"$gte": -count}},
			bson.M{"$inc": bson.M{"basic_info.diamond": count}})
	} else if count > 0 {
		result, err = coll.UpdateOne(ctx,
			bson.M{"uid": uid},
			bson.M{"$inc": bson.M{"basic_info.diamond": count}})
	}

	if err != nil {
		return err
	}
	if result != nil && result.ModifiedCount < 1 {
		return fmt.Errorf("UpdatePlayerDiamond() ModifiedCount<1")
	}

	return nil
}

func GetTestPlayerID(min, max int) ([]int64, error) {
	coll := mongoDBClient.Database(databaseName).Collection(collPlayer.CollPlayerInfo)
	ctx := context.Background()

	cur, err := coll.Find(ctx, bson.M{"uid": bson.M{"$gte": min, "$lte": max}})
	if err != nil {
		return nil, err
	}

	arr := make([]collPlayer.PlayerInfo, 0)
	err = cur.All(ctx, &arr)
	if err != nil {
		return nil, err
	}

	IDArr := make([]int64, len(arr))
	for i, _ := range arr {
		IDArr[i] = arr[i].UID
	}
	return IDArr, nil
}
