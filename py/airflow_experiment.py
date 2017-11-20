# DO NOT SUBMIT
# SOME experiments with airflow
from py import airflow
import google.auth
if __name__ == "__main__":
  state = "None"
  if state and not state == "running":
    print(state)
  PROW_K8S_MASTER = "35.202.163.166"
  base_url = "https://{0}/api/v1/proxy/namespaces/default/services/airflow:80".format(PROW_K8S_MASTER)

  credentials, _ = google.auth.default(
    scopes=["https://www.googleapis.com/auth/cloud-platform"])

  client = airflow.AirflowClient(base_url, credentials, verify=False)

  run_id = "2017-11-19T00:18:47"
  state = client.get_task_status(airflow.E2E_DAG, run_id, "undefined")
  print(state)