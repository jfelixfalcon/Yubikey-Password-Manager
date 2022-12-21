FROM postgres:alpine

COPY ./build/pg/config-files/ /run/postgresql/

RUN chown postgres /run/postgresql/*

CMD ["postgres", "-c", "config_file=/run/postgresql/postgresql.conf"]
