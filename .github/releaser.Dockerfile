FROM scratch

COPY isort /usr/bin/isort

ENTRYPOINT [ "/usr/bin/isort" ]