#!/bin/bash

curDir=`pwd` #$(cd $(dirname $0); pwd)

echo "当前目录:"$curDir

dirCount=0

for dir_item in `ls` #注意此处这是两个反引号，表示运行系统命令
do
 if [ -d $curDir"/"$dir_item ] && [[ "$dir_item" =~ [0-9]_mz ]];then
	mz_dir=$curDir"/"$dir_item
	
	dirCount=$(($dirCount+1))
	
	bakPath=$mz_dir"/appClub.bak"
	`rm $bakPath`  #1. 删掉备份
	
	sourcePath=$mz_dir"/appClub"
	`mv $sourcePath $bakPath`  #2. 备份当前
	
	# 3. 备份当前时，类似删除当前的
	
	sourcePath=$curDir"/appClub"
	targetPath=$mz_dir"/"
	
	`cp $sourcePath $targetPath`  #4. 拷贝最新的
	#echo $sourcePath"  "$targetPath
	
	filePath=$targetPath"/appClub"
	`chmod 777 $filePath`  #5. 给予执行权限
	#echo $filePath
	
	#重启
	cd $targetPath
	#echo $targetPath
	./restart.sh appClub
	# echo $targetPath"restart.sh appClub"
	#`$targetPath"restart.sh appClub"`
	cd ..
 fi
 
done

echo $dirCount