package worker

import (
	"vvService/commonPackge/mateProto"
)

func onBackFangKa(msg *mateProto.MessageMaTe) {
	//param := protoDefine.SS_BackFangKa{}
	//err := json.Unmarshal(msg.Data, &param)
	//if err != nil {
	//	glog.Warning("onBackFangKa() err:=", err.Error(), ",data:=", string(msg.Data))
	//	return
	//}
	//
	//ctx := context.Background()
	//coll := db.MongoClient.Database(db.CurDBName).Collection(collPlayer.CollPlayerInfo)
	//
	//filter := bson.M{"uid": param.UID}
	//update := bson.M{"$inc": bson.M{"basic_info.room_card": param.Count}}
	//
	//var res *mongo.UpdateResult
	//res, err = coll.UpdateOne(ctx, filter, update)
	//if err != nil {
	//	glog.Warning("onBackFangKa() err:=", err.Error(), ",data:=", string(msg.Data))
	//	return
	//}
	//
	//if res.ModifiedCount < 1 {
	//	glog.Warning("onBackFangKa() data:=", param)
	//	return
	//}
}

func onBackDiamond(msg *mateProto.MessageMaTe) {
	//param := protoDefine.SS_BackDiamond{}
	//err := json.Unmarshal(msg.Data, &param)
	//if err != nil {
	//	glog.Warning("onBackDiamond() err:=", err.Error(), ",data:=", string(msg.Data))
	//	return
	//}
	//
	//ctx := context.Background()
	//coll := db.MongoClient.Database(db.CurDBName).Collection(collPlayer.CollPlayerInfo)
	//
	//filter := bson.M{"uid": param.UID}
	//update := bson.M{"$inc": bson.M{"basic_info.diamond": param.Count}}
	//
	//var res *mongo.UpdateResult
	//res, err = coll.UpdateOne(ctx, filter, update)
	//if err != nil {
	//	glog.Warning("onBackDiamond() err:=", err.Error(), ",data:=", string(msg.Data))
	//	return
	//}
	//
	//if res.ModifiedCount < 1 {
	//	glog.Warning("onBackDiamond() data:=", param)
	//	return
	//}
}
