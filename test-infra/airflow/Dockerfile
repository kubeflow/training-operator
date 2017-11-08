FROM puckel/docker-airflow
COPY dags/ /usr/local/airflow/dags
COPY airflow.cfg /usr/local/airflow/
RUN mkdir -p /usr/local/airflow/logs
COPY entrypoint.sh /entrypoint.sh
USER root
RUN chmod a+x /entrypoint.sh
USER airflow
ENTRYPOINT ["/entrypoint.sh"]