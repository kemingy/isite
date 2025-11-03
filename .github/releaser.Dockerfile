FROM scratch
ARG TARGETPLATFORM
COPY $TARGETPLATFORM/isite /usr/bin/isite
ENTRYPOINT [ "/usr/bin/isite" ]
