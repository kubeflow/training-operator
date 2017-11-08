FROM puckel/docker-airflow
COPY dags/ /usr/local/airflow/dags
COPY airflow.cfg /usr/local/airflow/
RUN mkdir -p /usr/local/airflow/logs
COPY entrypoint.sh /entrypoint.sh
USER root
RUN chmod a+rx /entrypoint.sh && \
    chown airflow /entrypoint.sh && \
    chown -R airflow /usr/local/airflow
USER airflow
ENTRYPOINT ["/entrypoint.sh"]