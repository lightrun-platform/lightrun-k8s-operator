#!/bin/sh
set -e

if [[ ${LIGHTRUN_KEY} == "" ]]; then
    echo "Missing LIGHTRUN_KEY env variable"
    exit 1
fi
if [[ ${PINNED_CERT} == "" ]]; then
    echo "Missing PINNED_CERT env variable"
    exit 1
fi
if [[ ${LIGHTRUN_SERVER} == "" ]]; then
    echo "Missing LIGHTRUN_SERVER env variable"
    exit 1
fi


echo "Merging configs"
awk -F'=' '{ if($1 in b) a[b[$1]]=$0;else{a[++i]=$0; b[$1]=i} }END{for(j=1;j<=i;j++) print a[j]}' /agent/agent.config /tmp/cm/agent.config > /tmp/tempconf
cp /tmp/tempconf /agent/agent.config
cp /tmp/cm/agent.metadata.json /agent/agent.metadata.json
rm  /tmp/tempconf
echo "Set server and secrets"
sed -i.bak "s|com.lightrun.server=.*|com.lightrun.server=https://$LIGHTRUN_SERVER|" /agent/agent.config && rm /agent/agent.config.bak
sed -i.bak "s|com.lightrun.secret=.*|com.lightrun.secret=$LIGHTRUN_KEY|" /agent/agent.config && rm /agent/agent.config.bak
sed -i.bak "s|pinned_certs=.*|pinned_certs=$PINNED_CERT|" /agent/agent.config && rm /agent/agent.config.bak
mv /agent /tmp/agent
echo "Finished"
