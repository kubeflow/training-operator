import collections
import datetime
import logging
import os
import re
import runpy
import tempfile
import unittest
import uuid

import nbconvert
import nbformat
import six
from googleapiclient import discovery
from oauth2client.client import GoogleCredentials
from py import util

SubTuple = collections.namedtuple("SubTuple", ("name", "value", "pattern"))

def replace_vars(lines, new_values):
  """Substitutite in the values of the variables.

  This function assumes there is a single substitution for each variable.
  The function will return as soon as all variables have matched once.

  Args:
    lines: List of lines comprising the notebook.
    new_values: Key value pairs containing values to set different variables to.
  """

  # List of matches variables.
  matches = []
  matched = set()
  for k, v in new_values.iteritems():
    # Format the values appropriately
    new_value = v
    if isinstance(v, six.string_types):
      new_value = "\"{0}\"".format(v)
    matches.append(SubTuple(k, new_value,
                            re.compile(r"\s*{0}\s*=.*".format(k))))

  for i, l in enumerate(lines):
    for p in matches:
      # Check if this line matches
      if p.pattern.match(l):
        # Replace this line with this text
        # We need to preserve leading white space for indentation
        pieces = l.split("=", 1)
        lines[i] = "{0}={1}".format(pieces[0], p.value)
        matched.add(p.name)

  unmatched = []
  for k, _ in new_values.iteritems():
    if not k in matched:
      unmatched.append(k)
  if unmatched:
    raise ValueError("No matches for variables {0}".format(", ".join(unmatched)))

  return lines

def strip_appendix(lines):
  """Remove all code in the appendix."""

  p = re.compile(r"#\s*#*\s*Appendix")
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

class TestNotebook(unittest.TestCase):
  @staticmethod
  def run_test(project, zone, cluster, new_values):  # pylint: disable=too-many-locals
    # TODO(jeremy@lewi.us): Need to configure the notebook and test to build
    # using GCB.
    dirname = os.path.dirname(__file__)
    if not dirname:
      logging.info("__file__ doesn't apper to be absolute path.")
      dirname = os.getcwd()
    notebook_path = os.path.join(dirname, "TF on GKE.ipynb")
    logging.info("Reading notebook %s", notebook_path)
    if not os.path.exists(notebook_path):
      raise ValueError("%s does not exist" % notebook_path)

    with open(notebook_path) as hf:
      node = nbformat.read(hf, nbformat.NO_CONVERT)
    exporter = nbconvert.PythonExporter()
    raw, _ = nbconvert.export(exporter, node)

    credentials = GoogleCredentials.get_application_default()
    gke = discovery.build("container", "v1", credentials=credentials)

    lines = raw.splitlines()

    modified = replace_vars(lines, new_values)

    modified = strip_appendix(modified)

    modified = strip_unexecutable(modified)

    with tempfile.NamedTemporaryFile(suffix="notebook.py", prefix="tmpGke",
                                     mode="w", delete=False) as hf:
      code_path = hf.name
      hf.write("\n".join(modified))
    logging.info("Wrote notebook to: %s", code_path)

    with open(code_path) as hf:
      contents = hf.read()
      logging.info("Notebook contents:\n%s", contents)

    try:
      runpy.run_path(code_path)
    finally:
      logging.info("Deleting cluster; project=%s, zone=%s, name=%s", project,
                  zone, cluster)
      util.delete_cluster(gke, cluster, project, zone)

  def testCpu(self):
    """Test using CPU only."""
    now = datetime.datetime.now()
    project = "kubeflow-ci"
    cluster = ("gke-nb-test-" + now.strftime("v%Y%m%d") + "-"
                 + uuid.uuid4().hex[0:4])
    zone = "us-east1-d"
    new_values = {
      "project": project,
      "cluster_name": cluster,
      "zone": zone,
      "registry": "gcr.io/kubeflow-ci",
      "data_dir": "gs://kubeflow-ci_temp/cifar10/data",
      "job_dirs": "gs://kubeflow-ci_temp/cifar10/jobs",
      "num_steps": 10,
      "use_gpu": False,
    }
    self.run_test(project, zone, cluster, new_values)

  def testGpu(self):
    """Test using CPU only."""
    now = datetime.datetime.now()
    project = "kubeflow-ci"
    cluster = ("gke-nb-test-" + now.strftime("v%Y%m%d") + "-"
                 + uuid.uuid4().hex[0:4])
    zone = "us-east1-c"
    new_values = {
      "project": project,
      "cluster_name": cluster,
      "zone": zone,
      "registry": "gcr.io/kubeflow-ci",
      "data_dir": "gs://kubeflow-ci_temp/cifar10/data",
      "job_dirs": "gs://kubeflow-ci_temp/cifar10/jobs",
      "num_steps": 10,
      "use_gpu": True,
      "accelerator": "nvidia-tesla-k80",
      "accelerator_count": 1,
    }
    self.run_test(project, zone, cluster, new_values)

if __name__ == "__main__":
  logging.basicConfig(level=logging.INFO,
                      format=('%(levelname)s|%(asctime)s'
                              '|%(pathname)s|%(lineno)d| %(message)s'),
                      datefmt='%Y-%m-%dT%H:%M:%S',
                      )
  unittest.main()
