#/bin/bash

# serve.sh is a script to help with testing
# timdrysdale/bookjs. Actions:
#
# - generates 24 hr admin & user login tokens 
#   for use by bookjs during testing
#
# - generated credentials are NOT sensitive
#   in that they are time limited and cannot
#   be used for production servers anyway
#
# - tokens are echoed to console and put in
#     -  BOOKJS_ADMINTOKEN and
#     -  BOOKJS_USERTOKEN
#
# - starts a booking server using docker
#   and uploads ./manifest.yaml
#
# - starts a static server on port 6000
#   which serves the assets in ./assets
#
# - waits for the script to be cancelled,
#   ending the two server processses
#

# pad base64URL encoded to base64
# from https://gist.github.com/angelo-v/e0208a18d455e2e6ea3c40ad637aac53
paddit() {
  input=$1
  l=`echo -n $input | wc -c`
  while [ `expr $l % 4` -ne 0 ]
  do
    input="${input}="
    l=`echo -n $input | wc -c`
  done
  echo $input
}

# booking server configuration (see book.env for details)
export BOOK_FQDN=http://[::]:4000

# asset server settings
export ASSET_PORT=8008

# token common settings
export BOOKTOKEN_SECRET=somesecret
export BOOKTOKEN_AUDIENCE=$BOOK_FQDN
export BOOKTOKEN_LIFETIME=86400
export BOOKTOKEN_GROUPS="everyone controls3"

# generate admin token
export BOOKTOKEN_ADMIN=true
export BOOKJS_ADMINTOKEN=$(book token)
export BOOKUPLOAD_TOKEN=$BOOKJS_ADMINTOKEN
echo "Admin token:"
echo ${BOOKJS_ADMINTOKEN}

# read and split the token and do some base64URL translation
read h p s <<< $(echo $BOOKJS_ADMINTOKEN | tr [-_] [+/] | sed 's/\./ /g')

h=`paddit $h`
p=`paddit $p`
# assuming we have jq installed
echo $h | base64 -d | jq
echo $p | base64 -d | jq

# generate user token
export BOOKTOKEN_ADMIN=false
export USERTOKEN=$(book token)
export BOOKJS_USERTOKEN=$(book token)
echo "User token:"
echo ${BOOKJS_USERTOKEN}

# read and split the token and do some base64URL translation
read h p s <<< $(echo $BOOKJS_ADMINTOKEN | tr [-_] [+/] | sed 's/\./ /g')

h=`paddit $h`
p=`paddit $p`
# assuming we have jq installed
echo $h | base64 -d | jq
echo $p | base64 -d | jq

# manifest upload settings
export BOOKUPLOAD_SCHEME=http
export BOOKUPLOAD_HOST=[::]:4000

#poolstore reset settings
export BOOKRESET_HOST=$BOOKUPLOAD_HOST
export BOOKRESET_SCHEME=$BOOKUPLOAD_SCHEME

# storestatus settings
export BOOKSTATUS_HOST=$BOOKUPLOAD_HOST
export BOOKSTATUS_SCHEME=$BOOKUPLOAD_SCHEME
export BOOKSTATUS_TOKEN=$BOOKUPLOAD_TOKEN

set | grep BOOK

# start book server using docker
./book.sh > book.log 2>&1 &
export BOOK_PID=$!

#wait five seconds for server to start
sleep 1

#upload manifest
book upload ../../manifest/manifest.yaml


# start asset server
http-server -p $ASSET_PORT ./assets > asset.log 2>&1 &

export ASSET_PID=$!

echo "book server on port ${BOOK_PORT} logs to ./book.log"
echo "asset server on port ${ASSET_PORT} logs to ./asset.log"

echo "commands:"
echo "  a: tail of the assert server log"
echo "  b: tail of book server log [default]"
echo "  g: start insecure chrome"
echo "  l: Lock bookings"
echo "  n: uNlock bookings"
echo "  r: reset the poolstore (has confirm)"
echo "  s: get the status of the poolstore)"
echo "  u: re-upload manifest"
echo "  done: stop servers"

for (( ; ; ))
do
	read -p 'What next? [a/b/g/l/n/u/r/s/done]:' command

	echo $command

if [ "$command" = "done" ];
then
     echo -e "\nShutting down"
	 break
elif ([ "$command" = "b" ] || [ "$command" = "" ]);
then
	tail book.log
elif [ "$command" = "a" ];
then
	tail asset.log	
elif [ "$command" = "g" ];
then
	mkdir -p ../tmp/chrome-user
	google-chrome --disable-web-security --user-data-dir="../tmp/chrome-user" > chrome.log 2>&1 &
elif [ "$command" = "l" ];
then
	export BOOKTOKEN_ADMIN=true
    export BOOKSTATUS_TOKEN=$(book token)
	read -p 'Enter lock message:' message
	book setstatus lock "$message"
elif [ "$command" = "n" ];
then
	export BOOKTOKEN_ADMIN=true
    export BOOKSTATUS_TOKEN=$(book token)
	read -p 'Enter unlock message:' message
	book setstatus unlock "$message"
elif [ "$command" = "r" ];
then
	export BOOKTOKEN_ADMIN=true
    export BOOKRESET_TOKEN=$(book token)
    book reset
elif [ "$command" = "s" ];
then
	export BOOKTOKEN_ADMIN=true
    export BOOKSTATUS_TOKEN=$(book token)
	book getstatus
elif [ "$command" = "u" ];
then
	read -p "Definitely upload [y/N]?" confirm
	if ([ "$confirm" == "y" ] || [ "$confirm" == "Y" ]  || [ "$confirm" == "yes"  ] );
	then
		export BOOKTOKEN_ADMIN=true
		export BOOKUPLOAD_TOKEN=$(book token)
		book upload ../../manifest/manifest.yaml
	else
		echo "wise choice, aborting"
	fi
else	
     echo -e "\nUnknown command ${command}."
fi
done

kill -n 15 $BOOK_PID
kill -n 15 $ASSET_PID
