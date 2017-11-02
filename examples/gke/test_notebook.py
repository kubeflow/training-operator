import collections
import datetime
import logging
import json
import nbconvert
import nbformat
from py import util
import os
import re
import runpy
import six
import tempfile
import unittest
import uuid

from googleapiclient import discovery, errors
from oauth2client.client import GoogleCredentials

SubTuple = collections.namedtuple("SubTuple", ("name", "value", "pattern"))

def replace_vars_once(lines, new_values):
  """Substitutite in the values of the variables.

  This function assumes there is a single substitution for each variable.
  The function will return as soon as all variables have matched once.

  Args:
    lines: List of lines comprising the notebook.
    new_values: Key value pairs containing values to set different variables to.
  """

  # List of unmatched variables.
  unmatched = []
  for k, v in new_values.iteritems():
    # Format the values appropriately
    new_value = v
    if isinstance(v, six.string_types):
      new_value="\"{0}\"".format(v)
    unmatched.append(SubTuple(k, new_value, re.compile("\s*{0}\s*=*".format(k))))

  for i, l in enumerate(lines):
    for mindex, p in enumerate(unmatched):
      # Check if this line matches
      if p.pattern.match(l):
        # Replace this line with this text
        lines[i] = "{0}={1}".format(p.name, p.value)

        # Delete this pattern from unmatched. This is safe because we abort
        # the loop over unmatched since exactly one pattern can match
        del unmatched[mindex]

    if not unmatched:
      # All patterns matched.
      return lines

  names = [i.name for i in unmatched]
  raise ValueError("No matches for variables {0}".format(",".join(names)))

def strip_appendix(lines):
  """Remove all code in the appendix."""

  p = re.compile("#\s*#*\s*Appendix")
  for i in range(len(lines) - 1, 0, -1):
    if p.match(lines[i], re.IGNORECASE):
      return lines[0:i]
  raise ValueError("Could not find Appendix")

def strip_unexecutable(lines):
  """Remove all code that we can't execute"""

  valid = []
  for l in lines:
    if l.startswith("get_ipython"):
      continue
    valid.append(l)
  return valid

def delete_cluster(gke, name, project, zone):
  """Delete the cluster.

  Args:
    gke: Client for GKE.
    name: Name of the cluster.
    project: Project that owns the cluster.
    zone: Zone where the cluster is running.
  """

  request = gke.projects().zones().clusters().delete(clusterId=name,
                                                     projectId=project,
                                                     zone=zone)

  try:
    response = request.execute()
    logging.info("Response %s", response)
    delete_op = wait_for_operation(gke, project, zone, response["name"])
    logging.info("Cluster deletion done.\n %s", delete_op)

  except errors.HttpError as e:
    logging.error("Exception occured deleting cluster: %s, status: %s",
                  e, e.resp["status"])

class TestNotebook(unittest.TestCase):
  def testNotebook(self):
    """Test the GKE notebook."""
    notebook_path = os.path.join(os.path.dirname(__file__), "TF on GKE.ipynb")
    with open(notebook_path) as hf:
      node = nbformat.read(hf, nbformat.NO_CONVERT)
    exporter = nbconvert.PythonExporter()
    raw, _ = nbconvert.export(exporter, node)

    credentials = GoogleCredentials.get_application_default()
    gke = discovery.build("container", "v1", credentials=credentials)

    lines = raw.splitlines()
    # Dictionary of variables to substitute into the notebook.
    now = datetime.datetime.now()
    project = "mlkube-testing"
    cluster = ("gke-nb-test-" + now.strftime("v%Y%m%d") + "-"
                       + uuid.uuid4().hex[0:4])
    zone = "us-east1-d"
    new_values = {
      "project": project,
      "cluster_name": cluster,
      "zone": zone,
      "registry": "gcr.io/mlkube-testing",
      "data_dir": "gs://mlkube-testing_temp/cifar10/data",
      "job_dirs": "gs://mlkube-testing_temp/cifar10/jobs",
      "num_steps": 10,
    }
    modified = replace_vars_once(lines, new_values)

    modified = strip_appendix(modified)

    modified = strip_unexecutable(modified)

    with tempfile.NamedTemporaryFile(suffix="notebook.py", prefix="tmpGke",
                                     mode="w", delete=False) as hf:
      code_path = hf.name
      hf.write("\n".join(modified))
    logging.info("Wrote notebook to: %s", code_path)

    try:
      runpy.run_path(code_path)
    finally:
      logging.info("Deleting cluster; project=%s, zone=%s, name=%s", project,
                  zone, name)
      util.delete_cluster(gke, name, project, zone)
      logging.info("Cluster deletion done.\n %s", create_op)

if __name__ == "__main__":
  logging.getLogger().setLevel(logging.INFO)
  unittest.main()
