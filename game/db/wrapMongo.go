package db

import (
	"context"
	"fmt"
	"github.com/golang/glog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/bsonoptions"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"qpGame/commonDefine/mateProto"
	"reflect"
	"time"
)

var (
	mongoDBClient *mongo.Client
	databaseName  string
)

func ConnectMongoDB(mongoURL, dataName string) error {
	var (
		err error
	)
	databaseName = dataName

	builder := bsoncodec.NewRegistryBuilder()

	// 注册默认的编码和解码器
	bsoncodec.DefaultValueEncoders{}.RegisterDefaultEncoders(builder)
	bsoncodec.DefaultValueDecoders{}.RegisterDefaultDecoders(builder)

	// 注册时间解码器
	tTime := reflect.TypeOf(time.Time{})
	tCodec := bsoncodec.NewTimeCodec(bsonoptions.TimeCodec().SetUseLocalTimeZone(true))
	registry := builder.RegisterTypeDecoder(tTime, tCodec).Build()

	opt := options.Client().ApplyURI(mongoURL).SetMaxConnIdleTime(time.Hour).SetRegistry(registry)
	opt.SetMaxPoolSize(4) // default

	var ctx = context.Background()
	if mongoDBClient, err = mongo.Connect(ctx, opt); err != nil {
		return err
	}

	err = mongoDBClient.Ping(ctx, nil)
	return err
}

// 修改玩家的 钻石 数量
func ChangePlayerDiamondCount(uid int64, changeValue int32) (int32, error) {
	if changeValue == 0 {
		return 0, nil
	}

	coll := mongoDBClient.Database(databaseName).Collection("playerInfo")
	ctx := context.Background()

	update := bson.M{}
	filter := bson.M{"uid": uid}
	if changeValue < 0 {
		filter["basic_info.diamond"] = bson.M{"$gte": changeValue * -1}
	}
	update["$inc"] = bson.M{"basic_info.diamond": changeValue}

	updateRes, err := coll.UpdateOne(ctx, filter, update)
	if err != nil {
		glog.Warning("ChangePlayerDiamondCount() err. err :=", err.Error(), " ,uid:=", uid, ",value:=", changeValue)
		return 0, err
	}
	if updateRes.ModifiedCount < 1 {
		glog.Warning("ChangePlayerDiamondCount() err. uid:=", uid, ",value:=", changeValue)
		return mateProto.ErrDiamondNotEnough, fmt.Errorf("not update player")
	}

	if updateRes.MatchedCount < 1 {
		glog.Warning("ChangePlayerDiamondCount() err. uid:=", uid, ",value:=", changeValue)
		return 0, fmt.Errorf("not find player")
	}
	return 0, nil
}

func ChangePlayerRoomCardCount(uid int64, changeValue int32) (int32, error) {
	if changeValue == 0 {
		return 0, nil
	}

	coll := mongoDBClient.Database(databaseName).Collection("playerInfo")
	ctx := context.Background()

	update := bson.M{}
	filter := bson.M{"uid": uid}
	if changeValue < 0 {
		filter["basic_info.room_card"] = bson.M{"$gte": changeValue * -1}
	}
	update["$inc"] = bson.M{"basic_info.room_card": changeValue}

	updateRes, err := coll.UpdateOne(ctx, filter, update)
	if err != nil {
		glog.Warning("ChangePlayerRoomCardCount() err. uid:=", uid, ",value:=", changeValue)
		return 0, err
	}
	if updateRes.ModifiedCount < 1 {
		glog.Warning("ChangePlayerRoomCardCount() err. uid:=", uid, ",value:=", changeValue)
		return mateProto.ErrRoomCardNotEnough, fmt.Errorf("not update player")
	}
	if updateRes.MatchedCount < 1 {
		glog.Warning("ChangePlayerRoomCardCount() err. uid:=", uid, ",value:=", changeValue)
		return 0, fmt.Errorf("not find player")
	}

	return 0, nil
}

func ChangePlayerClubScore(clubID int32, uid, changeValue int64) {
	if changeValue == 0 {
		return
	}

	glog.Warning("ChangePlayerClubScore(). clubID:=", clubID, "  uid:=", uid, ",change:=", changeValue)

	coll := mongoDBClient.Database(databaseName).Collection("playerInfo")

	filter := bson.M{"uid": uid, "club_info": bson.M{"$elemMatch": bson.M{"clubID": clubID}}}
	update := bson.M{"$inc": bson.M{"club_info.$.score": changeValue}}
	opt := options.FindOneAndUpdateOptions{}
	opt.SetProjection(bson.M{"club_info": 1, "_id": 0})
	opt.SetReturnDocument(options.After)

	_, err := coll.UpdateOne(nil, filter, update)
	if err != nil {
		glog.Warning("ChangePlayerClubScore(). clubID:=", clubID, "  uid:=", uid, ",change:=", changeValue, ",err:=", err.Error())
		return
	}
}
