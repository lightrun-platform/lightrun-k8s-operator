ARG base_image_tag=alpine-3.20.0

FROM lightruncom/prod-base:${base_image_tag}
ARG FILE 

COPY lightrun-init-agent/$FILE /tmp/$FILE

RUN unzip -o /tmp/$FILE -d /agent ;\
    rm -rf /tmp/$FILE && \
    # Erase default values
    sed -i.bak "s|com.lightrun.secret=.*|com.lightrun.secret=|" /agent/agent.config && rm /agent/agent.config.bak && \
    sed -i.bak "s|pinned_certs=.*|pinned_certs=|" /agent/agent.config && rm /agent/agent.config.bak && \
    # In openshift UID will be dynamic per project, hence procide permissions to root group (defualt in k8s)
    chgrp -R 0 /agent && \
    chmod -R g=u /agent

USER 1000
COPY lightrun-init-agent/update_config.sh /update_config.sh

CMD [ "/bin/sh", "/update_config.sh" ]
