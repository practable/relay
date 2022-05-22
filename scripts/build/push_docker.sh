#!/bin/bash
echo "images for v0.2.2 have alreay been pushed - update push script!"
exit
docker tag relay_book:latest practable/relay:book-0.2.2
docker tag relay_session:latest practable/relay:session-0.2.2
docker tag relay_shell:latest practable/relay:shell-0.2.2
echo "You probably need to do $docker login -u practable #enter password for account admin@practable.io at prompt"
docker push practable/relay:book-0.2.2
docker push practable/relay:session-0.2.2
docker push practable/relay:shell-0.2.2 
