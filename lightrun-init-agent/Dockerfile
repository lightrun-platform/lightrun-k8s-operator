ARG base_image_tag=alpine-3.20.0-r1

FROM lightruncom/prod-base:${base_image_tag}
ARG FILE

COPY $FILE /tmp/$FILE

RUN unzip -o /tmp/$FILE -d /agent ;\
    rm -rf /tmp/$FILE && \
    # Erase default values
    sed -i.bak "s|com.lightrun.secret=.*|com.lightrun.secret=|" /agent/agent.config && rm /agent/agent.config.bak && \
    sed -i.bak "s|pinned_certs=.*|pinned_certs=|" /agent/agent.config && rm /agent/agent.config.bak && \
    # In openshift UID will be dynamic per project, hence procide permissions to root group (defualt in k8s)
    chgrp -R 0 /agent && \
    chmod -R g=u /agent && \
    # Set proper permissions for the agent directory
    chown -R 1000:1000 /agent && \
    chmod -R 750 /agent && \
    # Create secret directory with proper permissions
    mkdir -p /etc/lightrun/secret && \
    chown -R 1000:1000 /etc/lightrun/secret && \
    chmod -R 700 /etc/lightrun/secret

# Copy and set permissions for update_config.sh before switching user
COPY update_config.sh /update_config.sh
RUN chmod 750 /update_config.sh && \
    chown 1000:1000 /update_config.sh

USER 1000

CMD [ "/bin/sh", "/update_config.sh" ]
