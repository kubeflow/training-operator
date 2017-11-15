import logging
from xml.etree import ElementTree

import six

from py import util

class TestCase(object):
  def __init__(self):
    self.class_name = None
    self.name = None
    self.time = None
    # String describing the failure.
    self.failure = None


def create_junit_xml_file(test_cases, output_path, gcs_client=None):
  """Create a JUnit XML file.

  Args:
    test_cases: List of test case objects.
    output_path: Path to write the XML
    gcs_client: GCS client to use if output is GCS.
  """
  total_time = 0
  failures = 0
  for c in test_cases:
    total_time += c.time

    if c.failure:
      failures += 1
  attrib = {"failures": "{0}".format(failures), "tests": "{0}".format(len(test_cases)),
            "time": "{0}".format(total_time)}
  root = ElementTree.Element("testsuite", attrib)

  for c in test_cases:
    attrib = {
      "classname": c.class_name,
      "name": c.name,
      "time": "{0}".format(c.time),
    }
    if c.failure:
      attrib["failure"] = c.failure
    e = ElementTree.Element("testcase", attrib)

    root.append(e)

  t = ElementTree.ElementTree(root)
  logging.info("Creationg %s", output_path)
  if output_path.startswith("gs://"):
    b = six.StringIO()
    t.write(b)

    bucket_name, path = util.split_gcs_uri(output_path)
    bucket = gcs_client.get_bucket(bucket_name)
    blob = bucket.blob(path)
    blob.upload_from_string(b.getvalue())
  else:
    t.write(output_path)
