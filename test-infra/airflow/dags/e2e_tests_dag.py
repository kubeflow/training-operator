"""
Code that goes along with the Airflow located at:
http://airflow.readthedocs.org/en/latest/tutorial.html
"""
from airflow import DAG
from airflow.operators import PythonOperator
from datetime import datetime, timedelta


default_args = {
    'owner': 'airflow',
    'depends_on_past': False,
    'start_date': datetime(2015, 6, 1),
    'email': ['airflow@airflow.com'],
    'email_on_failure': False,
    'email_on_retry': False,
    'retries': 1,
    'retry_delay': timedelta(minutes=5),
    # 'queue': 'bash_queue',
    # 'pool': 'backfill',
    # 'priority_weight': 10,
    # 'end_date': datetime(2016, 1, 1),
}

dag = DAG(
    # Set schedule_interval to None
    'tf_k8s_tests', default_args=default_args, schedule_interval=None)

def build_images():
    print("Build the docker images")

def setup_cluster():
    print("setup cluster")

def create_cluster():
    print("Create cluster")

def deploy_crd():
    print("Deploy crd")

def run_tests():
    print("Run tests")

def delete_cluster():
    print("Delete cluster")


build_op = PythonOperator(
    task_id='build_images',
    provide_context=True,
    python_callable=build_images,
    dag=dag)

create_cluster_op = PythonOperator(
    task_id='create_cluster',
    provide_context=True,
    python_callable=create_cluster,
    dag=dag)

setup_cluster_op = PythonOperator(
    task_id='setup_cluster',
    provide_context=True,
    python_callable=setup_cluster,
    dag=dag)

setup_cluster_op.set_upstream(create_cluster_op)


deploy_crd_op = PythonOperator(
    task_id='deploy_crd',
    provide_context=True,
    python_callable=deploy_crd,
    dag=dag)

deploy_crd_op.set_upstream([setup_cluster_op, build_op])

run_tests_op = PythonOperator(
    task_id='run_tests',
    provide_context=True,
    python_callable=run_tests,
    dag=dag)

run_tests_op.set_upstream(deploy_crd_op)

delete_cluster_op = PythonOperator(
    task_id='delete_cluster',
    provide_context=True,
    python_callable=run_tests,
    dag=dag)


delete_cluster_op.set_upstream(run_tests_op)
