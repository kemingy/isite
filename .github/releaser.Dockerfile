FROM scratch

COPY isite /usr/bin/isite

ENTRYPOINT [ "/usr/bin/isite" ]
