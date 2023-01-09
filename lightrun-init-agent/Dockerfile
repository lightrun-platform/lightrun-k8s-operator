FROM --platform=linux/amd64 alpine:latest

ARG GITHUB_TOKEN
ARG VERSION 
ARG FILE 

RUN apk --no-cache add jq \
                       wget
RUN export REPO="lightrun-platform/athena"; \
    # curl inside alpine can't properly forward cookies when downloading asset, hence wget
    wget -q --auth-no-challenge --header='Accept:application/octet-stream' \
    https://$GITHUB_TOKEN:@api.github.com/repos/$REPO/releases/assets/`wget -q -O- --auth-no-challenge --header "Accept: application/vnd.github+json"  https://$GITHUB_TOKEN:@api.github.com/repos/$REPO/releases | jq ". | map(select(.tag_name == \"$VERSION\"))[0].assets | map(select(.name == \"$FILE\"))[0].id"` \
    -O /tmp/agent.zip; \
    mkdir /agent ;\
    unzip -o /tmp/agent.zip -d /agent ;\
    rm -rf /tmp/agent.zip && \
    # Erase default values
    sed -i.bak "s|com.lightrun.secret=.*|com.lightrun.secret=|" /agent/agent.config && rm /agent/agent.config.bak && \
    sed -i.bak "s|pinned_certs=.*|pinned_certs=|" /agent/agent.config && rm /agent/agent.config.bak && \
    # In openshift UID will be dynamic per project, hence chmo and not chown
    chmod -R 777 /agent 

USER 1000

COPY lightrun-init-agent/update_config.sh /update_config.sh

CMD [ "/bin/sh", "/update_config.sh" ]
