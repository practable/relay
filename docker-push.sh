#!/bin/bash
#we strip the first char (usually a v from the tag), because we typically tag with command like $ git tag v0.1.0
tag=$(git describe --tags --abbrev=0)
dockertag="practable/relay:${tag:1}-alpine"

read -p "Will push with tag ${dockertag}, proceed? (y/n) " yn

case $yn in 
	[yY] ) echo ok, we will proceed;;
	[nN] ) echo exiting...;
		exit;;
	* ) echo invalid response;
		exit 1;;
esac

docker tag relay_relay:latest $dockertag
#echo "You probably need to do $docker login -u practable #enter password for account admin@practable.io at prompt"
docker push $dockertag
