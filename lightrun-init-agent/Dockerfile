FROM --platform=linux/amd64 alpine:latest

ARG FILE 

RUN mkdir /agent
# Copy and extract
ADD lightrun-init-agent/$FILE /agent

# Erase default values
RUN sed -i.bak "s|com.lightrun.secret=.*|com.lightrun.secret=|" /agent/agent.config && rm /agent/agent.config.bak && \
    sed -i.bak "s|pinned_certs=.*|pinned_certs=|" /agent/agent.config && rm /agent/agent.config.bak && \
    # In openshift UID will be dynamic per project, hence chmod and not chown
    chmod -R 777 /agent 

USER 1000
COPY lightrun-init-agent/update_config.sh /update_config.sh

CMD [ "/bin/sh", "/update_config.sh" ]
